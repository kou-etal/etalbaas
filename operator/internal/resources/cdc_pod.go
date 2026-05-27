package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// DesiredCDCDeployment builds the desired CDC Pod Deployment.
// CDC Pod reads WAL logical replication from the project's PostgreSQL and publishes
// change events to NATS JetStream (subject: events.{project_id}.{table}).
func DesiredCDCDeployment(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *appsv1.Deployment {
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	labels := ComponentLabels(projectID, userID, plan, "cdc")
	selectorLabels := map[string]string{
		LabelComponent: "cdc",
		LabelProjectID: projectID,
	}

	var replicas int32 = 1
	slotName := "cdc_" + projectID

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cdc",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Strategy: appsv1.DeploymentStrategy{
				// Only one CDC Pod per project to avoid duplicate WAL reads
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergeMaps(labels, selectorLabels),
				},
				Spec: corev1.PodSpec{
					AutomountServiceAccountToken: boolPtr(false),
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot:   boolPtr(true),
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{
						{
							Name:  "cdc",
							Image: cfg.CDCImage,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(true),
								RunAsNonRoot:             boolPtr(true),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "CDC_PROJECT_ID",
									Value: projectID,
								},
								{
									Name:  "CDC_SLOT_NAME",
									Value: slotName,
								},
								{
									Name:  "CDC_NATS_URL",
									Value: "nats://" + cfg.NATSEndpoint,
								},
								{
									Name:  "CDC_NATS_SUBJECT_PREFIX",
									Value: "events.database",
								},
								{
									Name: "CDC_DB_HOST",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "db-cdc",
											},
											Key: "host",
										},
									},
								},
								{
									Name: "CDC_DB_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "db-cdc",
											},
											Key: "username",
										},
									},
								},
								{
									Name: "CDC_DB_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "db-cdc",
											},
											Key: "password",
										},
									},
								},
								{
									Name: "CDC_DB_NAME",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "db-cdc",
											},
											Key: "dbname",
										},
									},
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
						},
					},
				},
			},
		},
	}
}
