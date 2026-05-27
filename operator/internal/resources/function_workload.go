package resources

import (
	"log/slog"
	"strings"
	"unicode"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/natsadmin"
	"github.com/kou-etal/etalbaas/operator/internal/provider/gpu"
)

// DesiredFunctionDeployment builds the desired Deployment for a Function workload.
// Returns nil for heavy-job functions (they use Job templates, not Deployments).
func DesiredFunctionDeployment(fn *etalbaasv1alpha1.Function, imageRef string, cfg config.OperatorConfig) *appsv1.Deployment {
	if fn.Spec.Kind == etalbaasv1alpha1.FunctionKindHeavyJob {
		return nil
	}

	namespace := fn.Namespace
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	labels := functionLabels(projectID, funcName)
	selectorLabels := functionSelectorLabels(projectID, funcName)

	replicas := resolveReplicas(fn)
	podSpec := buildFunctionPodSpec(fn, imageRef, cfg)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "func-" + funcName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergeMaps(labels, selectorLabels),
				},
				Spec: podSpec,
			},
		},
	}
}

// buildFunctionPodSpec builds the PodSpec for a function workload.
func buildFunctionPodSpec(fn *etalbaasv1alpha1.Function, imageRef string, cfg config.OperatorConfig) corev1.PodSpec {
	container := corev1.Container{
		Name:  "function",
		Image: imageRef,
		Env:   resolveEnvVars(fn),
	}

	// Set resource requests/limits
	container.Resources = resolveResources(fn)

	// Security context: readOnlyRootFilesystem, runAsNonRoot, drop all capabilities
	trueVal := true
	var uid int64 = 65532
	container.SecurityContext = &corev1.SecurityContext{
		ReadOnlyRootFilesystem: &trueVal,
		RunAsNonRoot:           &trueVal,
		RunAsUser:              &uid,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		AllowPrivilegeEscalation: boolPtr(false),
	}

	podSpec := corev1.PodSpec{
		Containers:                   []corev1.Container{container},
		AutomountServiceAccountToken: boolPtr(false),
		SecurityContext: &corev1.PodSecurityContext{
			RunAsNonRoot: &trueVal,
			SeccompProfile: &corev1.SeccompProfile{
				Type: corev1.SeccompProfileTypeRuntimeDefault,
			},
		},
	}

	// Inject NATS sidecar for event-driven triggers (DatabaseChange or ObjectStorage).
	if HasEventTrigger(fn) && cfg.NATSSidecarImage == "" {
		slog.Warn("NATSSidecarImage not configured, skipping sidecar injection",
			"function", fn.Name, "namespace", fn.Namespace)
	}
	if HasEventTrigger(fn) && cfg.NATSSidecarImage != "" {
		projectID := fn.Spec.ProjectRef.Name
		var sidecarUID int64 = 65532
		sidecar := corev1.Container{
			Name:  "nats-sidecar",
			Image: cfg.NATSSidecarImage,
			Env: []corev1.EnvVar{
				{Name: "SIDECAR_NATS_URL", Value: "nats://" + cfg.NATSEndpoint},
				{Name: "SIDECAR_STREAM", Value: natsadmin.StreamName(projectID)},
				{Name: "SIDECAR_CONSUMER", Value: natsadmin.ConsumerName(fn.Name)},
				{Name: "SIDECAR_FUNCTION_URL", Value: "http://localhost:8080/invoke"},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: boolPtr(false),
				ReadOnlyRootFilesystem:   boolPtr(true),
				RunAsNonRoot:             &trueVal,
				RunAsUser:                &sidecarUID,
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			},
			LivenessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(8081),
					},
				},
				InitialDelaySeconds: 5,
				PeriodSeconds:       15,
			},
			ReadinessProbe: &corev1.Probe{
				ProbeHandler: corev1.ProbeHandler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt32(8081),
					},
				},
				InitialDelaySeconds: 3,
				PeriodSeconds:       10,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("25m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("64Mi"),
				},
			},
		}
		podSpec.Containers = append(podSpec.Containers, sidecar)
	}

	// RuntimeClass: use config default, override from Function CR, nvidia for self-managed GPU
	if isGPUSelfManaged(fn) {
		applySelfManagedGPU(&podSpec, cfg)
	} else {
		rc := cfg.SandboxRuntimeClass // may be empty (e.g., dev/kind environments)
		if fn.Spec.Sandbox != nil && fn.Spec.Sandbox.RuntimeClass != "" {
			rc = fn.Spec.Sandbox.RuntimeClass
		}
		if rc != "" {
			podSpec.RuntimeClassName = &rc
		}
	}

	return podSpec
}

func resolveReplicas(fn *etalbaasv1alpha1.Function) int32 {
	switch fn.Spec.Kind {
	case etalbaasv1alpha1.FunctionKindLightDeployment:
		// Light: min 1 (always running)
		if fn.Spec.Execution != nil && fn.Spec.Execution.Deployment != nil && fn.Spec.Execution.Deployment.MinReplicas != nil {
			min := *fn.Spec.Execution.Deployment.MinReplicas
			if min < 1 {
				return 1
			}
			return min
		}
		return 1
	case etalbaasv1alpha1.FunctionKindHeavyDeployment:
		// Heavy: min 0 (scale to zero via KEDA)
		if fn.Spec.Execution != nil && fn.Spec.Execution.Deployment != nil && fn.Spec.Execution.Deployment.MinReplicas != nil {
			return *fn.Spec.Execution.Deployment.MinReplicas
		}
		return 0
	default:
		return 0
	}
}

func resolveResources(fn *etalbaasv1alpha1.Function) corev1.ResourceRequirements {
	// Default resources by kind
	defaults := defaultResourcesByKind(fn.Spec.Kind)

	if fn.Spec.Resources == nil {
		return defaults
	}

	result := defaults
	if fn.Spec.Resources.Requests != nil {
		if fn.Spec.Resources.Requests.CPU != "" {
			if q, err := resource.ParseQuantity(fn.Spec.Resources.Requests.CPU); err == nil {
				result.Requests[corev1.ResourceCPU] = q
			}
		}
		if fn.Spec.Resources.Requests.Memory != "" {
			if q, err := resource.ParseQuantity(fn.Spec.Resources.Requests.Memory); err == nil {
				result.Requests[corev1.ResourceMemory] = q
			}
		}
	}
	if fn.Spec.Resources.Limits != nil {
		if fn.Spec.Resources.Limits.CPU != "" {
			if q, err := resource.ParseQuantity(fn.Spec.Resources.Limits.CPU); err == nil {
				result.Limits[corev1.ResourceCPU] = q
			}
		}
		if fn.Spec.Resources.Limits.Memory != "" {
			if q, err := resource.ParseQuantity(fn.Spec.Resources.Limits.Memory); err == nil {
				result.Limits[corev1.ResourceMemory] = q
			}
		}
	}
	return result
}

func defaultResourcesByKind(kind string) corev1.ResourceRequirements {
	switch kind {
	case etalbaasv1alpha1.FunctionKindHeavyJob:
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("1000m"),
				corev1.ResourceMemory: resource.MustParse("1Gi"),
			},
		}
	case etalbaasv1alpha1.FunctionKindHeavyDeployment:
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("500m"),
				corev1.ResourceMemory: resource.MustParse("512Mi"),
			},
		}
	default: // light-deployment
		return corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("250m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		}
	}
}

// blockedEnvVars contains system environment variable names that users must not
// override. Overriding these could break container behaviour or escalate privileges.
var blockedEnvVars = map[string]bool{
	"PATH":            true,
	"HOME":            true,
	"USER":            true,
	"SHELL":           true,
	"LD_PRELOAD":      true,
	"LD_LIBRARY_PATH": true,
	"HOSTNAME":        true,
}

func resolveEnvVars(fn *etalbaasv1alpha1.Function) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	for _, e := range fn.Spec.Env {
		if blockedEnvVars[strings.ToUpper(e.Name)] {
			slog.Warn("blocked env var override", "name", e.Name, "function", fn.Name)
			continue
		}
		if e.SecretName != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name: e.Name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: toK8sSecretName(e.SecretName),
						},
						Key: e.SecretName,
					},
				},
			})
		} else {
			envVars = append(envVars, corev1.EnvVar{
				Name:  e.Name,
				Value: e.Value,
			})
		}
	}
	return envVars
}

// toK8sSecretName converts a user-facing secret name to a DNS-1123 compatible
// k8s Secret name.  Must match pkg/k8s.ToK8sSecretName.
// Example: "GEMINI_API_KEY" → "gemini-api-key"
func toK8sSecretName(name string) string {
	lower := strings.ToLower(name)
	var b strings.Builder
	for _, r := range lower {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func isGPUSelfManaged(fn *etalbaasv1alpha1.Function) bool {
	return fn.Spec.GPU != nil && fn.Spec.GPU.Required && fn.Spec.GPU.Provider == "self-managed"
}

// IsExternalGPU returns true if the function requires an external GPU provider.
func IsExternalGPU(fn *etalbaasv1alpha1.Function) bool {
	return fn.Spec.GPU != nil && fn.Spec.GPU.Required && fn.Spec.GPU.Provider != "self-managed"
}

// applySelfManagedGPU configures the PodSpec for self-managed GPU workloads.
// Values come from operator config (Helm values) via the GPU provider config.
func applySelfManagedGPU(podSpec *corev1.PodSpec, cfg config.OperatorConfig) {
	smCfg := ResolveSelfManagedConfig(cfg)

	runtimeClass := smCfg.RuntimeClass
	podSpec.RuntimeClassName = &runtimeClass
	podSpec.NodeSelector = smCfg.NodeSelector
	podSpec.Tolerations = []corev1.Toleration{
		{
			Key:      "nvidia.com/gpu",
			Operator: corev1.TolerationOpExists,
			Effect:   corev1.TaintEffectNoSchedule,
		},
	}
	podSpec.Containers[0].Resources.Limits[corev1.ResourceName("nvidia.com/gpu")] = resource.MustParse(smCfg.ResourceLimit)
	podSpec.Containers[0].SecurityContext.ReadOnlyRootFilesystem = boolPtr(false)
}

// ResolveSelfManagedConfig extracts SelfManagedProviderConfig from operator config,
// falling back to defaults if GPU config is not set or self-managed is not configured.
func ResolveSelfManagedConfig(cfg config.OperatorConfig) gpu.SelfManagedProviderConfig {
	if cfg.GPU.Providers.SelfManaged != nil && cfg.GPU.Providers.SelfManaged.Enabled {
		smCfg := cfg.GPU.Providers.SelfManaged
		p := gpu.NewSelfManagedProvider(gpu.SelfManagedProviderConfig{
			NodeSelector:  smCfg.NodeSelector,
			RuntimeClass:  smCfg.RuntimeClass,
			ResourceLimit: smCfg.ResourceLimit,
		})
		return p.Config()
	}
	// Defaults: same as pre-refactor hardcoded values
	return gpu.NewSelfManagedProvider(gpu.SelfManagedProviderConfig{}).Config()
}

func functionLabels(projectID, funcName string) map[string]string {
	return map[string]string{
		LabelProjectID:  projectID,
		"etalbaas.io/function": funcName,
		"etalbaas.io/type":     "function",
		LabelManagedBy:  ManagedByValue,
		LabelPartOf:     PartOfValue,
	}
}

func functionSelectorLabels(projectID, funcName string) map[string]string {
	return map[string]string{
		LabelProjectID:         projectID,
		"etalbaas.io/function": funcName,
	}
}

func boolPtr(b bool) *bool {
	return &b
}
