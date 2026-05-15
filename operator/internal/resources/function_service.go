package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/intstr"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// DesiredFunctionService builds the desired Service for a Function (HTTP trigger).
// Returns nil for heavy-job functions (no persistent service needed).
func DesiredFunctionService(fn *etalbaasv1alpha1.Function) *corev1.Service {
	if fn.Spec.Kind == etalbaasv1alpha1.FunctionKindHeavyJob {
		return nil
	}

	// Only create service if there's an HTTP trigger
	if !hasHTTPTrigger(fn) {
		return nil
	}

	namespace := fn.Namespace
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "func-" + funcName,
			Namespace: namespace,
			Labels:    functionLabels(projectID, funcName),
		},
		Spec: corev1.ServiceSpec{
			Selector: functionSelectorLabels(projectID, funcName),
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       8080,
					TargetPort: intstr.FromInt32(8080),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// DesiredFunctionHTTPRoute builds the desired HTTPRoute for a Function's HTTP trigger.
func DesiredFunctionHTTPRoute(fn *etalbaasv1alpha1.Function, cfg config.OperatorConfig) *unstructured.Unstructured {
	if fn.Spec.Kind == etalbaasv1alpha1.FunctionKindHeavyJob {
		return nil
	}

	if !hasHTTPTrigger(fn) {
		return nil
	}

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
									"value": "/functions/" + funcName + "/invoke",
								},
							},
						},
						"backendRefs": []interface{}{
							map[string]interface{}{
								"name": "func-" + funcName,
								"port": int64(8080),
							},
						},
					},
				},
			},
		},
	}
}

func hasHTTPTrigger(fn *etalbaasv1alpha1.Function) bool {
	for _, trigger := range fn.Spec.Triggers {
		if trigger.Type == "Http" {
			return true
		}
	}
	return false
}
