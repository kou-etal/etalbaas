package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// DesiredPostgresMetaDeployment builds the desired postgres-meta Deployment.
// postgres-meta is deployed automatically when PostgreSQL is enabled.
func DesiredPostgresMetaDeployment(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *appsv1.Deployment {
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	labels := ComponentLabels(projectID, userID, plan, "postgres-meta")
	selectorLabels := map[string]string{
		LabelComponent: "postgres-meta",
		LabelProjectID: projectID,
	}

	var replicas int32 = 1

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-meta",
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
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
						RunAsUser:      int64Ptr(1000),
						RunAsGroup:     int64Ptr(1000),
						FSGroup:        int64Ptr(1000),
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{
						{
							Name:  "postgres-meta",
							Image: cfg.PostgresMetaImage,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								RunAsNonRoot:             boolPtr(true),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 8080, Protocol: corev1.ProtocolTCP},
							},
							Env: []corev1.EnvVar{
								{Name: "PG_META_PORT", Value: "8080"},
								{Name: "PG_META_DB_HOST", Value: "db-rw"},
								{Name: "PG_META_DB_PORT", Value: "5432"},
								{Name: "PG_META_DB_NAME", Value: "postgres"},
								{
									Name: "PG_META_DB_USER",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "db-app",
											},
											Key: "username",
										},
									},
								},
								{
									Name: "PG_META_DB_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "db-app",
											},
											Key: "password",
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
									corev1.ResourceCPU:    resource.MustParse("100m"),
									corev1.ResourceMemory: resource.MustParse("128Mi"),
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt32(8080),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
						},
					},
				},
			},
		},
	}
}

// DesiredPostgresMetaService builds the desired postgres-meta Service.
func DesiredPostgresMetaService(project *etalbaasv1alpha1.Project) *corev1.Service {
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgres-meta",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "postgres-meta"),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				LabelComponent: "postgres-meta",
				LabelProjectID: projectID,
			},
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
