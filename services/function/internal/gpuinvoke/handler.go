package gpuinvoke

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/google/uuid"
)

const (
	maxRequestBodySize = 1 << 20 // 1 MB
	jobWatchTimeout    = 300 * time.Second
)

var functionGVR = schema.GroupVersionResource{
	Group:    "etalbaas.io",
	Version:  "v1alpha1",
	Resource: "functions",
}

// Config holds the configuration for the GPU invoke handler.
type Config struct {
	PlatformNamespace   string
	DispatcherImage     string
	SandboxRuntimeClass string
	GPUAPIKeySecret     string // K8s Secret name containing GPU provider API key
}

// Handler handles GPU function invocations via Dispatcher Jobs.
type Handler struct {
	k8sClient   kubernetes.Interface
	dynClient   dynamic.Interface
	config      Config
	rateLimiter *projectRateLimiter
}

// NewHandler creates a new GPU invoke handler.
func NewHandler(k8sClient kubernetes.Interface, dynClient dynamic.Interface, cfg Config) *Handler {
	return &Handler{
		k8sClient:   k8sClient,
		dynClient:   dynClient,
		config:      cfg,
		rateLimiter: newProjectRateLimiter(10, time.Minute), // 10 GPU invokes/min per project
	}
}

// GPUInvokeResponse is the JSON response returned to the client.
type GPUInvokeResponse struct {
	Status      string `json:"status"`
	Output      string `json:"output,omitempty"`
	Error       string `json:"error,omitempty"`
	DelayMs     string `json:"delay_ms,omitempty"`
	ExecutionMs string `json:"execution_ms,omitempty"`
}

// ServeHTTP handles POST /gpu-invoke.
// Headers X-Function-Name and X-Project-ID are set by the HTTPRoute (IDOR-safe).
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	funcName := r.Header.Get("X-Function-Name")
	projectID := r.Header.Get("X-Project-ID")
	if funcName == "" || projectID == "" {
		http.Error(w, "missing X-Function-Name or X-Project-ID header", http.StatusBadRequest)
		return
	}

	// Enforce per-project GPU invoke rate limit.
	if !h.rateLimiter.allow(projectID) {
		slog.Warn("GPU invoke rate limit exceeded", "project", projectID, "function", funcName)
		http.Error(w, "GPU invoke rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	// Read request body (user's input payload).
	body, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodySize))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	// Read Function CRD to get GPU config + imageRef (CRD is source of truth).
	fnSpec, err := h.getFunctionSpec(ctx, projectID, funcName)
	if err != nil {
		slog.Error("failed to get function CRD", "error", err, "function", funcName, "project", projectID)
		http.Error(w, "function not found or not ready", http.StatusNotFound)
		return
	}

	if !fnSpec.GPURequired {
		http.Error(w, "function does not require GPU", http.StatusBadRequest)
		return
	}
	if fnSpec.ImageRef == "" {
		http.Error(w, "function image not built yet", http.StatusServiceUnavailable)
		return
	}

	// Generate invocation ID and create Dispatcher Job.
	invocationID := uuid.New().String()
	jobName, resultCMName, err := h.createDispatcherJob(ctx, fnSpec, invocationID, body)
	if err != nil {
		slog.Error("failed to create dispatcher job", "error", err, "function", funcName)
		http.Error(w, "failed to create GPU dispatch job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	slog.Info("dispatcher job created", "job", jobName, "function", funcName, "invocation", invocationID)

	// Watch Job completion.
	if err := h.waitForJobCompletion(ctx, jobName); err != nil {
		slog.Error("job watch failed", "error", err, "job", jobName)
		http.Error(w, "GPU job failed or timed out: "+err.Error(), http.StatusBadGateway)
		return
	}

	// Read result ConfigMap.
	result, err := h.readAndCleanupResult(ctx, resultCMName)
	if err != nil {
		slog.Error("failed to read result", "error", err, "configmap", resultCMName)
		http.Error(w, "failed to read GPU job result: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result.Status == "FAILED" {
		w.WriteHeader(http.StatusBadGateway)
	}
	json.NewEncoder(w).Encode(result)
}

// functionSpec holds the parsed spec from a Function CRD.
type functionSpec struct {
	ProjectID      string
	FuncName       string
	ImageRef       string
	GPURequired    bool
	GPUProvider    string
	GPUProduct     string
	GPUType        string
	ProviderConfig map[string]string
}

func (h *Handler) getFunctionSpec(ctx context.Context, projectID, funcName string) (*functionSpec, error) {
	namespace := "project-" + projectID
	obj, err := h.dynClient.Resource(functionGVR).Namespace(namespace).Get(ctx, funcName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	spec := &functionSpec{
		ProjectID: projectID,
		FuncName:  funcName,
	}

	// Parse status.build.imageRef
	imageRef, _, _ := unstructured.NestedString(obj.Object, "status", "build", "imageRef")
	spec.ImageRef = imageRef

	// Parse spec.gpu
	gpuRequired, _, _ := unstructured.NestedBool(obj.Object, "spec", "gpu", "required")
	spec.GPURequired = gpuRequired
	if gpuRequired {
		spec.GPUProvider, _, _ = unstructured.NestedString(obj.Object, "spec", "gpu", "provider")
		spec.GPUProduct, _, _ = unstructured.NestedString(obj.Object, "spec", "gpu", "product")
		spec.GPUType, _, _ = unstructured.NestedString(obj.Object, "spec", "gpu", "type")

		pcRaw, exists, _ := unstructured.NestedStringMap(obj.Object, "spec", "gpu", "providerConfig")
		if exists {
			spec.ProviderConfig = pcRaw
		}
	}

	return spec, nil
}

func (h *Handler) createDispatcherJob(ctx context.Context, fn *functionSpec, invocationID string, input []byte) (jobName, resultCMName string, err error) {
	resultCMName = gpuResultConfigMapName(invocationID)
	jobName = gpuDispatcherJobName(fn.FuncName, invocationID)

	gpuType := fn.GPUType
	if gpuType == "" {
		gpuType = "any"
	}
	gpuProduct := fn.GPUProduct
	if gpuProduct == "" {
		gpuProduct = "serverless"
	}

	env := []corev1.EnvVar{
		{Name: "GPU_PROVIDER", Value: fn.GPUProvider},
		{Name: "GPU_PRODUCT", Value: gpuProduct},
		{Name: "GPU_TYPE", Value: gpuType},
		{Name: "FUNCTION_IMAGE", Value: fn.ImageRef},
		{Name: "FUNCTION_NAME", Value: fn.FuncName},
		{Name: "PROJECT_ID", Value: fn.ProjectID},
		{Name: "INVOCATION_ID", Value: invocationID},
		{Name: "RESULT_CONFIGMAP_NAME", Value: resultCMName},
		{Name: "RESULT_NAMESPACE", Value: h.config.PlatformNamespace},
	}

	// Pass provider config as PROVIDER_CONFIG_* env vars.
	for k, v := range fn.ProviderConfig {
		env = append(env, corev1.EnvVar{
			Name:  "PROVIDER_CONFIG_" + toEnvKey(k),
			Value: v,
		})
	}

	// GPU API key from K8s Secret.
	if h.config.GPUAPIKeySecret != "" {
		env = append(env, corev1.EnvVar{
			Name: "GPU_API_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: h.config.GPUAPIKeySecret,
					},
					Key: "api-key",
				},
			},
		})
	}

	// Pass user input as INPUT_PAYLOAD env var (for small payloads).
	if len(input) > 0 {
		env = append(env, corev1.EnvVar{
			Name:  "INPUT_PAYLOAD",
			Value: string(input),
		})
	}

	var backoffLimit int32
	var ttlSeconds int32 = 600
	var activeDeadline int64 = 1800
	trueVal := true
	var uid int64 = 65532

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: h.config.PlatformNamespace,
			Labels: map[string]string{
				"etalbaas.io/project-id":    fn.ProjectID,
				"etalbaas.io/function":      fn.FuncName,
				"etalbaas.io/type":          "gpu-dispatcher",
				"etalbaas.io/gpu-dispatch":  "true",
				"app.kubernetes.io/managed-by": "etalbaas-function-ms",
				"app.kubernetes.io/part-of":    "etalbaas",
			},
			Annotations: map[string]string{
				"etalbaas.io/invocation-id": invocationID,
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSeconds,
			ActiveDeadlineSeconds:   &activeDeadline,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"etalbaas.io/project-id":   fn.ProjectID,
						"etalbaas.io/function":     fn.FuncName,
						"etalbaas.io/type":         "gpu-dispatcher",
						"etalbaas.io/gpu-dispatch": "true",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:           "gpu-dispatcher",
					RestartPolicy:                corev1.RestartPolicyNever,
					AutomountServiceAccountToken: &trueVal,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &trueVal,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					Containers: []corev1.Container{
						{
							Name:  "dispatcher",
							Image: h.config.DispatcherImage,
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

	// Set RuntimeClassName if configured (gVisor in production).
	if h.config.SandboxRuntimeClass != "" {
		rc := h.config.SandboxRuntimeClass
		job.Spec.Template.Spec.RuntimeClassName = &rc
	}

	_, err = h.k8sClient.BatchV1().Jobs(h.config.PlatformNamespace).Create(ctx, job, metav1.CreateOptions{})
	return jobName, resultCMName, err
}

func (h *Handler) waitForJobCompletion(ctx context.Context, jobName string) error {
	ctx, cancel := context.WithTimeout(ctx, jobWatchTimeout)
	defer cancel()

	watcher, err := h.k8sClient.BatchV1().Jobs(h.config.PlatformNamespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: "metadata.name=" + jobName,
	})
	if err != nil {
		return fmt.Errorf("watch job: %w", err)
	}
	defer watcher.Stop()

	for event := range watcher.ResultChan() {
		if event.Type == watch.Error {
			return fmt.Errorf("watch error")
		}
		job, ok := event.Object.(*batchv1.Job)
		if !ok {
			continue
		}
		if job.Status.Succeeded > 0 {
			return nil
		}
		for _, c := range job.Status.Conditions {
			if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
				return fmt.Errorf("dispatcher job failed: %s", c.Message)
			}
		}
	}

	return fmt.Errorf("job watch channel closed (timeout or context cancelled)")
}

func (h *Handler) readAndCleanupResult(ctx context.Context, cmName string) (*GPUInvokeResponse, error) {
	cm, err := h.k8sClient.CoreV1().ConfigMaps(h.config.PlatformNamespace).Get(ctx, cmName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get result configmap %s: %w", cmName, err)
	}

	result := &GPUInvokeResponse{
		Status:      cm.Data["status"],
		Output:      cm.Data["output"],
		Error:       cm.Data["error"],
		DelayMs:     cm.Data["delay_ms"],
		ExecutionMs: cm.Data["execution_ms"],
	}

	// Cleanup: delete the result ConfigMap (best effort).
	_ = h.k8sClient.CoreV1().ConfigMaps(h.config.PlatformNamespace).Delete(ctx, cmName, metav1.DeleteOptions{})

	return result, nil
}

// gpuResultConfigMapName matches the naming in operator/internal/resources/gpu_dispatcher.go.
func gpuResultConfigMapName(invocationID string) string {
	// Use first 8 chars of invocation ID for brevity.
	// Must match the operator's GPUResultConfigMapName function.
	if len(invocationID) > 8 {
		return "gpu-result-" + invocationID[:8]
	}
	return "gpu-result-" + invocationID
}

// gpuDispatcherJobName builds a deterministic Job name.
// Must produce names compatible with the operator's naming scheme.
func gpuDispatcherJobName(funcName, invocationID string) string {
	suffix := invocationID
	if len(suffix) > 8 {
		suffix = suffix[:8]
	}
	const maxFuncLen = 41
	name := funcName
	if len(name) > maxFuncLen {
		name = name[:maxFuncLen]
	}
	return "gpu-dispatch-" + name + "-" + suffix
}

// toEnvKey converts a provider config key to an env var suffix.
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

func boolPtr(b bool) *bool {
	return &b
}
