package resources

import (
	"crypto/sha256"
	"encoding/hex"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// DesiredGPUDispatcherJob builds the Job spec for a GPU dispatcher that calls
// an external GPU provider (e.g., RunPod) on behalf of a function invocation.
// The dispatcher runs as a CPU-only gVisor-sandboxed Job in the platform namespace
// to protect provider credentials from tenant workloads.
func DesiredGPUDispatcherJob(
	fn *etalbaasv1alpha1.Function,
	imageRef string,
	cfg config.OperatorConfig,
	providerName string,
	invocationID string,
) *batchv1.Job {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	labels := gpuDispatcherLabels(projectID, funcName)

	env := []corev1.EnvVar{
		{Name: "GPU_PROVIDER", Value: providerName},
		{Name: "GPU_PRODUCT", Value: resolveGPUProduct(fn)},
		{Name: "GPU_TYPE", Value: resolveGPUType(fn)},
		{Name: "FUNCTION_IMAGE", Value: imageRef},
		{Name: "FUNCTION_NAME", Value: funcName},
		{Name: "PROJECT_ID", Value: projectID},
		{Name: "INVOCATION_ID", Value: invocationID},
	}

	// Pass provider-specific config as env vars with PROVIDER_CONFIG_ prefix.
	if fn.Spec.GPU != nil {
		for k, v := range fn.Spec.GPU.ProviderConfig {
			env = append(env, corev1.EnvVar{
				Name:  "PROVIDER_CONFIG_" + toEnvKey(k),
				Value: v,
			})
		}
	}

	// API key from K8s Secret (mounted as env var).
	// The Secret lives in the platform namespace alongside the dispatcher Job.
	apiKeySecretName := resolveAPIKeySecretName(fn, cfg)
	if apiKeySecretName != "" {
		env = append(env, corev1.EnvVar{
			Name: "GPU_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: apiKeySecretName,
					},
					Key: "api-key",
				},
			},
		})
	}

	// BackoffLimit=0: Submit to external providers is non-idempotent.
	// Retries would create duplicate billable jobs.
	var backoffLimit int32
	var ttlSeconds int32 = 600
	// ActiveDeadlineSeconds: prevent stuck dispatchers from running forever.
	var activeDeadline int64 = 1800 // 30 minutes
	gvisorRuntime := "gvisor"

	trueVal := true
	var uid int64 = 65532

	annotations := map[string]string{
		"etalbaas.io/invocation-id": invocationID,
		"etalbaas.io/function":      funcName,
		"etalbaas.io/project-id":    projectID,
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      gpuDispatcherJobName(funcName, invocationID),
			Namespace: cfg.PlatformNamespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   &activeDeadline,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      labels,
					Annotations: annotations,
				},
				Spec: corev1.PodSpec{
					RestartPolicy:    corev1.RestartPolicyNever,
					RuntimeClassName: &gvisorRuntime,
					AutomountServiceAccountToken: boolPtr(false),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &trueVal,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "dispatcher",
							Image: cfg.DispatcherImage,
							Env:   env,
							SecurityContext: &corev1.SecurityContext{
								ReadOnlyRootFilesystem:   &trueVal,
								RunAsNonRoot:             &trueVal,
								RunAsUser:                &uid,
								AllowPrivilegeEscalation: boolPtr(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("250m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

// gpuDispatcherJobName builds a deterministic Job name that fits within
// the K8s 63-character DNS label limit.
// Format: "gpu-dispatch-{funcName}-{hash8}" where hash8 is derived from invocationID.
// Function names are validated to ≤40 chars at the API layer, so the total
// is at most 13 + 40 + 1 + 8 = 62 chars. Truncation is a safety net.
func gpuDispatcherJobName(funcName, invocationID string) string {
	h := sha256.Sum256([]byte(invocationID))
	suffix := hex.EncodeToString(h[:4]) // 8 hex chars

	// "gpu-dispatch-" (13) + funcName + "-" (1) + suffix (8) = 22 + len(funcName)
	// Max 63 chars → funcName max 41.
	const maxFuncLen = 41
	name := funcName
	if len(name) > maxFuncLen {
		name = name[:maxFuncLen]
	}

	return "gpu-dispatch-" + name + "-" + suffix
}

func gpuDispatcherLabels(projectID, funcName string) map[string]string {
	return map[string]string{
		LabelProjectID:           projectID,
		"etalbaas.io/function":   funcName,
		"etalbaas.io/type":       "gpu-dispatcher",
		"etalbaas.io/gpu-dispatch": "true",
		LabelManagedBy:           ManagedByValue,
		LabelPartOf:              PartOfValue,
	}
}

func resolveGPUProduct(fn *etalbaasv1alpha1.Function) string {
	if fn.Spec.GPU != nil && fn.Spec.GPU.Product != "" {
		return fn.Spec.GPU.Product
	}
	return "serverless"
}

func resolveGPUType(fn *etalbaasv1alpha1.Function) string {
	if fn.Spec.GPU != nil && fn.Spec.GPU.Type != "" {
		return fn.Spec.GPU.Type
	}
	return "any"
}

// resolveAPIKeySecretName returns the K8s Secret name for the GPU provider API key.
func resolveAPIKeySecretName(fn *etalbaasv1alpha1.Function, cfg config.OperatorConfig) string {
	if fn.Spec.GPU == nil {
		return ""
	}
	switch fn.Spec.GPU.Provider {
	case "runpod":
		if cfg.GPU.Providers.RunPod != nil {
			return cfg.GPU.Providers.RunPod.APIKeySecret
		}
	}
	return ""
}

// toEnvKey converts a provider config key to an environment variable suffix.
// e.g., "endpoint_id" → "ENDPOINT_ID"
func toEnvKey(key string) string {
	result := make([]byte, len(key))
	for i, c := range key {
		if c >= 'a' && c <= 'z' {
			result[i] = byte(c - 'a' + 'A')
		} else if c == '-' || c == '.' {
			result[i] = '_'
		} else {
			result[i] = byte(c)
		}
	}
	return string(result)
}
