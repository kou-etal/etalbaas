package resources

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/provider/gpu"
)

func TestDesiredGPUDispatcherJob_BasicFields(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Type:     "A100",
				Provider: "runpod",
				Product:  "serverless",
				ProviderConfig: map[string]string{
					"endpoint_id": "ep-abc123",
				},
			},
		},
	}

	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:v1",
		GPU: gpu.GPUConfig{
			Enabled: true,
			Providers: gpu.ProvidersConfig{
				RunPod: &gpu.RunPodConfig{
					Enabled:      true,
					APIKeySecret: "runpod-api-key",
				},
			},
		},
	}

	job := DesiredGPUDispatcherJob(fn, "registry/my-func:sha-abc", cfg, "runpod", "inv-001")

	// Name uses hash suffix for DNS label safety (max 63 chars).
	wantName := gpuDispatcherJobName("my-func", "inv-001")
	if job.Name != wantName {
		t.Errorf("Name = %q, want %q", job.Name, wantName)
	}
	if !strings.HasPrefix(job.Name, "gpu-dispatch-my-func-") {
		t.Errorf("Name should start with %q, got %q", "gpu-dispatch-my-func-", job.Name)
	}
	if len(job.Name) > 63 {
		t.Errorf("Name length = %d, exceeds K8s DNS label limit of 63", len(job.Name))
	}
	if job.Namespace != "platform-system" {
		t.Errorf("Namespace = %q, want %q", job.Namespace, "platform-system")
	}
}

func TestDesiredGPUDispatcherJob_Labels(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Provider: "runpod",
			},
		},
	}

	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:latest",
		GPU:               gpu.DefaultGPUConfig(),
	}

	job := DesiredGPUDispatcherJob(fn, "img:v1", cfg, "runpod", "inv-002")

	labels := job.Labels
	if labels[LabelProjectID] != "proj-123" {
		t.Errorf("project-id label = %q, want %q", labels[LabelProjectID], "proj-123")
	}
	if labels["etalbaas.io/function"] != "my-func" {
		t.Errorf("function label = %q, want %q", labels["etalbaas.io/function"], "my-func")
	}
	if labels["etalbaas.io/type"] != "gpu-dispatcher" {
		t.Errorf("type label = %q, want %q", labels["etalbaas.io/type"], "gpu-dispatcher")
	}
	if labels["etalbaas.io/gpu-dispatch"] != "true" {
		t.Errorf("gpu-dispatch label = %q, want %q", labels["etalbaas.io/gpu-dispatch"], "true")
	}
	if labels[LabelManagedBy] != ManagedByValue {
		t.Errorf("managed-by label = %q, want %q", labels[LabelManagedBy], ManagedByValue)
	}
}

func TestDesiredGPUDispatcherJob_EnvVars(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Type:     "H100",
				Provider: "runpod",
				Product:  "serverless",
				ProviderConfig: map[string]string{
					"endpoint_id": "ep-xyz",
				},
			},
		},
	}

	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:latest",
		GPU: gpu.GPUConfig{
			Enabled: true,
			Providers: gpu.ProvidersConfig{
				RunPod: &gpu.RunPodConfig{
					Enabled:      true,
					APIKeySecret: "runpod-api-key",
				},
			},
		},
	}

	job := DesiredGPUDispatcherJob(fn, "img:v1", cfg, "runpod", "inv-003")

	container := job.Spec.Template.Spec.Containers[0]
	envMap := make(map[string]string)
	var secretEnvVars []string
	for _, e := range container.Env {
		if e.ValueFrom != nil && e.ValueFrom.SecretKeyRef != nil {
			secretEnvVars = append(secretEnvVars, e.Name)
			envMap[e.Name] = "secret:" + e.ValueFrom.SecretKeyRef.Name
		} else {
			envMap[e.Name] = e.Value
		}
	}

	// Check standard env vars
	checks := map[string]string{
		"GPU_PROVIDER":                 "runpod",
		"GPU_PRODUCT":                  "serverless",
		"GPU_TYPE":                     "H100",
		"FUNCTION_IMAGE":              "img:v1",
		"FUNCTION_NAME":               "my-func",
		"PROJECT_ID":                  "proj-123",
		"INVOCATION_ID":               "inv-003",
		"PROVIDER_CONFIG_ENDPOINT_ID": "ep-xyz",
	}
	for k, want := range checks {
		if got := envMap[k]; got != want {
			t.Errorf("env %s = %q, want %q", k, got, want)
		}
	}

	// Check API key from Secret
	if envMap["GPU_API_KEY"] != "secret:runpod-api-key" {
		t.Errorf("GPU_API_KEY secret ref = %q, want secret:runpod-api-key", envMap["GPU_API_KEY"])
	}
}

func TestDesiredGPUDispatcherJob_GVisorSandbox(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Provider: "runpod",
			},
		},
	}

	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:latest",
		GPU:               gpu.DefaultGPUConfig(),
	}

	job := DesiredGPUDispatcherJob(fn, "img:v1", cfg, "runpod", "inv-004")

	podSpec := job.Spec.Template.Spec

	// RuntimeClass should be gvisor (CPU-only Job)
	if podSpec.RuntimeClassName == nil || *podSpec.RuntimeClassName != "gvisor" {
		rc := "<nil>"
		if podSpec.RuntimeClassName != nil {
			rc = *podSpec.RuntimeClassName
		}
		t.Errorf("RuntimeClassName = %q, want %q", rc, "gvisor")
	}

	// RestartPolicy
	if podSpec.RestartPolicy != "Never" {
		t.Errorf("RestartPolicy = %q, want %q", podSpec.RestartPolicy, "Never")
	}

	// Security context
	container := podSpec.Containers[0]
	if container.SecurityContext == nil {
		t.Fatal("SecurityContext is nil")
	}
	if container.SecurityContext.ReadOnlyRootFilesystem == nil || !*container.SecurityContext.ReadOnlyRootFilesystem {
		t.Error("ReadOnlyRootFilesystem should be true")
	}
	if container.SecurityContext.RunAsNonRoot == nil || !*container.SecurityContext.RunAsNonRoot {
		t.Error("RunAsNonRoot should be true")
	}
	if container.SecurityContext.AllowPrivilegeEscalation == nil || *container.SecurityContext.AllowPrivilegeEscalation {
		t.Error("AllowPrivilegeEscalation should be false")
	}
	if len(container.SecurityContext.Capabilities.Drop) != 1 || container.SecurityContext.Capabilities.Drop[0] != "ALL" {
		t.Errorf("Capabilities.Drop = %v, want [ALL]", container.SecurityContext.Capabilities.Drop)
	}

	// AutomountServiceAccountToken should be false
	if podSpec.AutomountServiceAccountToken == nil || *podSpec.AutomountServiceAccountToken {
		t.Error("AutomountServiceAccountToken should be false")
	}
}

func TestDesiredGPUDispatcherJob_Resources(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Provider: "runpod",
			},
		},
	}

	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:latest",
		GPU:               gpu.DefaultGPUConfig(),
	}

	job := DesiredGPUDispatcherJob(fn, "img:v1", cfg, "runpod", "inv-005")

	container := job.Spec.Template.Spec.Containers[0]
	res := container.Resources

	cpuReq := res.Requests["cpu"]
	if cpuReq.String() != "100m" {
		t.Errorf("CPU request = %q, want %q", cpuReq.String(), "100m")
	}
	memReq := res.Requests["memory"]
	if memReq.String() != "64Mi" {
		t.Errorf("Memory request = %q, want %q", memReq.String(), "64Mi")
	}
	cpuLim := res.Limits["cpu"]
	if cpuLim.String() != "250m" {
		t.Errorf("CPU limit = %q, want %q", cpuLim.String(), "250m")
	}
	memLim := res.Limits["memory"]
	if memLim.String() != "128Mi" {
		t.Errorf("Memory limit = %q, want %q", memLim.String(), "128Mi")
	}
}

func TestDesiredGPUDispatcherJob_NoAPIKeySecret(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Provider: "runpod",
			},
		},
	}

	// No RunPod config → no API key secret
	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:latest",
		GPU:               gpu.DefaultGPUConfig(),
	}

	job := DesiredGPUDispatcherJob(fn, "img:v1", cfg, "runpod", "inv-006")

	container := job.Spec.Template.Spec.Containers[0]
	for _, e := range container.Env {
		if e.Name == "GPU_API_KEY" {
			t.Error("GPU_API_KEY should not be set when no API key secret is configured")
		}
	}
}

func TestDesiredGPUDispatcherJob_DefaultProduct(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Provider: "runpod",
				// Product not set → defaults to "serverless"
			},
		},
	}

	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:latest",
		GPU:               gpu.DefaultGPUConfig(),
	}

	job := DesiredGPUDispatcherJob(fn, "img:v1", cfg, "runpod", "inv-007")

	container := job.Spec.Template.Spec.Containers[0]
	for _, e := range container.Env {
		if e.Name == "GPU_PRODUCT" && e.Value != "serverless" {
			t.Errorf("GPU_PRODUCT = %q, want %q", e.Value, "serverless")
		}
		if e.Name == "GPU_TYPE" && e.Value != "any" {
			t.Errorf("GPU_TYPE = %q, want %q", e.Value, "any")
		}
	}
}

func TestDesiredGPUDispatcherJob_BackoffAndTTL(t *testing.T) {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-func",
			Namespace: "proj-ns",
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			ProjectRef: etalbaasv1alpha1.ProjectReference{Name: "proj-123"},
			GPU: &etalbaasv1alpha1.GPUSpec{
				Required: true,
				Provider: "runpod",
			},
		},
	}

	cfg := config.OperatorConfig{
		PlatformNamespace: "platform-system",
		DispatcherImage:   "etalbaas/gpu-dispatcher:latest",
		GPU:               gpu.DefaultGPUConfig(),
	}

	job := DesiredGPUDispatcherJob(fn, "img:v1", cfg, "runpod", "inv-008")

	if job.Spec.BackoffLimit == nil || *job.Spec.BackoffLimit != 0 {
		t.Errorf("BackoffLimit = %v, want 0", job.Spec.BackoffLimit)
	}
	if job.Spec.TTLSecondsAfterFinished == nil || *job.Spec.TTLSecondsAfterFinished != 600 {
		t.Errorf("TTLSecondsAfterFinished = %v, want 600", job.Spec.TTLSecondsAfterFinished)
	}
	if job.Spec.ActiveDeadlineSeconds == nil || *job.Spec.ActiveDeadlineSeconds != 1800 {
		t.Errorf("ActiveDeadlineSeconds = %v, want 1800", job.Spec.ActiveDeadlineSeconds)
	}
}

func TestGpuDispatcherJobName(t *testing.T) {
	tests := []struct {
		name         string
		funcName     string
		invocationID string
	}{
		{"short", "my-func", "inv-001"},
		{"max-func-name-40", strings.Repeat("a", 40), "inv-001"},
		{"long-uuid-invocation", "my-func", "550e8400-e29b-41d4-a716-446655440000"},
		{"both-long", strings.Repeat("a", 40), "550e8400-e29b-41d4-a716-446655440000"},
		{"exceeds-safety", strings.Repeat("x", 50), "550e8400-e29b-41d4-a716-446655440000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := gpuDispatcherJobName(tt.funcName, tt.invocationID)
			if len(got) > 63 {
				t.Errorf("name length = %d, exceeds 63: %q", len(got), got)
			}
			if !strings.HasPrefix(got, "gpu-dispatch-") {
				t.Errorf("name should start with 'gpu-dispatch-', got %q", got)
			}
			// Deterministic: same inputs → same output
			got2 := gpuDispatcherJobName(tt.funcName, tt.invocationID)
			if got != got2 {
				t.Errorf("not deterministic: %q != %q", got, got2)
			}
		})
	}

	// Different invocationIDs → different names
	a := gpuDispatcherJobName("func", "inv-001")
	b := gpuDispatcherJobName("func", "inv-002")
	if a == b {
		t.Errorf("different invocationIDs produced same name: %q", a)
	}
}

func TestToEnvKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"endpoint_id", "ENDPOINT_ID"},
		{"zone", "ZONE"},
		{"api-key", "API_KEY"},
		{"some.value", "SOME_VALUE"},
		{"ALREADY_UPPER", "ALREADY_UPPER"},
	}
	for _, tt := range tests {
		got := toEnvKey(tt.input)
		if got != tt.want {
			t.Errorf("toEnvKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
