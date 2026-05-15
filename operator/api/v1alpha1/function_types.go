package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// FunctionSpec defines the desired state of a Function.
type FunctionSpec struct {
	// DisplayName is the human-readable name of the function.
	// +kubebuilder:validation:MaxLength=100
	// +optional
	DisplayName string `json:"displayName,omitempty"`

	// ProjectRef references the parent Project.
	ProjectRef ProjectReference `json:"projectRef"`

	// Kind is the function kind that determines execution strategy.
	// +kubebuilder:validation:Enum="heavy-job";"heavy-deployment";"light-deployment"
	Kind string `json:"kind"`

	// Source defines where the function code is located.
	Source FunctionSource `json:"source"`

	// Runtime defines the runtime configuration for building the function.
	Runtime RuntimeSpec `json:"runtime"`

	// Resources defines the compute resources for the function.
	// +optional
	Resources *FunctionResources `json:"resources,omitempty"`

	// Sandbox defines the sandbox configuration. Platform-enforced, not user-configurable.
	// +optional
	Sandbox *SandboxSpec `json:"sandbox,omitempty"`

	// GPU defines GPU requirements. If not set, the function runs on CPU only.
	// +optional
	GPU *GPUSpec `json:"gpu,omitempty"`

	// Execution defines the execution mode and parameters.
	// +optional
	Execution *ExecutionSpec `json:"execution,omitempty"`

	// Triggers defines the event triggers for this function.
	// +optional
	Triggers []TriggerSpec `json:"triggers,omitempty"`

	// Env defines environment variables for the function.
	// +optional
	Env []EnvVar `json:"env,omitempty"`
}

// ProjectReference is a reference to a Project CR.
type ProjectReference struct {
	// Name is the name of the Project CR.
	Name string `json:"name"`
}

// FunctionSource defines the code source for a function.
type FunctionSource struct {
	// Type is the source type discriminator.
	// +kubebuilder:validation:Enum=git;zip;inline
	Type string `json:"type"`

	// Git defines the git source configuration.
	// +optional
	Git *GitSource `json:"git,omitempty"`

	// Zip defines the zip source configuration.
	// +optional
	Zip *ZipSource `json:"zip,omitempty"`

	// Inline defines the inline source configuration.
	// +optional
	Inline *InlineSource `json:"inline,omitempty"`
}

// GitSource defines a git repository as the function source.
type GitSource struct {
	// Repo is the git repository URL.
	Repo string `json:"repo"`

	// Ref is the git reference (branch, tag, or commit SHA).
	// +optional
	Ref string `json:"ref,omitempty"`

	// Path is the subdirectory within the repository.
	// +kubebuilder:default="/"
	// +optional
	Path string `json:"path,omitempty"`
}

// ZipSource defines a zip archive as the function source.
type ZipSource struct {
	// StoragePath is the path to the uploaded zip in object storage.
	StoragePath string `json:"storagePath"`

	// Sha256 is the SHA-256 hash of the zip file for integrity verification.
	// +optional
	Sha256 string `json:"sha256,omitempty"`
}

// InlineSource defines inline code as the function source.
type InlineSource struct {
	// Files is a map of filename to file content.
	Files map[string]string `json:"files"`

	// Entrypoint is the main file to execute.
	Entrypoint string `json:"entrypoint"`
}

// RuntimeSpec defines the runtime configuration.
type RuntimeSpec struct {
	// Preset is the runtime preset name.
	// +kubebuilder:validation:Enum="python-3.11";"python-3.11-ml";"python-3.12";"node-20";"node-22";"go-1.22";"custom"
	Preset string `json:"preset"`

	// Requirements is a list of additional packages to install (pip/npm).
	// +optional
	Requirements []string `json:"requirements,omitempty"`

	// Dockerfile is the custom Dockerfile content. Only valid when preset is "custom".
	// +optional
	Dockerfile string `json:"dockerfile,omitempty"`
}

// FunctionResources defines the compute resources for a function.
type FunctionResources struct {
	// Requests defines the minimum resources.
	// +optional
	Requests *ComponentResources `json:"requests,omitempty"`

	// Limits defines the maximum resources.
	// +optional
	Limits *ComponentResources `json:"limits,omitempty"`
}

// SandboxSpec defines the sandbox configuration for function execution.
type SandboxSpec struct {
	// RuntimeClass is the Kubernetes RuntimeClass to use.
	// +kubebuilder:default="gvisor"
	RuntimeClass string `json:"runtimeClass,omitempty"`

	// ReadOnlyRootFilesystem enforces a read-only root filesystem.
	// +kubebuilder:default=true
	ReadOnlyRootFilesystem *bool `json:"readOnlyRootFilesystem,omitempty"`

	// RunAsNonRoot enforces running as a non-root user.
	// +kubebuilder:default=true
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`
}

// GPUSpec defines GPU requirements for a function.
type GPUSpec struct {
	// Required indicates whether GPU is required.
	Required bool `json:"required"`

	// Type is the GPU type (e.g., T4, A10G, A100, H100).
	// +optional
	Type string `json:"type,omitempty"`

	// Provider is the GPU provider.
	// +kubebuilder:validation:Enum=runpod;"lambda-labs";sakura;"self-managed"
	Provider string `json:"provider"`

	// Product is the provider-specific product (e.g., serverless, pods).
	// +optional
	Product string `json:"product,omitempty"`
}

// ExecutionSpec defines the execution mode and parameters.
type ExecutionSpec struct {
	// Mode is the execution mode.
	// +kubebuilder:validation:Enum=Job;Deployment
	Mode string `json:"mode"`

	// Deployment defines deployment-specific parameters.
	// +optional
	Deployment *DeploymentExecutionSpec `json:"deployment,omitempty"`

	// Job defines job-specific parameters.
	// +optional
	Job *JobExecutionSpec `json:"job,omitempty"`
}

// DeploymentExecutionSpec defines parameters for Deployment-mode execution.
type DeploymentExecutionSpec struct {
	// MinReplicas is the minimum number of replicas.
	// +kubebuilder:default=0
	MinReplicas *int32 `json:"minReplicas,omitempty"`

	// MaxReplicas is the maximum number of replicas.
	// +kubebuilder:default=10
	MaxReplicas *int32 `json:"maxReplicas,omitempty"`

	// ScaleDownDelay is the seconds to wait before scaling to zero (Heavy Deployment only).
	// +optional
	ScaleDownDelay *int32 `json:"scaleDownDelay,omitempty"`
}

// JobExecutionSpec defines parameters for Job-mode execution.
type JobExecutionSpec struct {
	// Timeout is the job timeout in seconds. 0 means no timeout.
	// +kubebuilder:default=0
	Timeout *int32 `json:"timeout,omitempty"`

	// TTLSecondsAfterFinished is the time to keep the Job after completion.
	// +kubebuilder:default=600
	TTLSecondsAfterFinished *int32 `json:"ttlSecondsAfterFinished,omitempty"`

	// BackoffLimit is the number of retries before considering the Job failed.
	// +kubebuilder:default=2
	BackoffLimit *int32 `json:"backoffLimit,omitempty"`
}

// TriggerSpec defines an event trigger for a function.
type TriggerSpec struct {
	// Type is the trigger type.
	// +kubebuilder:validation:Enum=Http;ObjectStorage;DatabaseChange
	Type string `json:"type"`

	// Http defines HTTP trigger configuration.
	// +optional
	Http *HttpTrigger `json:"http,omitempty"`

	// ObjectStorage defines object storage trigger configuration.
	// +optional
	ObjectStorage *ObjectStorageTrigger `json:"objectStorage,omitempty"`

	// DatabaseChange defines database change trigger configuration.
	// +optional
	DatabaseChange *DatabaseChangeTrigger `json:"databaseChange,omitempty"`

	// RetryPolicy defines the retry policy for this trigger.
	// +optional
	RetryPolicy *RetryPolicy `json:"retryPolicy,omitempty"`
}

// HttpTrigger defines an HTTP trigger.
type HttpTrigger struct {
	// Path is the HTTP path for the trigger.
	// +kubebuilder:default="/invoke"
	Path string `json:"path,omitempty"`

	// Authentication is the authentication method.
	// +kubebuilder:default="apikey"
	Authentication string `json:"authentication,omitempty"`
}

// ObjectStorageTrigger defines an object storage trigger.
type ObjectStorageTrigger struct {
	// Bucket is the storage bucket name.
	Bucket string `json:"bucket"`

	// Prefix is the object key prefix to filter.
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Events is the list of storage events to trigger on.
	Events []string `json:"events"`
}

// DatabaseChangeTrigger defines a database change (CDC) trigger.
type DatabaseChangeTrigger struct {
	// Table is the database table to watch.
	Table string `json:"table"`

	// Operations is the list of operations to trigger on (INSERT, UPDATE, DELETE).
	Operations []string `json:"operations"`

	// Filter is an optional WHERE condition.
	// +optional
	Filter string `json:"filter,omitempty"`

	// IncludeColumns limits which columns are included in the change event.
	// +optional
	IncludeColumns []string `json:"includeColumns,omitempty"`
}

// RetryPolicy defines the retry behavior for a trigger.
type RetryPolicy struct {
	// MaxAttempts is the maximum number of retry attempts.
	// +kubebuilder:default=3
	MaxAttempts *int32 `json:"maxAttempts,omitempty"`

	// Backoff defines the backoff strategy.
	// +optional
	Backoff *BackoffPolicy `json:"backoff,omitempty"`
}

// BackoffPolicy defines the backoff parameters.
type BackoffPolicy struct {
	// Initial is the initial backoff duration (e.g., "5s").
	// +kubebuilder:default="5s"
	Initial string `json:"initial,omitempty"`

	// Multiplier is the backoff multiplier.
	// +kubebuilder:default=2
	Multiplier *int32 `json:"multiplier,omitempty"`
}

// EnvVar defines an environment variable for a function.
type EnvVar struct {
	// Name is the environment variable name.
	Name string `json:"name"`

	// Value is the plain-text value. Mutually exclusive with SecretName.
	// +optional
	Value string `json:"value,omitempty"`

	// SecretName is the k8s Secret name to reference. Mutually exclusive with Value.
	// +optional
	SecretName string `json:"secretName,omitempty"`
}

// FunctionStatus defines the observed state of a Function.
type FunctionStatus struct {
	// Phase is the current phase of the function.
	// +kubebuilder:validation:Enum=Building;Ready;Failed
	Phase string `json:"phase,omitempty"`

	// Build contains the build status information.
	// +optional
	Build *BuildStatus `json:"build,omitempty"`

	// Execution contains the execution status information.
	// +optional
	Execution *ExecutionStatus `json:"execution,omitempty"`

	// Triggers contains the status of each trigger.
	// +optional
	Triggers []TriggerStatus `json:"triggers,omitempty"`

	// Stats contains runtime statistics.
	// +optional
	Stats *FunctionStats `json:"stats,omitempty"`

	// ObservedGeneration is the most recent generation observed.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of the function's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// BuildStatus contains information about the function build.
type BuildStatus struct {
	// Status is the build status (e.g., "Building", "Succeeded", "Failed").
	Status string `json:"status,omitempty"`

	// ImageRef is the built container image reference.
	// +optional
	ImageRef string `json:"imageRef,omitempty"`

	// ImageDigest is the image digest.
	// +optional
	ImageDigest string `json:"imageDigest,omitempty"`

	// BuildDurationSeconds is the build duration in seconds.
	// +optional
	BuildDurationSeconds *int32 `json:"buildDurationSeconds,omitempty"`

	// LastBuiltAt is the timestamp of the last successful build.
	// +optional
	LastBuiltAt *metav1.Time `json:"lastBuiltAt,omitempty"`
}

// ExecutionStatus contains the runtime execution information.
type ExecutionStatus struct {
	// Mode is the resolved execution mode (Job or Deployment).
	// +optional
	Mode string `json:"mode,omitempty"`

	// JobTemplate is the Job template name for heavy-job functions.
	// +optional
	JobTemplate string `json:"jobTemplate,omitempty"`

	// RuntimeClass is the resolved RuntimeClass.
	// +optional
	RuntimeClass string `json:"runtimeClass,omitempty"`

	// GpuProvider is the resolved GPU provider.
	// +optional
	GpuProvider string `json:"gpuProvider,omitempty"`
}

// TriggerStatus contains the status of a single trigger.
type TriggerStatus struct {
	// Type is the trigger type.
	Type string `json:"type"`

	// Status is the trigger status (e.g., "Active", "Inactive").
	Status string `json:"status,omitempty"`

	// Endpoint is the trigger endpoint URL (for HTTP triggers).
	// +optional
	Endpoint string `json:"endpoint,omitempty"`
}

// FunctionStats contains runtime statistics.
type FunctionStats struct {
	// TotalInvocations is the total number of invocations.
	TotalInvocations int64 `json:"totalInvocations,omitempty"`

	// SuccessCount is the number of successful invocations.
	SuccessCount int64 `json:"successCount,omitempty"`

	// ErrorCount is the number of failed invocations.
	ErrorCount int64 `json:"errorCount,omitempty"`

	// AvgDurationMs is the average invocation duration in milliseconds.
	AvgDurationMs int64 `json:"avgDurationMs,omitempty"`
}

// Function kind constants.
const (
	FunctionKindHeavyJob        = "heavy-job"
	FunctionKindHeavyDeployment = "heavy-deployment"
	FunctionKindLightDeployment = "light-deployment"
)

// Function phase constants.
const (
	FunctionPhaseBuilding = "Building"
	FunctionPhaseReady    = "Ready"
	FunctionPhaseFailed   = "Failed"
)

// Build status constants.
const (
	BuildStatusBuilding  = "Building"
	BuildStatusSucceeded = "Succeeded"
	BuildStatusFailed    = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Kind",type=string,JSONPath=`.spec.kind`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Function is the Schema for the functions API.
type Function struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FunctionSpec   `json:"spec,omitempty"`
	Status FunctionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FunctionList contains a list of Function.
type FunctionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Function `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Function{}, &FunctionList{})
}
