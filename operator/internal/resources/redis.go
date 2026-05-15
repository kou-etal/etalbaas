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

// DesiredRedisDeployment builds the desired Redis Deployment.
func DesiredRedisDeployment(project *etalbaasv1alpha1.Project, cfg config.OperatorConfig) *appsv1.Deployment {
	redis := project.Spec.Stack.Redis
	if redis == nil || !redis.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	labels := ComponentLabels(projectID, userID, plan, "redis")
	selectorLabels := map[string]string{
		LabelComponent: "redis",
		LabelProjectID: projectID,
	}

	var replicas int32 = 1

	// Phase 1: Redis is cache-only. Always disable persistence regardless of flag.
	args := []string{"redis-server", "--maxmemory", "128mb", "--maxmemory-policy", "allkeys-lru", "--save", ""}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
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
						RunAsUser:      int64Ptr(999), // redis user in official image
						RunAsGroup:     int64Ptr(999),
						SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
					},
					Containers: []corev1.Container{
						{
							Name:    "redis",
							Image:   cfg.RedisImage,
							Command: args,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: boolPtr(false),
								ReadOnlyRootFilesystem:   boolPtr(true),
								RunAsNonRoot:             boolPtr(true),
								Capabilities:             &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}},
							},
							Ports: []corev1.ContainerPort{
								{ContainerPort: 6379, Protocol: corev1.ProtocolTCP},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("50m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("200m"),
									corev1.ResourceMemory: resource.MustParse("192Mi"),
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									TCPSocket: &corev1.TCPSocketAction{
										Port: intstr.FromInt32(6379),
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

// DesiredRedisService builds the desired Redis Service.
func DesiredRedisService(project *etalbaasv1alpha1.Project) *corev1.Service {
	redis := project.Spec.Stack.Redis
	if redis == nil || !redis.Enabled {
		return nil
	}

	namespace := "project-" + project.Name
	projectID := project.Name
	userID := project.Labels[LabelUserID]
	plan := project.Spec.Plan

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "redis",
			Namespace: namespace,
			Labels:    ComponentLabels(projectID, userID, plan, "redis"),
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				LabelComponent: "redis",
				LabelProjectID: projectID,
			},
			Ports: []corev1.ServicePort{
				{
					Name:       "redis",
					Port:       6379,
					TargetPort: intstr.FromInt32(6379),
					Protocol:   corev1.ProtocolTCP,
				},
			},
		},
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}
