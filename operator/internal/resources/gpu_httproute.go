package resources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

const (
	referenceGrantAPIVersion = "gateway.networking.k8s.io/v1beta1"
	referenceGrantKind       = "ReferenceGrant"
)

// DesiredGPUFunctionHTTPRoute builds the HTTPRoute for a GPU function.
// Instead of routing to the function Pod (which doesn't exist for external GPU),
// it routes to Function MS in platform-system via cross-namespace backendRef.
// GPU config is NOT passed via headers (IDOR prevention per D-220).
// Only routing metadata (function name, project ID) is set via RequestHeaderModifier.
func DesiredGPUFunctionHTTPRoute(fn *etalbaasv1alpha1.Function, cfg config.OperatorConfig) *unstructured.Unstructured {
	namespace := fn.Namespace
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name
	hostname := projectID + ".api." + cfg.BaseDomain

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": gatewayAPIVersion,
			"kind":       httpRouteKind,
			"metadata": map[string]interface{}{
				"name":      "func-" + funcName,
				"namespace": namespace,
				"labels":    toUnstructuredLabels(functionLabels(projectID, funcName)),
			},
			"spec": map[string]interface{}{
				"parentRefs": []interface{}{
					map[string]interface{}{
						"name":        cfg.GatewayName,
						"namespace":   cfg.GatewayNamespace,
						"sectionName": "https-api-wildcard",
					},
				},
				"hostnames": []interface{}{hostname},
				"rules": []interface{}{
					map[string]interface{}{
						"matches": []interface{}{
							map[string]interface{}{
								"path": map[string]interface{}{
									"type":  "PathPrefix",
									"value": "/functions/" + funcName + "/invoke",
								},
							},
						},
						"filters": []interface{}{
							// RequestHeaderModifier: set routing metadata.
							// MUST use "set" (not "add") to prevent IDOR via header injection.
							map[string]interface{}{
								"type": "RequestHeaderModifier",
								"requestHeaderModifier": map[string]interface{}{
									"set": []interface{}{
										map[string]interface{}{
											"name":  "X-Function-Name",
											"value": funcName,
										},
										map[string]interface{}{
											"name":  "X-Project-ID",
											"value": projectID,
										},
									},
								},
							},
							// URLRewrite: strip the function-specific prefix.
							map[string]interface{}{
								"type": "URLRewrite",
								"urlRewrite": map[string]interface{}{
									"path": map[string]interface{}{
										"type":               "ReplacePrefixMatch",
										"replacePrefixMatch": "/gpu-invoke",
									},
								},
							},
						},
						"backendRefs": []interface{}{
							map[string]interface{}{
								"name":      "function",
								"namespace": cfg.PlatformNamespace,
								"port":      int64(8080),
							},
						},
					},
				},
			},
		},
	}
}

// DesiredGPUReferenceGrant builds a ReferenceGrant in the platform-system namespace
// that allows HTTPRoutes from a project namespace to reference the Function MS Service.
// Required for cross-namespace backendRef in Gateway API.
func DesiredGPUReferenceGrant(projectNamespace string, cfg config.OperatorConfig) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": referenceGrantAPIVersion,
			"kind":       referenceGrantKind,
			"metadata": map[string]interface{}{
				"name":      "gpu-route-from-" + projectNamespace,
				"namespace": cfg.PlatformNamespace,
				"labels": toUnstructuredLabels(map[string]string{
					LabelManagedBy: ManagedByValue,
					LabelPartOf:    PartOfValue,
					"etalbaas.io/type": "gpu-reference-grant",
				}),
			},
			"spec": map[string]interface{}{
				"from": []interface{}{
					map[string]interface{}{
						"group":     "gateway.networking.k8s.io",
						"kind":      "HTTPRoute",
						"namespace": projectNamespace,
					},
				},
				"to": []interface{}{
					map[string]interface{}{
						"group": "",
						"kind":  "Service",
						"name":  "function",
					},
				},
			},
		},
	}
}
