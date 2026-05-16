package gpu

import (
	"context"
	"fmt"
	"time"
)

// JobID is a unique identifier for a GPU job (provider-specific format).
type JobID string

// JobState represents the state of a GPU job.
type JobState string

const (
	JobStateQueued    JobState = "QUEUED"
	JobStateRunning   JobState = "RUNNING"
	JobStateCompleted JobState = "COMPLETED"
	JobStateFailed    JobState = "FAILED"
	JobStateCancelled JobState = "CANCELLED"
)

// IsTerminal returns true if the job state is a final state.
func (s JobState) IsTerminal() bool {
	return s == JobStateCompleted || s == JobStateFailed || s == JobStateCancelled
}

// GPUJob defines the parameters for submitting a GPU job to a provider.
type GPUJob struct {
	// FunctionName is the k8s-compatible function name.
	FunctionName string
	// ProjectID is the tenant project ID.
	ProjectID string
	// Image is the container image to run.
	Image string
	// GPUType is the requested GPU type (e.g., "A10G", "H100").
	GPUType string
	// Input is an opaque JSON payload to pass to the function.
	Input []byte
	// InputStoragePath is the object storage path for large input data (optional).
	InputStoragePath string
	// OutputStoragePath is the object storage path where results should be written.
	OutputStoragePath string
	// ProviderConfig holds provider-specific settings (e.g., RunPod endpoint_id).
	ProviderConfig map[string]string
	// Env contains additional environment variables for the GPU worker.
	Env map[string]string
	// Timeout is the maximum duration for the job. Zero means provider default.
	Timeout time.Duration
}

// Validate performs basic validation on a GPUJob.
func (j GPUJob) Validate() error {
	if j.Image == "" {
		return fmt.Errorf("gpu job image must not be empty")
	}
	if j.GPUType == "" {
		return fmt.Errorf("gpu job gpu_type must not be empty")
	}
	return nil
}

// JobStatus contains the current state and result information of a GPU job.
type JobStatus struct {
	State JobState
	// DelayTimeMs is the time spent waiting in queue (milliseconds).
	DelayTimeMs int64
	// ExecutionTimeMs is the time spent executing (milliseconds).
	ExecutionTimeMs int64
	// Output is the raw output from the GPU worker.
	Output []byte
	// Error is the error message if the job failed.
	Error string
}

// Cost represents an estimated or actual cost for a GPU job.
type Cost struct {
	Amount   float64
	Currency string // "USD"
	Unit     string // "per-second", "per-hour", etc.
}

// Provider is the abstraction for GPU compute providers.
type Provider interface {
	// Name returns the provider identifier (e.g., "runpod", "self-managed").
	Name() string
	// SupportedProducts returns the product types this provider supports.
	SupportedProducts() []string
	// DataRegion returns the data residency region ("us", "jp", "any").
	DataRegion() string
	// SupportedGPUTypes returns the GPU models this provider supports.
	SupportedGPUTypes() []string

	// Submit sends a GPU job to the provider and returns the job ID.
	Submit(ctx context.Context, job GPUJob) (JobID, error)
	// Status retrieves the current status of a previously submitted job.
	Status(ctx context.Context, id JobID) (JobStatus, error)
	// Cancel attempts to cancel a running or queued job.
	Cancel(ctx context.Context, id JobID) error
	// EstimateCost returns an estimated cost for the given job parameters.
	// Phase 1: returns zero cost (no billing).
	EstimateCost(job GPUJob) (Cost, error)
}
