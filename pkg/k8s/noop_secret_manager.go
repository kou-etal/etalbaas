package k8s

import "context"

// NoopSecretManager is a SecretManager that does nothing.
// Used when K8S_ENABLED=false (e.g., E2E tests without a k8s cluster).
type NoopSecretManager struct{}

func (NoopSecretManager) CreateSecret(_ context.Context, _, _, _, _ string) error { return nil }
func (NoopSecretManager) UpdateSecret(_ context.Context, _, _, _, _ string) error { return nil }
func (NoopSecretManager) DeleteSecret(_ context.Context, _, _ string) error       { return nil }
