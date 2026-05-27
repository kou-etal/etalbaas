package resources

import (
	"fmt"

	autoscalingv2 "k8s.io/api/autoscaling/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/natsadmin"
)

// KEDAScaledObjectGVK returns the GroupVersionKind for KEDA ScaledObject.
func KEDAScaledObjectGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "keda.sh",
		Version: "v1alpha1",
		Kind:    "ScaledObject",
	}
}

// KEDAHTTPScaledObjectGVK returns the GroupVersionKind for KEDA HTTP add-on HTTPScaledObject.
func KEDAHTTPScaledObjectGVK() schema.GroupVersionKind {
	return schema.GroupVersionKind{
		Group:   "http.keda.sh",
		Version: "v1alpha1",
		Kind:    "HTTPScaledObject",
	}
}

// DesiredKEDAScaledObject builds the desired KEDA ScaledObject for a heavy-deployment Function
// with an event-driven trigger (DatabaseChange or ObjectStorage). Uses nats-jetstream trigger
// based on consumer pending count.
// Returns nil if the function is not heavy-deployment or has no event trigger.
func DesiredKEDAScaledObject(fn *etalbaasv1alpha1.Function, cfg config.OperatorConfig) *unstructured.Unstructured {
	if fn.Spec.Kind != etalbaasv1alpha1.FunctionKindHeavyDeployment {
		return nil
	}

	if !HasEventTrigger(fn) {
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
				"pollingInterval": int64(15),
				"cooldownPeriod":  cooldownPeriod,
				"minReplicaCount": int64(0),
				"maxReplicaCount": maxReplicas,
				"triggers": []interface{}{
					map[string]interface{}{
						"type": "nats-jetstream",
						"metadata": map[string]interface{}{
							"natsServerMonitoringEndpoint": cfg.NATSMonitoringEndpoint,
							"account":                      "$G",
							"stream":                       natsadmin.StreamName(projectID),
							"consumer":                     natsadmin.ConsumerName(funcName),
							"lagThreshold":                 "10",
						},
					},
				},
			},
		},
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

// DesiredHTTPScaledObject builds the desired KEDA HTTP add-on HTTPScaledObject
// for heavy-deployment Functions with Http triggers (no DatabaseChange).
// Enables 0→N scaling based on HTTP request rate.
// Returns nil for non-heavy-deployment, heavy-job, heavy-deployment with DatabaseChange,
// or functions without Http trigger.
func DesiredHTTPScaledObject(fn *etalbaasv1alpha1.Function, cfg config.OperatorConfig) *unstructured.Unstructured {
	if fn.Spec.Kind != etalbaasv1alpha1.FunctionKindHeavyDeployment {
		return nil
	}

	// Heavy-deployment with event triggers uses ScaledObject (nats-jetstream), not HTTPScaledObject.
	if HasEventTrigger(fn) {
		return nil
	}

	if !hasHTTPTrigger(fn) {
		return nil
	}

	namespace := fn.Namespace
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	maxReplicas := int64(10)
	if fn.Spec.Execution != nil && fn.Spec.Execution.Deployment != nil {
		if fn.Spec.Execution.Deployment.MaxReplicas != nil {
			maxReplicas = int64(*fn.Spec.Execution.Deployment.MaxReplicas)
		}
	}

	hostname := fmt.Sprintf("%s.api.%s", projectID, cfg.BaseDomain)
	pathPrefix := fmt.Sprintf("/functions/%s/invoke", funcName)

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "http.keda.sh/v1alpha1",
			"kind":       "HTTPScaledObject",
			"metadata": map[string]interface{}{
				"name":      "func-" + funcName,
				"namespace": namespace,
				"labels":    toUnstructuredLabels(functionLabels(projectID, funcName)),
			},
			"spec": map[string]interface{}{
				"hosts":        []interface{}{hostname},
				"pathPrefixes": []interface{}{pathPrefix},
				"scaleTargetRef": map[string]interface{}{
					"deployment": "func-" + funcName,
					"service":    "func-" + funcName,
					"port":       int64(8080),
				},
				"replicas": map[string]interface{}{
					"min": int64(0),
					"max": maxReplicas,
				},
				"scalingMetric": map[string]interface{}{
					"requestRate": map[string]interface{}{
						"targetValue": int64(100),
					},
				},
			},
		},
	}
}

// HasDatabaseChangeTrigger returns true if the function has any valid DatabaseChange trigger.
func HasDatabaseChangeTrigger(fn *etalbaasv1alpha1.Function) bool {
	for _, t := range fn.Spec.Triggers {
		if t.Type == "DatabaseChange" && t.DatabaseChange != nil {
			return true
		}
	}
	return false
}

// HasObjectStorageTrigger returns true if the function has any valid ObjectStorage trigger.
func HasObjectStorageTrigger(fn *etalbaasv1alpha1.Function) bool {
	for _, t := range fn.Spec.Triggers {
		if t.Type == "ObjectStorage" && t.ObjectStorage != nil {
			return true
		}
	}
	return false
}

// HasEventTrigger returns true if the function has any event-driven trigger (DB or Storage).
func HasEventTrigger(fn *etalbaasv1alpha1.Function) bool {
	return HasDatabaseChangeTrigger(fn) || HasObjectStorageTrigger(fn)
}

// hasHTTPTrigger is defined in function_service.go
