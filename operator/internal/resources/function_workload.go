package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
)

// DesiredFunctionDeployment builds the desired Deployment for a Function workload.
// Returns nil for heavy-job functions (they use Job templates, not Deployments).
func DesiredFunctionDeployment(fn *etalbaasv1alpha1.Function, imageRef string) *appsv1.Deployment {
	if fn.Spec.Kind == etalbaasv1alpha1.FunctionKindHeavyJob {
		return nil
	}

	namespace := fn.Namespace
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	labels := functionLabels(projectID, funcName)
	selectorLabels := functionSelectorLabels(projectID, funcName)

	replicas := resolveReplicas(fn)
	podSpec := buildFunctionPodSpec(fn, imageRef)

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
func buildFunctionPodSpec(fn *etalbaasv1alpha1.Function, imageRef string) corev1.PodSpec {
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

	// RuntimeClass: gVisor by default, nvidia for self-managed GPU
	if isGPUSelfManaged(fn) {
		nvidiaRuntime := "nvidia"
		podSpec.RuntimeClassName = &nvidiaRuntime
		// GPU node selector and tolerations
		podSpec.NodeSelector = map[string]string{"gpu": "true"}
		podSpec.Tolerations = []corev1.Toleration{
			{
				Key:      "nvidia.com/gpu",
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			},
		}
		// Add GPU resource limit
		podSpec.Containers[0].Resources.Limits["nvidia.com/gpu"] = resource.MustParse("1")

		// Override security context for GPU (gVisor is not compatible)
		podSpec.Containers[0].SecurityContext.ReadOnlyRootFilesystem = boolPtr(false)
	} else {
		gvisorRuntime := "gvisor"
		if fn.Spec.Sandbox != nil && fn.Spec.Sandbox.RuntimeClass != "" {
			gvisorRuntime = fn.Spec.Sandbox.RuntimeClass
		}
		podSpec.RuntimeClassName = &gvisorRuntime
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

func resolveEnvVars(fn *etalbaasv1alpha1.Function) []corev1.EnvVar {
	var envVars []corev1.EnvVar
	for _, e := range fn.Spec.Env {
		if e.SecretName != "" {
			envVars = append(envVars, corev1.EnvVar{
				Name: e.Name,
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: e.SecretName,
						},
						Key: "value",
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

func isGPUSelfManaged(fn *etalbaasv1alpha1.Function) bool {
	return fn.Spec.GPU != nil && fn.Spec.GPU.Required && fn.Spec.GPU.Provider == "self-managed"
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
