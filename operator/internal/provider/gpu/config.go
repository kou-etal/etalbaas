package gpu

// GPUConfig holds the GPU-related operator configuration (parsed from Helm values).
type GPUConfig struct {
	// Enabled controls whether GPU features are active.
	Enabled bool
	// DefaultProvider is the provider to use when not specified per-function.
	DefaultProvider string
	// DataResidency defines the data residency constraint.
	DataResidency DataResidencyConfig
	// Providers is the map of configured providers.
	Providers ProvidersConfig
}

// DataResidencyConfig holds the data residency configuration.
type DataResidencyConfig struct {
	// Region is the data residency constraint: "any", "jp-only", "eu-only".
	Region string
}

// ProvidersConfig holds per-provider configurations.
type ProvidersConfig struct {
	RunPod      *RunPodConfig      `json:"runpod,omitempty"`
	SelfManaged *SelfManagedConfig `json:"selfManaged,omitempty"`
	// Phase 2: LambdaLabs *LambdaLabsConfig, Sakura *SakuraConfig
}

// RunPodConfig holds RunPod-specific configuration.
type RunPodConfig struct {
	Enabled      bool
	APIKeySecret string // K8s Secret name containing the API key (key: "api-key")
}

// SelfManagedConfig holds self-managed GPU node configuration.
type SelfManagedConfig struct {
	Enabled       bool
	NodeSelector  map[string]string
	RuntimeClass  string
	ResourceLimit string // e.g., "nvidia.com/gpu: 1"
}

// DefaultGPUConfig returns the default (disabled) GPU configuration.
func DefaultGPUConfig() GPUConfig {
	return GPUConfig{
		Enabled:         false,
		DefaultProvider: "runpod",
		DataResidency: DataResidencyConfig{
			Region: "any",
		},
	}
}
