package resources

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
)

const (
	envoyGatewayAPIVersion         = "gateway.envoyproxy.io/v1alpha1"
	backendTrafficPolicyKind       = "BackendTrafficPolicy"
)

// BackendTrafficPolicyGVK returns the GVK for Envoy Gateway BackendTrafficPolicy.
func BackendTrafficPolicyGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "gateway.envoyproxy.io",
		Version: "v1alpha1",
		Kind:    backendTrafficPolicyKind,
	}
}

// DesiredGPUBackendTrafficPolicy builds a BackendTrafficPolicy that extends the
// Envoy Gateway timeout for GPU function HTTPRoutes.
// Default Envoy timeout is 15s, but RunPod /runsync cold start can take 60-120s.
// We set 300s (5 min) to accommodate worst-case cold starts.
func DesiredGPUBackendTrafficPolicy(fn *etalbaasv1alpha1.Function) *unstructured.Unstructured {
	namespace := fn.Namespace
	funcName := fn.Name
	projectID := fn.Spec.ProjectRef.Name

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": envoyGatewayAPIVersion,
			"kind":       backendTrafficPolicyKind,
			"metadata": map[string]interface{}{
				"name":      "gpu-timeout-" + funcName,
				"namespace": namespace,
				"labels": toUnstructuredLabels(map[string]string{
					LabelProjectID:     projectID,
					"etalbaas.io/function": funcName,
					"etalbaas.io/type":     "gpu-timeout",
					LabelManagedBy:     ManagedByValue,
					LabelPartOf:        PartOfValue,
				}),
			},
			"spec": map[string]interface{}{
				"targetRefs": []interface{}{
					map[string]interface{}{
						"group": "gateway.networking.k8s.io",
						"kind":  "HTTPRoute",
						"name":  "func-" + funcName,
					},
				},
				"timeout": map[string]interface{}{
					"http": map[string]interface{}{
						"requestTimeout": "300s",
						"idleTimeout":    "300s",
					},
				},
			},
		},
	}
}
