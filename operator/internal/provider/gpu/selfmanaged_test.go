package gpu

import (
	"context"
	"testing"
)

func TestSelfManagedProvider_Name(t *testing.T) {
	p := NewSelfManagedProvider(SelfManagedProviderConfig{})
	if got := p.Name(); got != "self-managed" {
		t.Errorf("Name() = %q, want %q", got, "self-managed")
	}
}

func TestSelfManagedProvider_Defaults(t *testing.T) {
	p := NewSelfManagedProvider(SelfManagedProviderConfig{})
	cfg := p.Config()
	if cfg.RuntimeClass != "nvidia" {
		t.Errorf("RuntimeClass = %q, want %q", cfg.RuntimeClass, "nvidia")
	}
	if cfg.NodeSelector["gpu"] != "true" {
		t.Errorf("NodeSelector = %v, want {gpu: true}", cfg.NodeSelector)
	}
	if cfg.ResourceLimit != "1" {
		t.Errorf("ResourceLimit = %q, want %q", cfg.ResourceLimit, "1")
	}
}

func TestSelfManagedProvider_CustomConfig(t *testing.T) {
	p := NewSelfManagedProvider(SelfManagedProviderConfig{
		NodeSelector:  map[string]string{"accelerator": "nvidia-a100"},
		RuntimeClass:  "nvidia-a100",
		ResourceLimit: "nvidia.com/gpu: 2",
	})
	cfg := p.Config()
	if cfg.RuntimeClass != "nvidia-a100" {
		t.Errorf("RuntimeClass = %q, want %q", cfg.RuntimeClass, "nvidia-a100")
	}
	if cfg.NodeSelector["accelerator"] != "nvidia-a100" {
		t.Errorf("NodeSelector = %v", cfg.NodeSelector)
	}
	if cfg.ResourceLimit != "nvidia.com/gpu: 2" {
		t.Errorf("ResourceLimit = %q", cfg.ResourceLimit)
	}
}

func TestSelfManagedProvider_SubmitReturnsError(t *testing.T) {
	p := NewSelfManagedProvider(SelfManagedProviderConfig{})
	_, err := p.Submit(context.Background(), GPUJob{})
	if err == nil {
		t.Error("Submit() should return error for self-managed provider")
	}
}

func TestSelfManagedProvider_DataRegion(t *testing.T) {
	p := NewSelfManagedProvider(SelfManagedProviderConfig{})
	if got := p.DataRegion(); got != "any" {
		t.Errorf("DataRegion() = %q, want %q", got, "any")
	}
}

func TestSelfManagedProvider_EstimateCost(t *testing.T) {
	p := NewSelfManagedProvider(SelfManagedProviderConfig{})
	cost, err := p.EstimateCost(GPUJob{})
	if err != nil {
		t.Fatal(err)
	}
	if cost.Amount != 0 {
		t.Errorf("Amount = %f, want 0", cost.Amount)
	}
}
