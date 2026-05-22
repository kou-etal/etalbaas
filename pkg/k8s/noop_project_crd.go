package k8s

import "context"

// NoopProjectCRDManager is a no-op implementation of ProjectCRDManager
// used when K8s integration is disabled.
type NoopProjectCRDManager struct{}

func (NoopProjectCRDManager) CreateOrUpdate(_ context.Context, _ ProjectCRDParams) error { return nil }
func (NoopProjectCRDManager) Delete(_ context.Context, _ string) error                   { return nil }
