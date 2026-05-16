package gpu

import (
	"testing"
)

func TestFactory_IsEnabled(t *testing.T) {
	f := NewFactory(GPUConfig{Enabled: false}, nil, nil, "")
	if f.IsEnabled() {
		t.Error("IsEnabled() = true, want false")
	}
	f = NewFactory(GPUConfig{Enabled: true}, nil, nil, "")
	if !f.IsEnabled() {
		t.Error("IsEnabled() = false, want true")
	}
}

func TestFactory_ProviderFor_GPUDisabled(t *testing.T) {
	f := NewFactory(GPUConfig{Enabled: false}, nil, nil, "")
	_, err := f.ProviderFor("runpod", "serverless")
	if err == nil {
		t.Error("expected error when GPU disabled")
	}
}

func TestFactory_ProviderFor_UnknownProvider(t *testing.T) {
	f := NewFactory(GPUConfig{Enabled: true}, nil, nil, "")
	_, err := f.ProviderFor("unknown", "")
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestFactory_ProviderFor_RunPodDisabled(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled: true,
		Providers: ProvidersConfig{
			RunPod: &RunPodConfig{Enabled: false},
		},
	}, nil, nil, "")

	_, err := f.ProviderFor("runpod", "serverless")
	if err == nil {
		t.Error("expected error when RunPod disabled")
	}
}

func TestFactory_ProviderFor_RunPodServerless(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled: true,
		Providers: ProvidersConfig{
			RunPod: &RunPodConfig{
				Enabled:      true,
				APIKeySecret: "runpod-creds",
			},
		},
	}, nil, nil, "platform-system")

	p, err := f.ProviderFor("runpod", "serverless")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "runpod" {
		t.Errorf("Name() = %q, want %q", p.Name(), "runpod")
	}
}

func TestFactory_ProviderFor_RunPodDefaultProduct(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled: true,
		Providers: ProvidersConfig{
			RunPod: &RunPodConfig{Enabled: true},
		},
	}, nil, nil, "")

	p, err := f.ProviderFor("runpod", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "runpod" {
		t.Errorf("Name() = %q, want %q", p.Name(), "runpod")
	}
}

func TestFactory_ProviderFor_RunPodUnsupportedProduct(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled: true,
		Providers: ProvidersConfig{
			RunPod: &RunPodConfig{Enabled: true},
		},
	}, nil, nil, "")

	_, err := f.ProviderFor("runpod", "pods")
	if err == nil {
		t.Error("expected error for unsupported RunPod product")
	}
}

func TestFactory_ProviderFor_SelfManaged(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled: true,
		Providers: ProvidersConfig{
			SelfManaged: &SelfManagedConfig{
				Enabled:       true,
				NodeSelector:  map[string]string{"gpu": "a100"},
				RuntimeClass:  "nvidia",
				ResourceLimit: "nvidia.com/gpu: 1",
			},
		},
	}, nil, nil, "")

	p, err := f.ProviderFor("self-managed", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "self-managed" {
		t.Errorf("Name() = %q, want %q", p.Name(), "self-managed")
	}

	sm, ok := p.(*SelfManagedProvider)
	if !ok {
		t.Fatal("expected *SelfManagedProvider")
	}
	if sm.Config().NodeSelector["gpu"] != "a100" {
		t.Errorf("NodeSelector = %v", sm.Config().NodeSelector)
	}
}

func TestFactory_ProviderFor_SelfManagedDisabled(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled: true,
		Providers: ProvidersConfig{
			SelfManaged: &SelfManagedConfig{Enabled: false},
		},
	}, nil, nil, "")

	_, err := f.ProviderFor("self-managed", "")
	if err == nil {
		t.Error("expected error when self-managed disabled")
	}
}

func TestFactory_ResolveProvider_DefaultProvider(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled:         true,
		DefaultProvider: "self-managed",
		Providers: ProvidersConfig{
			SelfManaged: &SelfManagedConfig{Enabled: true},
		},
	}, nil, nil, "")

	p, err := f.ResolveProvider("", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "self-managed" {
		t.Errorf("Name() = %q, want %q", p.Name(), "self-managed")
	}
}

func TestFactory_ResolveProvider_ExplicitOverridesDefault(t *testing.T) {
	f := NewFactory(GPUConfig{
		Enabled:         true,
		DefaultProvider: "self-managed",
		Providers: ProvidersConfig{
			RunPod:      &RunPodConfig{Enabled: true},
			SelfManaged: &SelfManagedConfig{Enabled: true},
		},
	}, nil, nil, "")

	p, err := f.ResolveProvider("runpod", "serverless")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name() != "runpod" {
		t.Errorf("Name() = %q, want %q", p.Name(), "runpod")
	}
}
