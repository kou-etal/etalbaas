package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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

	job := gpu.GPUJob{
		FunctionName:   getEnv("FUNCTION_NAME", ""),
		ProjectID:      getEnv("PROJECT_ID", ""),
		Image:          functionImage,
		GPUType:        gpuType,
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
