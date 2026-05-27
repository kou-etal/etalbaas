package resources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// TLSRouteGVK returns the GroupVersionKind for Gateway API TLSRoute.
func TLSRouteGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1alpha2",
		Kind:    tlsRouteKind,
	}
}

const (
	tlsRouteAPIVersion = "gateway.networking.k8s.io/v1alpha2"
	tlsRouteKind       = "TLSRoute"
)

// DesiredTLSRoute builds the desired Gateway API TLSRoute for direct DB connections.
// Routes: {dbSubdomain}.db.{domain} → CloudNativePG Pooler service (TLS passthrough).
func DesiredTLSRoute(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *unstructured.Unstructured {
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	dbSubdomain := project.Spec.Networking.DbSubdomain
	if dbSubdomain == "" {
		dbSubdomain = project.Spec.Networking.Subdomain
	}
	hostname := dbSubdomain + ".db." + cfg.BaseDomain

	// Target the pooler if available, otherwise the primary service
	backendName := "db-rw"
	if pg.Pooler != nil && pg.Pooler.Enabled {
		backendName = "db-pooler-rw"
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": tlsRouteAPIVersion,
			"kind":       tlsRouteKind,
			"metadata": map[string]interface{}{
				"name":      "db-route",
				"namespace": namespace,
				"labels":    toUnstructuredLabels(ComponentLabels(projectID, userID, plan, "tlsroute")),
			},
			"spec": map[string]interface{}{
				"parentRefs": []interface{}{
					map[string]interface{}{
						"name":      cfg.GatewayName,
						"namespace": cfg.GatewayNamespace,
					},
				},
				"hostnames": []interface{}{hostname},
				"rules": []interface{}{
					map[string]interface{}{
						"backendRefs": []interface{}{
							map[string]interface{}{
								"name": backendName,
								"port": int64(5432),
							},
						},
					},
				},
			},
		},
	}
}
