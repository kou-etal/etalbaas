package resources

import (
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
)

// KEDAScaledObjectGVK returns the GroupVersionKind for KEDA ScaledObject.
func KEDAScaledObjectGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "keda.sh",
		Version: "v1alpha1",
		Kind:    "ScaledObject",
	}
}

// DesiredHPA builds the desired HorizontalPodAutoscaler for a light-deployment Function.
// Returns nil for non-light-deployment functions.
func DesiredHPA(fn *etalbaasv1alpha1.Function) *autoscalingv2.HorizontalPodAutoscaler {
	if fn.Spec.Kind != etalbaasv1alpha1.FunctionKindLightDeployment {
		return nil
	}

	namespace := fn.Namespace
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	minReplicas := int32(1)
	maxReplicas := int32(10)

	if fn.Spec.Execution != nil && fn.Spec.Execution.Deployment != nil {
		if fn.Spec.Execution.Deployment.MinReplicas != nil && *fn.Spec.Execution.Deployment.MinReplicas >= 1 {
			minReplicas = *fn.Spec.Execution.Deployment.MinReplicas
		}
		if fn.Spec.Execution.Deployment.MaxReplicas != nil {
			maxReplicas = *fn.Spec.Execution.Deployment.MaxReplicas
		}
	}

	cpuTarget := int32(70)

	return &autoscalingv2.HorizontalPodAutoscaler{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "func-" + funcName,
			Namespace: namespace,
			Labels:    functionLabels(projectID, funcName),
		},
		Spec: autoscalingv2.HorizontalPodAutoscalerSpec{
			ScaleTargetRef: autoscalingv2.CrossVersionObjectReference{
				APIVersion: "apps/v1",
				Kind:       "Deployment",
				Name:       "func-" + funcName,
			},
			MinReplicas: &minReplicas,
			MaxReplicas: maxReplicas,
			Metrics: []autoscalingv2.MetricSpec{
				{
					Type: autoscalingv2.ResourceMetricSourceType,
					Resource: &autoscalingv2.ResourceMetricSource{
						Name: "cpu",
						Target: autoscalingv2.MetricTarget{
							Type:               autoscalingv2.UtilizationMetricType,
							AverageUtilization: &cpuTarget,
						},
					},
				},
			},
		},
	}
}

// DesiredKEDAScaledObject builds the desired KEDA ScaledObject for a heavy-deployment Function.
// Returns nil for non-heavy-deployment functions.
func DesiredKEDAScaledObject(fn *etalbaasv1alpha1.Function) *unstructured.Unstructured {
	if fn.Spec.Kind != etalbaasv1alpha1.FunctionKindHeavyDeployment {
		return nil
	}

	namespace := fn.Namespace
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	maxReplicas := int64(10)
	cooldownPeriod := int64(300) // 5 minutes

	if fn.Spec.Execution != nil && fn.Spec.Execution.Deployment != nil {
		if fn.Spec.Execution.Deployment.MaxReplicas != nil {
			maxReplicas = int64(*fn.Spec.Execution.Deployment.MaxReplicas)
		}
		if fn.Spec.Execution.Deployment.ScaleDownDelay != nil {
			cooldownPeriod = int64(*fn.Spec.Execution.Deployment.ScaleDownDelay)
		}
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "keda.sh/v1alpha1",
			"kind":       "ScaledObject",
			"metadata": map[string]interface{}{
				"name":      "func-" + funcName,
				"namespace": namespace,
				"labels":    toUnstructuredLabels(functionLabels(projectID, funcName)),
			},
			"spec": map[string]interface{}{
				"scaleTargetRef": map[string]interface{}{
					"name": "func-" + funcName,
				},
				"pollingInterval":  int64(15),
				"cooldownPeriod":   cooldownPeriod,
				"minReplicaCount":  int64(0),
				"maxReplicaCount":  maxReplicas,
				"triggers": []interface{}{
					map[string]interface{}{
						"type": "kubernetes-workload",
						"metadata": map[string]interface{}{
							"podSelector": "etalbaas.io/function=" + funcName,
							"value":       "1",
						},
					},
				},
			},
		},
	}
}
