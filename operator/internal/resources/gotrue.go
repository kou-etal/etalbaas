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

// DesiredGoTrueDeployment builds the desired GoTrue Deployment.
// GoTrue is deployed automatically when PostgreSQL is enabled.
func DesiredGoTrueDeployment(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *appsv1.Deployment {
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	labels := ComponentLabels(projectID, userID, plan, "gotrue")
	selectorLabels := map[string]string{
		LabelComponent: "gotrue",
		LabelProjectID: projectID,
	}

	var replicas int32 = 1

	subdomain := project.Spec.Networking.Subdomain
	externalURL := "https://" + subdomain + ".api." + cfg.BaseDomain + "/auth"

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gotrue",
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
							Name:  "gotrue",
							Image: cfg.GoTrueImage,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(true),
								RunAsNonRoot:             boolPtr(true),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 9999, Protocol: corev1.ProtocolTCP},
							},
							Env: []corev1.EnvVar{
								// Basic settings
								{Name: "GOTRUE_API_HOST", Value: "0.0.0.0"},
								{Name: "GOTRUE_API_PORT", Value: "9999"},
								{Name: "API_EXTERNAL_URL", Value: externalURL},
								{Name: "GOTRUE_SITE_URL", Value: externalURL},
								{Name: "GOTRUE_DB_DRIVER", Value: "postgres"},
								{Name: "GOTRUE_DB_NAMESPACE", Value: "auth"},
								{Name: "GOTRUE_JWT_DEFAULT_GROUP_NAME", Value: "authenticated"},
								{Name: "GOTRUE_DISABLE_SIGNUP", Value: "false"},
								{Name: "GOTRUE_EXTERNAL_EMAIL_ENABLED", Value: "true"},
								{Name: "GOTRUE_MAILER_AUTOCONFIRM", Value: "true"},
								{Name: "GOTRUE_JWT_EXP", Value: "3600"},
								// DB connection: gotrue-db-url is db-app URI with ?search_path=auth appended.
								{
									Name: "GOTRUE_DB_DATABASE_URL",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "gotrue-db-url"},
											Key:                  "uri",
										},
									},
								},
								// RS256 settings (D-190 / D-191)
								// D-191: GOTRUE_JWT_SECRET is required for backwards compatibility but not used for verification.
								{
									Name: "GOTRUE_JWT_SECRET",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "project-jwt-secret"},
											Key:                  "secret",
										},
									},
								},
								// D-190: RS256 key pair for JWT signing.
								{
									Name: "GOTRUE_JWT_KEYS",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: "gotrue-jwt-keys"},
											Key:                  "jwk-set",
										},
									},
								},
								// JWT_METHODS=ASYMMETRIC: Prevents Key Confusion Attack (CVE-2016-5431).
								// Without this, GoTrue falls back to HS256 verification using GOTRUE_JWT_SECRET
								// when receiving HS256-signed tokens, defeating the purpose of RS256.
								{Name: "JWT_METHODS", Value: "ASYMMETRIC"},
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "tmp", MountPath: "/tmp"},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("256Mi"),
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt32(9999),
									},
								},
								InitialDelaySeconds: 5,
								PeriodSeconds:       10,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/health",
										Port: intstr.FromInt32(9999),
									},
								},
								InitialDelaySeconds: 15,
								PeriodSeconds:       20,
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "tmp",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
				},
			},
		},
	}
}

// DesiredGoTrueService builds the desired GoTrue Service.
func DesiredGoTrueService(project *etalbaasv1alpha1.Project) *corev1.Service {
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
			Name:      "gotrue",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "gotrue"),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				LabelComponent: "gotrue",
				LabelProjectID: projectID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       9999,
					TargetPort: intstr.FromInt32(9999),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}
