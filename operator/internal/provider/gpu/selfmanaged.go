package gpu

import (
	"context"
	"fmt"
)

// SelfManagedProviderConfig holds the resolved configuration for self-managed GPU workloads.
type SelfManagedProviderConfig struct {
	NodeSelector  map[string]string
	RuntimeClass  string
	ResourceLimit string
}

// SelfManagedProvider implements Provider for on-prem GPU nodes.
// Unlike external providers, self-managed GPU runs directly as a K8s workload.
// Submit/Status/Cancel return errors because the operator manages the workload directly.
type SelfManagedProvider struct {
	config SelfManagedProviderConfig
}

// NewSelfManagedProvider creates a SelfManagedProvider with defaults applied.
func NewSelfManagedProvider(config SelfManagedProviderConfig) *SelfManagedProvider {
	if config.RuntimeClass == "" {
		config.RuntimeClass = "nvidia"
	}
	if len(config.NodeSelector) == 0 {
		config.NodeSelector = map[string]string{"gpu": "true"}
	}
	if config.ResourceLimit == "" {
		config.ResourceLimit = "1"
	}
	return &SelfManagedProvider{config: config}
}

func (p *SelfManagedProvider) Name() string                { return "self-managed" }
func (p *SelfManagedProvider) SupportedProducts() []string { return []string{"kubernetes"} }
func (p *SelfManagedProvider) DataRegion() string          { return "any" }
func (p *SelfManagedProvider) SupportedGPUTypes() []string {
	return []string{"any"} // Determined by what's available on the nodes
}

func (p *SelfManagedProvider) Submit(_ context.Context, _ GPUJob) (JobID, error) {
	return "", fmt.Errorf("self-managed provider does not use Submit; use direct K8s workload creation")
}

func (p *SelfManagedProvider) Status(_ context.Context, _ JobID) (JobStatus, error) {
	return JobStatus{}, fmt.Errorf("self-managed provider does not use Status; observe K8s workload directly")
}

func (p *SelfManagedProvider) Cancel(_ context.Context, _ JobID) error {
	return fmt.Errorf("self-managed provider does not use Cancel; delete K8s workload directly")
}

func (p *SelfManagedProvider) EstimateCost(_ GPUJob) (Cost, error) {
	return Cost{Amount: 0, Currency: "USD", Unit: "self-managed"}, nil
}

// Config returns the provider configuration for use by the resource builder.
func (p *SelfManagedProvider) Config() SelfManagedProviderConfig {
	return p.config
}
