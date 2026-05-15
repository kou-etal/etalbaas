package resources

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// DesiredResourceQuota builds the desired ResourceQuota for a Project namespace.
func DesiredResourceQuota(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *corev1.ResourceQuota {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	quota := cfg.FreePlanQuota

	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-quota",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "resourcequota"),
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourceRequestsCPU:    resource.MustParse(quota.CPU),
				corev1.ResourceRequestsMemory: resource.MustParse(quota.Memory),
				corev1.ResourceLimitsCPU:      resource.MustParse(quota.CPU),
				corev1.ResourceLimitsMemory:   resource.MustParse(quota.Memory),
				corev1.ResourcePods:           *resource.NewQuantity(int64(quota.Pods), resource.DecimalSI),
			},
		},
	}
}

// DesiredLimitRange builds the desired LimitRange for a Project namespace.
func DesiredLimitRange(project *etalbaasv1alpha1.Project) *corev1.LimitRange {
	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-limits",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "limitrange"),
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type: corev1.LimitTypeContainer,
					Default: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("250m"),
						corev1.ResourceMemory: resource.MustParse("256Mi"),
					},
					DefaultRequest: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
			},
		},
	}
}
