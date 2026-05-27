package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kou-etal/etalbaas/operator/internal/provider/gpu"
)

// maxConsecutiveErrors is the maximum number of consecutive status check failures
// before the dispatcher gives up. Prevents burning the active deadline on
// permanent errors (auth failure, invalid endpoint, etc.).
const maxConsecutiveErrors = 10

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		slog.Error("dispatcher failed", "error", err)
		os.Exit(1)
	}
	slog.Info("dispatcher completed successfully")
}

func run(ctx context.Context) error {
	providerName := os.Getenv("GPU_PROVIDER")
	if providerName == "" {
		return fmt.Errorf("required environment variable GPU_PROVIDER not set")
	}
	functionImage := os.Getenv("FUNCTION_IMAGE")
	if functionImage == "" {
		return fmt.Errorf("required environment variable FUNCTION_IMAGE not set")
	}

	gpuType := getEnv("GPU_TYPE", "any")
	apiKey := getEnv("GPU_API_KEY", "")
	resultCMName := os.Getenv("RESULT_CONFIGMAP_NAME")
	resultNS := os.Getenv("RESULT_NAMESPACE")

	providerConfig := loadProviderConfig()

	slog.Info("starting GPU dispatch",
		"provider", providerName,
		"gpu_type", gpuType,
		"function_image", functionImage,
	)

	provider, err := buildProvider(providerName, apiKey, providerConfig)
	if err != nil {
		return fmt.Errorf("build provider: %w", err)
	}

	var inputPayload json.RawMessage
	if v := os.Getenv("INPUT_PAYLOAD"); v != "" {
		inputPayload = json.RawMessage(v)
	}

	job := gpu.GPUJob{
		FunctionName:   getEnv("FUNCTION_NAME", ""),
		ProjectID:      getEnv("PROJECT_ID", ""),
		Image:          functionImage,
		GPUType:        gpuType,
		Input:          inputPayload,
		ProviderConfig: providerConfig,
	}

	if err := job.Validate(); err != nil {
		return fmt.Errorf("validate job: %w", err)
	}

	// Submit
	slog.Info("submitting job to provider", "provider", provider.Name())
	jobID, err := provider.Submit(ctx, job)
	if err != nil {
		return fmt.Errorf("submit job: %w", err)
	}
	slog.Info("job submitted", "job_id", string(jobID))

	// Poll until terminal
	status, err := pollUntilDone(ctx, provider, jobID)
	if err != nil {
		return fmt.Errorf("poll job: %w", err)
	}

	slog.Info("job finished",
		"state", string(status.State),
		"job_id", string(jobID),
		"delay_ms", status.DelayTimeMs,
		"execution_ms", status.ExecutionTimeMs,
	)

	// Write result to ConfigMap if configured (Function MS reads this).
	if resultCMName != "" && resultNS != "" {
		if writeErr := writeResultConfigMap(ctx, resultCMName, resultNS, status, job); writeErr != nil {
			slog.Error("failed to write result ConfigMap", "error", writeErr)
			// Don't fail the dispatcher for ConfigMap write errors;
			// the job result is already logged above.
		} else {
			slog.Info("result ConfigMap written", "name", resultCMName, "namespace", resultNS)
		}
	}

	if status.State == gpu.JobStateFailed {
		errMsg := status.Error
		if errMsg == "" {
			errMsg = "unknown error"
		}
		return fmt.Errorf("GPU job %s failed: %s", string(jobID), errMsg)
	}
	if status.State == gpu.JobStateCancelled {
		return fmt.Errorf("GPU job %s was cancelled", string(jobID))
	}

	return nil
}

// writeResultConfigMap creates a ConfigMap with the GPU job result so that
// Function MS can read it after the Job completes.
func writeResultConfigMap(ctx context.Context, name, namespace string, status gpu.JobStatus, job gpu.GPUJob) error {
	config, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("in-cluster config: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("create k8s client: %w", err)
	}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"etalbaas.io/type":       "gpu-result",
				"etalbaas.io/project-id": job.ProjectID,
				"etalbaas.io/function":   job.FunctionName,
			},
		},
		Data: map[string]string{
			"status":       string(status.State),
			"output":       string(status.Output),
			"error":        status.Error,
			"delay_ms":     strconv.FormatInt(status.DelayTimeMs, 10),
			"execution_ms": strconv.FormatInt(status.ExecutionTimeMs, 10),
		},
	}

	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	return err
}

func buildProvider(name, apiKey string, providerConfig map[string]string) (gpu.Provider, error) {
	switch name {
	case "runpod":
		if apiKey == "" {
			return nil, fmt.Errorf("GPU_API_KEY is required for RunPod provider")
		}
		opts := []gpu.RunPodServerlessOption{
			gpu.WithRunPodAPIKey(apiKey),
		}
		if endpointID := providerConfig["endpoint_id"]; endpointID != "" {
			opts = append(opts, gpu.WithRunPodEndpointID(endpointID))
		}
		httpClient := &http.Client{Timeout: 30 * time.Second}
		return gpu.NewRunPodServerlessProvider("", httpClient, nil, "", opts...), nil
	default:
		return nil, fmt.Errorf("unsupported GPU provider for dispatcher: %s", name)
	}
}

func pollUntilDone(ctx context.Context, provider gpu.Provider, jobID gpu.JobID) (gpu.JobStatus, error) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			slog.Warn("context cancelled, attempting to cancel job", "job_id", string(jobID))
			cancelCtx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancelFn()
			if err := provider.Cancel(cancelCtx, jobID); err != nil {
				slog.Error("failed to cancel job", "error", err)
			}
			return gpu.JobStatus{State: gpu.JobStateCancelled}, ctx.Err()

		case <-ticker.C:
			status, err := provider.Status(ctx, jobID)
			if err != nil {
				consecutiveErrors++
				slog.Warn("status check failed",
					"error", err,
					"consecutive_errors", consecutiveErrors,
					"job_id", string(jobID),
				)
				if consecutiveErrors >= maxConsecutiveErrors {
					return gpu.JobStatus{}, fmt.Errorf("giving up after %d consecutive status check failures: %w", consecutiveErrors, err)
				}
				continue
			}
			consecutiveErrors = 0
			slog.Info("job status", "state", string(status.State), "job_id", string(jobID))

			if status.State.IsTerminal() {
				return status, nil
			}
		}
	}
}

// loadProviderConfig reads PROVIDER_CONFIG_* env vars into a map.
func loadProviderConfig() map[string]string {
	config := make(map[string]string)
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "PROVIDER_CONFIG_") {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) == 2 {
				key := strings.ToLower(strings.TrimPrefix(parts[0], "PROVIDER_CONFIG_"))
				config[key] = parts[1]
			}
		}
	}
	return config
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
