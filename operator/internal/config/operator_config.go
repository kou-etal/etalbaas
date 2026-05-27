package config

import (
	"time"

	"github.com/kou-etal/etalbaas/operator/internal/provider/gpu"
)

// OperatorConfig holds the configuration for the operator.
type OperatorConfig struct {
	// PlatformNamespace is the namespace where platform components run.
	PlatformNamespace string

	// BaseDomain is the base domain for project endpoints (e.g., "yourdomain.com").
	BaseDomain string

	// RegistryEndpoint is the internal container registry endpoint (Zot).
	RegistryEndpoint string

	// NATSEndpoint is the NATS JetStream endpoint.
	NATSEndpoint string

	// GatewayName is the name of the Gateway resource for HTTPRoute/TLSRoute parentRefs.
	GatewayName string

	// GatewayNamespace is the namespace of the Gateway resource.
	GatewayNamespace string

	// NATSMonitoringEndpoint is the NATS monitoring HTTP endpoint for KEDA scalers.
	NATSMonitoringEndpoint string

	// NATSSidecarImage is the container image for the NATS-to-HTTP sidecar.
	NATSSidecarImage string

	// CDCImage is the container image for the CDC Pod.
	CDCImage string

	// PostgRESTImage is the container image for PostgREST.
	PostgRESTImage string

	// RedisImage is the container image for Redis.
	RedisImage string

	// KanikoImage is the container image for Kaniko builds.
	KanikoImage string

	// BuildTimeout is the maximum duration for a Kaniko build.
	BuildTimeout time.Duration

	// FreePlanQuota defines resource quotas for the free plan.
	FreePlanQuota PlanQuota

	// GPU holds GPU provider configuration. Parsed from Helm values.
	GPU gpu.GPUConfig

	// DispatcherImage is the container image for the GPU dispatcher.
	DispatcherImage string

	// PostgresMetaImage is the container image for supabase/postgres-meta.
	PostgresMetaImage string

	// MaxTotalProjects is the maximum number of projects across the entire cluster.
	// When reached, new projects will be set to "Queued" phase and requeued.
	// 0 means unlimited.
	MaxTotalProjects int

	// MaxConcurrentBuilds is the maximum number of concurrent Kaniko build jobs
	// per project. When reached, new builds will be requeued.
	MaxConcurrentBuilds int

	// RegistryInsecure allows Kaniko to push to an insecure (HTTP) registry.
	// Set to true for local/dev registries. Defaults to false in production.
	RegistryInsecure bool

	// SandboxRuntimeClass is the Kubernetes RuntimeClass for sandboxed workloads
	// (build jobs, function deployments, GPU dispatchers).
	// Set to "gvisor" in production. When empty, RuntimeClassName is not set on pods.
	SandboxRuntimeClass string

	// GoTrueImage is the container image for GoTrue auth server.
	GoTrueImage string

	// JWTSecret is the symmetric JWT secret shared with GoTrue.
	// Replicated into each project namespace as "project-jwt-secret" for PostgREST.
	JWTSecret string
}

// PlanQuota defines resource quotas for a billing plan.
type PlanQuota struct {
	CPU    string
	Memory string
	Pods   int32
}

// DefaultConfig returns the default operator configuration.
func DefaultConfig() OperatorConfig {
	return OperatorConfig{
		PlatformNamespace: "platform-system",
		BaseDomain:        "yourdomain.com",
		RegistryEndpoint:  "zot.platform-system.svc:5000",
		NATSEndpoint:              "nats.platform-system.svc:4222",
		NATSMonitoringEndpoint:    "nats.platform-system.svc:8222",
		NATSSidecarImage:          "etalbaas/nats-sidecar:latest",
		GatewayName:       "etalbaas-gateway",
		GatewayNamespace:  "platform-system",
		CDCImage:          "etalbaas/cdc-pod:latest",
		PostgRESTImage:    "postgrest/postgrest:v12.2.0",
		RedisImage:        "redis:7-alpine",
		KanikoImage:       "gcr.io/kaniko-project/executor:latest",
		BuildTimeout:      15 * time.Minute,
		FreePlanQuota: PlanQuota{
			CPU:    "2",
			Memory: "4Gi",
			Pods:   20,
		},
		MaxTotalProjects:    50,
		MaxConcurrentBuilds: 2,
		GPU:                 gpu.DefaultGPUConfig(),
		DispatcherImage:     "etalbaas/gpu-dispatcher:latest",
		PostgresMetaImage:   "supabase/postgres-meta:v0.91.0",
		GoTrueImage:         "supabase/gotrue:v2.164.0",
		SandboxRuntimeClass: "gvisor",
	}
}
