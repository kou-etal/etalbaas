package build

import (
	"crypto/sha256"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
)

// validGitPathRe allows only safe characters in git subdirectory paths.
var validGitPathRe = regexp.MustCompile(`^[a-zA-Z0-9._/\-]+$`)

// validGitRefRe allows only safe characters in git branch/tag references.
var validGitRefRe = regexp.MustCompile(`^[a-zA-Z0-9._/\-]+$`)

// validateGitPath rejects paths that could lead to shell injection.
func validateGitPath(p string) error {
	if strings.Contains(p, "..") {
		return fmt.Errorf("git path must not contain '..': %s", p)
	}
	if !validGitPathRe.MatchString(p) {
		return fmt.Errorf("git path contains invalid characters: %s", p)
	}
	return nil
}

// validateGitRepo ensures the repo URL is HTTPS and does not point to internal networks.
func validateGitRepo(repo string) error {
	u, err := url.Parse(repo)
	if err != nil {
		return fmt.Errorf("invalid git repo URL: %w", err)
	}
	if u.Scheme != "https" {
		return fmt.Errorf("git repo must use https:// scheme, got %q", u.Scheme)
	}
	hostname := u.Hostname()
	if hostname == "" {
		return fmt.Errorf("git repo URL has no hostname")
	}

	// Reject well-known internal/metadata hostnames.
	blocked := []string{
		"metadata.google.internal",
		"169.254.169.254",
		"metadata.internal",
		"kubernetes.default",
		"kubernetes.default.svc",
		"localhost",
		"127.0.0.1",
		"[::1]",
	}
	lower := strings.ToLower(hostname)
	for _, b := range blocked {
		if lower == b {
			return fmt.Errorf("git repo URL points to blocked host: %s", hostname)
		}
	}
	// Reject .svc and .internal suffixes (Kubernetes internal services).
	if strings.HasSuffix(lower, ".svc") || strings.HasSuffix(lower, ".svc.cluster.local") || strings.HasSuffix(lower, ".internal") {
		return fmt.Errorf("git repo URL points to internal service: %s", hostname)
	}

	// Reject private/link-local IP ranges.
	if ip := net.ParseIP(hostname); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
			return fmt.Errorf("git repo URL points to private/internal IP: %s", hostname)
		}
	}

	return nil
}

// validateGitRef ensures the git ref contains only safe characters.
func validateGitRef(ref string) error {
	if ref == "" || ref == "HEAD" {
		return nil
	}
	if !validGitRefRe.MatchString(ref) {
		return fmt.Errorf("git ref contains invalid characters: %s", ref)
	}
	return nil
}

// KanikoBuildJob creates a Kaniko Job spec for building a Function's container image.
// The Job is created in the platform-system namespace for security isolation.
func KanikoBuildJob(
	fn *etalbaasv1alpha1.Function,
	cfg config.OperatorConfig,
) (*batchv1.Job, error) {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name
	generation := fn.Generation
	shortHash := shortSha(fmt.Sprintf("%s-%s-%d", projectID, funcName, generation))
	imageTag := fmt.Sprintf("%d-%s", generation, shortHash)
	destination := fmt.Sprintf("%s/project-%s/%s:%s", cfg.RegistryEndpoint, projectID, funcName, imageTag)

	var backoffLimit int32 = 1
	activeDeadlineSeconds := int64(cfg.BuildTimeout.Seconds())

	labels := map[string]string{
		"etalbaas.io/project-id":    projectID,
		"etalbaas.io/function":      funcName,
		"etalbaas.io/build":         "true",
		"etalbaas.io/generation":    fmt.Sprintf("%d", generation),
		"app.kubernetes.io/part-of": "etalbaas",
	}

	// Build init container based on source type
	initContainers, err := buildInitContainers(fn)
	if err != nil {
		return nil, fmt.Errorf("build init containers: %w", err)
	}

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
							Name:            "kaniko",
							Image:           cfg.KanikoImage,
							ImagePullPolicy: corev1.PullIfNotPresent,
							Args: kanikoArgs(destination, cfg),
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
					Volumes: buildVolumes(fn, cfg),
				},
			},
		},
	}

	// Set RuntimeClassName only when configured (e.g., "gvisor" in production).
	if cfg.SandboxRuntimeClass != "" {
		rc := cfg.SandboxRuntimeClass
		job.Spec.Template.Spec.RuntimeClassName = &rc
	}

	return job, nil
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

// buildVolumes returns the volumes for the build Job.
// Always includes workspace (emptyDir) and dockerfile (ConfigMap).
// For inline/zip source types, adds a source-data volume backed by a ConfigMap.
func buildVolumes(fn *etalbaasv1alpha1.Function, cfg config.OperatorConfig) []corev1.Volume {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	volumes := []corev1.Volume{
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
	}

	// Inline source: files are delivered via ConfigMap.
	// Zip source: requires object storage download (not yet implemented).
	if fn.Spec.Source.Type == "inline" {
		volumes = append(volumes, corev1.Volume{
			Name: "source-data",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: SourceConfigMapName(projectID, funcName),
					},
				},
			},
		})
	}

	return volumes
}

// SourceConfigMapName returns the ConfigMap name for inline/zip source data.
func SourceConfigMapName(projectID, funcName string) string {
	return fmt.Sprintf("build-%s-%s-source", projectID, funcName)
}

// BuildSourceConfigMap creates a ConfigMap containing inline source files.
// For Node.js presets, automatically injects index.js (runtime wrapper) and
// package.json if not provided by the user.
func BuildSourceConfigMap(
	fn *etalbaasv1alpha1.Function,
	cfg config.OperatorConfig,
) *corev1.ConfigMap {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	data := make(map[string]string)
	if fn.Spec.Source.Inline != nil {
		for k, v := range fn.Spec.Source.Inline.Files {
			data[k] = v
		}
	}

	// Inject Node.js runtime wrapper if this is a Node preset and
	// the user hasn't provided their own index.js or package.json.
	preset := fn.Spec.Runtime.Preset
	if isNodePreset(preset) {
		if _, ok := data["index.js"]; !ok {
			data["index.js"] = NodeRuntimeWrapper
		}
		if _, ok := data["package.json"]; !ok {
			data["package.json"] = `{"type":"module"}`
		}
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SourceConfigMapName(projectID, funcName),
			Namespace: cfg.PlatformNamespace,
			Labels: map[string]string{
				"etalbaas.io/project-id":    projectID,
				"etalbaas.io/function":      funcName,
				"etalbaas.io/build":         "true",
				"app.kubernetes.io/part-of": "etalbaas",
			},
		},
		Data: data,
	}
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

func buildInitContainers(fn *etalbaasv1alpha1.Function) ([]corev1.Container, error) {
	source := fn.Spec.Source
	sc := initContainerSecurityContext()

	switch source.Type {
	case "git":
		if source.Git == nil {
			return []corev1.Container{dockerfileCopyInitContainer()}, nil
		}
		git := source.Git
		if err := validateGitRepo(git.Repo); err != nil {
			return nil, err
		}
		ref := "HEAD"
		if git.Ref != "" {
			if err := validateGitRef(git.Ref); err != nil {
				return nil, err
			}
			ref = git.Ref
		}
		// Use git clone with explicit argument separation to avoid shell injection.
		cloneArgs := []string{
			"git", "clone", "--depth", "1", "--branch", ref, git.Repo, "/tmp/repo",
		}
		copyCmd := "cp -r /tmp/repo/. /workspace/source/"
		if git.Path != "" && git.Path != "/" {
			if err := validateGitPath(git.Path); err != nil {
				return nil, err
			}
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
		}, nil
	case "inline":
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
		}, nil
	case "zip":
		// TODO: zip source requires downloading from object storage.
		// Not yet implemented; the build will fail with a clear error.
		return []corev1.Container{
			{
				Name:            "source-fetch",
				Image:           alpineImage,
				Command:         []string{"sh", "-c"},
				Args:            []string{"echo 'ERROR: zip source build not yet implemented' && exit 1"},
				SecurityContext: sc,
				VolumeMounts: []corev1.VolumeMount{
					{Name: "workspace", MountPath: "/workspace"},
				},
			},
			dockerfileCopyInitContainer(),
		}, nil
	default:
		return []corev1.Container{dockerfileCopyInitContainer()}, nil
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

func isNodePreset(preset string) bool {
	return strings.HasPrefix(preset, "node-")
}

// kanikoArgs builds the Kaniko executor arguments.
// --insecure and --skip-tls-verify are only added when RegistryInsecure is true.
func kanikoArgs(destination string, cfg config.OperatorConfig) []string {
	args := []string{
		"--dockerfile=/workspace/Dockerfile",
		"--context=/workspace/source",
		"--destination=" + destination,
		"--cache=true",
		"--cache-repo=" + cfg.RegistryEndpoint + "/cache",
	}
	if cfg.RegistryInsecure {
		args = append(args, "--insecure", "--skip-tls-verify")
	}
	return args
}

func shortSha(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", h[:4])
}
