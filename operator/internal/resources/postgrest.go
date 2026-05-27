package resources

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// DesiredPostgRESTDeployment builds the desired PostgREST Deployment.
func DesiredPostgRESTDeployment(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *appsv1.Deployment {
	pr := project.Spec.Stack.PostgREST
	if pr == nil || !pr.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	labels := ComponentLabels(projectID, userID, plan, "postgrest")
	selectorLabels := map[string]string{
		LabelComponent: "postgrest",
		LabelProjectID: projectID,
	}

	var replicas int32 = 1

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgrest",
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
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{
						{
							Name:  "postgrest",
							Image: cfg.PostgRESTImage,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(true),
								RunAsNonRoot:             boolPtr(true),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 3000, Protocol: corev1.ProtocolTCP},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "postgrest-config",
										},
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "PGRST_DB_URI",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "db-app",
											},
											Key: "uri",
										},
									},
								},
								{
									Name: "PGRST_JWT_SECRET",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "jwt-verification-key",
											},
											Key: "jwk",
										},
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/",
										Port: intstr.FromInt32(3000),
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

// DesiredPostgRESTService builds the desired PostgREST Service.
func DesiredPostgRESTService(project *etalbaasv1alpha1.Project) *corev1.Service {
	pr := project.Spec.Stack.PostgREST
	if pr == nil || !pr.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgrest",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "postgrest"),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				LabelComponent: "postgrest",
				LabelProjectID: projectID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       3000,
					TargetPort: intstr.FromInt32(3000),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

// DesiredPostgRESTConfigMap builds the desired PostgREST ConfigMap.
func DesiredPostgRESTConfigMap(project *etalbaasv1alpha1.Project) *corev1.ConfigMap {
	pr := project.Spec.Stack.PostgREST
	if pr == nil || !pr.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	anonRole := "anon"
	if pr.AnonRole != "" {
		anonRole = pr.AnonRole
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "postgrest-config",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "postgrest"),
		},
		Data: map[string]string{
			"PGRST_DB_ANON_ROLE":  anonRole,
			"PGRST_DB_SCHEMAS":    "public",
			"PGRST_SERVER_PORT":   "3000",
			"PGRST_OPENAPI_MODE":  "ignore-privileges",
		},
	}
}

func mergeMaps(base, overlay map[string]string) map[string]string {
	result := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		result[k] = v
	}
	for k, v := range overlay {
		result[k] = v
	}
	return result
}
