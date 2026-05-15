package build

import (
	"crypto/sha256"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// KanikoBuildJob creates a Kaniko Job spec for building a Function's container image.
// The Job is created in the platform-system namespace for security isolation.
func KanikoBuildJob(
	fn *etalbaasv1alpha1.Function,
	cfg config.OperatorConfig,
) *batchv1.Job {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name
	generation := fn.Generation
	shortHash := shortSha(fmt.Sprintf("%s-%s-%d", projectID, funcName, generation))
	imageTag := fmt.Sprintf("%d-%s", generation, shortHash)
	destination := fmt.Sprintf("%s/project-%s/%s:%s", cfg.RegistryEndpoint, projectID, funcName, imageTag)

	var backoffLimit int32 = 1
	activeDeadlineSeconds := int64(cfg.BuildTimeout.Seconds())
	runtimeClassName := "gvisor"

	labels := map[string]string{
		"etalbaas.io/project-id":    projectID,
		"etalbaas.io/function":      funcName,
		"etalbaas.io/build":         "true",
		"etalbaas.io/generation":    fmt.Sprintf("%d", generation),
		"app.kubernetes.io/part-of": "etalbaas",
	}

	// Build init container based on source type
	initContainers := buildInitContainers(fn)

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("build-%s-%s-%s", projectID, funcName, shortHash),
			Namespace: cfg.PlatformNamespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:          &backoffLimit,
			ActiveDeadlineSeconds: &activeDeadlineSeconds,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					RuntimeClassName:             &runtimeClassName,
					RestartPolicy:                corev1.RestartPolicyNever,
					AutomountServiceAccountToken: boolPtr(false),
					SecurityContext: &corev1.PodSecurityContext{
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name:  "kaniko",
							Image: cfg.KanikoImage,
							Args: []string{
								"--dockerfile=/workspace/Dockerfile",
								"--context=/workspace/source",
								"--destination=" + destination,
								"--cache=true",
								"--cache-repo=" + cfg.RegistryEndpoint + "/cache",
								"--insecure",
								"--skip-tls-verify",
							},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "workspace", MountPath: "/workspace"},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("500m"),
									corev1.ResourceMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("2"),
									corev1.ResourceMemory: resource.MustParse("2Gi"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "workspace",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "dockerfile",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: fmt.Sprintf("build-%s-%s-dockerfile", projectID, funcName),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

// BuildDockerfileConfigMap creates a ConfigMap containing the generated Dockerfile.
func BuildDockerfileConfigMap(
	fn *etalbaasv1alpha1.Function,
	dockerfile string,
	cfg config.OperatorConfig,
) *corev1.ConfigMap {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("build-%s-%s-dockerfile", projectID, funcName),
			Namespace: cfg.PlatformNamespace,
			Labels: map[string]string{
				"etalbaas.io/project-id":  projectID,
				"etalbaas.io/function":    funcName,
				"etalbaas.io/build":       "true",
				"app.kubernetes.io/part-of": "etalbaas",
			},
		},
		Data: map[string]string{
			"Dockerfile": dockerfile,
		},
	}
}

// ImageDestination returns the image reference for a built function.
func ImageDestination(fn *etalbaasv1alpha1.Function, cfg config.OperatorConfig) string {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name
	generation := fn.Generation
	shortHash := shortSha(fmt.Sprintf("%s-%s-%d", projectID, funcName, generation))
	imageTag := fmt.Sprintf("%d-%s", generation, shortHash)
	return fmt.Sprintf("%s/project-%s/%s:%s", cfg.RegistryEndpoint, projectID, funcName, imageTag)
}

// Pinned image versions for reproducibility and supply-chain safety.
const (
	alpineGitImage = "alpine/git:v2.45.2"
	alpineImage    = "alpine:3.21"
)

// initContainerSecurityContext returns the SecurityContext for init containers.
// Init containers run simple copy/clone operations and don't need elevated privileges.
func initContainerSecurityContext() *corev1.SecurityContext {
	return &corev1.SecurityContext{
		AllowPrivilegeEscalation: boolPtr(false),
		ReadOnlyRootFilesystem:   boolPtr(false), // needs to write to /tmp
		RunAsNonRoot:             boolPtr(true),
		RunAsUser:                int64Ptr(65532),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}
}

func buildInitContainers(fn *etalbaasv1alpha1.Function) []corev1.Container {
	source := fn.Spec.Source
	sc := initContainerSecurityContext()

	switch source.Type {
	case "git":
		if source.Git == nil {
			return []corev1.Container{dockerfileCopyInitContainer()}
		}
		git := source.Git
		ref := "HEAD"
		if git.Ref != "" {
			ref = git.Ref
		}
		// Use git clone with explicit argument separation to avoid shell injection.
		cloneArgs := []string{
			"git", "clone", "--depth", "1", "--branch", ref, git.Repo, "/tmp/repo",
		}
		copyCmd := "cp -r /tmp/repo/. /workspace/source/"
		if git.Path != "" && git.Path != "/" {
			copyCmd = fmt.Sprintf("cp -r /tmp/repo/%s/. /workspace/source/", git.Path)
		}
		return []corev1.Container{
			{
				Name:            "source-fetch",
				Image:           alpineGitImage,
				Command:         cloneArgs,
				SecurityContext: sc,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "workspace", MountPath: "/workspace"},
				},
			},
			{
				Name:            "source-copy",
				Image:           alpineImage,
				Command:         []string{"sh", "-c"},
				Args:            []string{"mkdir -p /workspace/source && " + copyCmd},
				SecurityContext: sc,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "workspace", MountPath: "/workspace"},
				},
			},
			dockerfileCopyInitContainer(),
		}
	case "zip", "inline":
		return []corev1.Container{
			{
				Name:            "source-fetch",
				Image:           alpineImage,
				Command:         []string{"sh", "-c"},
				Args:            []string{"mkdir -p /workspace/source && cp -r /source-data/. /workspace/source/"},
				SecurityContext: sc,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "workspace", MountPath: "/workspace"},
					{Name: "source-data", MountPath: "/source-data"},
				},
			},
			dockerfileCopyInitContainer(),
		}
	default:
		return []corev1.Container{dockerfileCopyInitContainer()}
	}
}

func dockerfileCopyInitContainer() corev1.Container {
	return corev1.Container{
		Name:            "copy-dockerfile",
		Image:           alpineImage,
		Command:         []string{"sh", "-c"},
		Args:            []string{"mkdir -p /workspace/source && cp /dockerfile-data/Dockerfile /workspace/Dockerfile"},
		SecurityContext: initContainerSecurityContext(),
		VolumeMounts: []corev1.VolumeMount{
			{Name: "workspace", MountPath: "/workspace"},
			{Name: "dockerfile", MountPath: "/dockerfile-data"},
		},
	}
}

func int64Ptr(i int64) *int64 {
	return &i
}

func boolPtr(b bool) *bool {
	return &b
}

func shortSha(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:4])
}
