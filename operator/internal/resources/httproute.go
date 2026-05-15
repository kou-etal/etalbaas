package resources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// HTTPRouteGVK returns the GroupVersionKind for Gateway API HTTPRoute.
func HTTPRouteGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    httpRouteKind,
	}
}

const (
	gatewayAPIVersion    = "gateway.networking.k8s.io/v1"
	httpRouteKind        = "HTTPRoute"
)

// DesiredHTTPRoute builds the desired Gateway API HTTPRoute for the project's REST API.
// Routes: {subdomain}.api.{domain} → PostgREST service
func DesiredHTTPRoute(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *unstructured.Unstructured {
	pr := project.Spec.Stack.PostgREST
	if pr == nil || !pr.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan
	hostname := project.Spec.Networking.Subdomain + ".api." + cfg.BaseDomain

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gatewayAPIVersion,
			"kind":       httpRouteKind,
			"metadata": map[string]interface{}{
				"name":      "api-route",
				"namespace": namespace,
				"labels":    toUnstructuredLabels(ComponentLabels(projectID, userID, plan, "httproute")),
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
						"matches": []interface{}{
							map[string]interface{}{
								"path": map[string]interface{}{
									"type":  "PathPrefix",
									"value": "/rest",
								},
							},
						},
						"backendRefs": []interface{}{
							map[string]interface{}{
								"name": "postgrest",
								"port": int64(3000),
							},
						},
					},
				},
			},
		},
	}
}
