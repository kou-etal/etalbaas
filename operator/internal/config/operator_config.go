package config

import "time"

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
		NATSEndpoint:      "nats.platform-system.svc:4222",
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
	}
}
