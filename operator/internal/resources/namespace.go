package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
)

// DesiredNamespace builds the desired Namespace for a Project.
// It includes Pod Security Standards labels for the restricted profile.
func DesiredNamespace(project *etalbaasv1alpha1.Project) *corev1.Namespace {
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	labels := CommonLabels(projectID, userID, plan)
	// Pod Security Standards: restricted profile
	labels["pod-security.kubernetes.io/enforce"] = "restricted"
	labels["pod-security.kubernetes.io/enforce-version"] = "latest"
	labels["pod-security.kubernetes.io/warn"] = "restricted"
	labels["pod-security.kubernetes.io/warn-version"] = "latest"
	labels["pod-security.kubernetes.io/audit"] = "restricted"
	labels["pod-security.kubernetes.io/audit-version"] = "latest"

	return &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   "project-" + projectID,
			Labels: labels,
		},
	}
}
