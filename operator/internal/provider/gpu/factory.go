package gpu

import (
	"fmt"
	"net/http"

	"k8s.io/client-go/kubernetes"
)

// Factory creates GPU Provider instances from configuration.
type Factory struct {
	config     GPUConfig
	httpClient *http.Client
	k8sClient  kubernetes.Interface
	namespace  string // namespace for API key Secrets
}

// NewFactory creates a new provider factory.
func NewFactory(config GPUConfig, httpClient *http.Client, k8sClient kubernetes.Interface, namespace string) *Factory {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Factory{
		config:     config,
		httpClient: httpClient,
		k8sClient:  k8sClient,
		namespace:  namespace,
	}
}

// IsEnabled returns true if GPU features are enabled.
func (f *Factory) IsEnabled() bool {
	return f.config.Enabled
}

// ProviderFor returns the appropriate Provider for the given provider name and product.
// It validates that the provider is enabled and the product is available.
func (f *Factory) ProviderFor(providerName, product string) (Provider, error) {
	if !f.config.Enabled {
		return nil, fmt.Errorf("GPU is disabled in platform configuration")
	}

	switch providerName {
	case "runpod":
		return f.buildRunPodProvider(product)
	case "self-managed":
		return f.buildSelfManagedProvider()
	default:
		return nil, fmt.Errorf("unsupported GPU provider: %s", providerName)
	}
}

// ResolveProvider resolves the provider for a function's GPU spec.
// If provider is empty, uses the default provider from config.
func (f *Factory) ResolveProvider(provider, product string) (Provider, error) {
	if provider == "" {
		provider = f.config.DefaultProvider
	}
	return f.ProviderFor(provider, product)
}

func (f *Factory) buildRunPodProvider(product string) (Provider, error) {
	if f.config.Providers.RunPod == nil || !f.config.Providers.RunPod.Enabled {
		return nil, fmt.Errorf("RunPod provider is not enabled")
	}

	switch product {
	case "serverless", "":
		return NewRunPodServerlessProvider(
			f.config.Providers.RunPod.APIKeySecret,
			f.httpClient,
			f.k8sClient,
			f.namespace,
		), nil
	default:
		return nil, fmt.Errorf("unsupported RunPod product: %s (Phase 1 supports serverless only)", product)
	}
}

func (f *Factory) buildSelfManagedProvider() (Provider, error) {
	if f.config.Providers.SelfManaged == nil || !f.config.Providers.SelfManaged.Enabled {
		return nil, fmt.Errorf("self-managed GPU provider is not enabled")
	}
	smCfg := f.config.Providers.SelfManaged
	return NewSelfManagedProvider(SelfManagedProviderConfig{
		NodeSelector:  smCfg.NodeSelector,
		RuntimeClass:  smCfg.RuntimeClass,
		ResourceLimit: smCfg.ResourceLimit,
	}), nil
}
