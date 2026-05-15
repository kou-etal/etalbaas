package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/build"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/resources"
)

func functionTestScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = etalbaasv1alpha1.AddToScheme(s)
	_ = autoscalingv2.AddToScheme(s)
	return s
}

func newTestFunction(name, projectID string, kind string, opts ...func(*etalbaasv1alpha1.Function)) *etalbaasv1alpha1.Function {
	fn := &etalbaasv1alpha1.Function{
		ObjectMeta: metav1.ObjectMeta{
			Name:       name,
			Namespace:  "project-" + projectID,
			Generation: 1,
			Labels: map[string]string{
				"etalbaas.io/project-id": projectID,
			},
		},
		Spec: etalbaasv1alpha1.FunctionSpec{
			DisplayName: "Test Function",
			ProjectRef:  etalbaasv1alpha1.ProjectReference{Name: projectID},
			Kind:        kind,
			Source: etalbaasv1alpha1.FunctionSource{
				Type: "git",
				Git: &etalbaasv1alpha1.GitSource{
					Repo: "https://github.com/user/my-func",
					Ref:  "main",
					Path: "/",
				},
			},
			Runtime: etalbaasv1alpha1.RuntimeSpec{
				Preset: "python-3.11",
			},
			Triggers: []etalbaasv1alpha1.TriggerSpec{
				{
					Type: "Http",
					Http: &etalbaasv1alpha1.HttpTrigger{
						Path:           "/invoke",
						Authentication: "apikey",
					},
				},
			},
		},
	}
	for _, opt := range opts {
		opt(fn)
	}
	return fn
}

func newTestParentProject(projectID string) *etalbaasv1alpha1.Project {
	return &etalbaasv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      projectID,
			Namespace: "project-" + projectID,
			Labels: map[string]string{
				"etalbaas.io/user-id":    "user123",
				"etalbaas.io/project-id": projectID,
			},
		},
		Spec: etalbaasv1alpha1.ProjectSpec{
			Plan: "free",
		},
		Status: etalbaasv1alpha1.ProjectStatus{
			Phase: etalbaasv1alpha1.ProjectPhaseReady,
		},
	}
}

// --- Dockerfile tests ---

func TestGenerateDockerfile_Python311(t *testing.T) {
	df, err := build.GenerateDockerfile("python-3.11", []string{"torch==2.1.0"}, "")
	if err != nil {
		t.Fatalf("GenerateDockerfile failed: %v", err)
	}
	if df == "" {
		t.Fatal("Expected non-empty Dockerfile")
	}
	assertContains(t, df, "FROM python:3.11-slim")
	assertContains(t, df, "pip install --no-cache-dir torch==2.1.0")
	assertContains(t, df, "USER appuser")
}

func TestGenerateDockerfile_Python311ML(t *testing.T) {
	df, err := build.GenerateDockerfile("python-3.11-ml", nil, "")
	if err != nil {
		t.Fatalf("GenerateDockerfile failed: %v", err)
	}
	assertContains(t, df, "numpy pandas scikit-learn")
}

func TestGenerateDockerfile_Node20(t *testing.T) {
	df, err := build.GenerateDockerfile("node-20", nil, "")
	if err != nil {
		t.Fatalf("GenerateDockerfile failed: %v", err)
	}
	assertContains(t, df, "FROM node:20-slim")
	assertContains(t, df, "npm ci")
}

func TestGenerateDockerfile_Go122(t *testing.T) {
	df, err := build.GenerateDockerfile("go-1.22", nil, "")
	if err != nil {
		t.Fatalf("GenerateDockerfile failed: %v", err)
	}
	assertContains(t, df, "FROM golang:1.22")
	assertContains(t, df, "CGO_ENABLED=0")
	assertContains(t, df, "distroless")
}

func TestGenerateDockerfile_Custom(t *testing.T) {
	customDF := "FROM alpine:latest\nCOPY . /app\n"
	df, err := build.GenerateDockerfile("custom", nil, customDF)
	if err != nil {
		t.Fatalf("GenerateDockerfile failed: %v", err)
	}
	if df != customDF {
		t.Errorf("Expected custom Dockerfile to be passed through")
	}
}

func TestGenerateDockerfile_CustomEmpty(t *testing.T) {
	_, err := build.GenerateDockerfile("custom", nil, "")
	if err == nil {
		t.Error("Expected error for custom preset without Dockerfile")
	}
}

func TestGenerateDockerfile_Unsupported(t *testing.T) {
	_, err := build.GenerateDockerfile("ruby-3.2", nil, "")
	if err == nil {
		t.Error("Expected error for unsupported preset")
	}
}

// --- Kaniko Job tests ---

func TestKanikoBuildJob(t *testing.T) {
	fn := newTestFunction("image-gen", "abc123", etalbaasv1alpha1.FunctionKindHeavyDeployment)
	cfg := config.DefaultConfig()
	job := build.KanikoBuildJob(fn, cfg)

	if job.Namespace != cfg.PlatformNamespace {
		t.Errorf("Expected namespace %s, got %s", cfg.PlatformNamespace, job.Namespace)
	}

	// Verify gVisor RuntimeClass
	if job.Spec.Template.Spec.RuntimeClassName == nil || *job.Spec.Template.Spec.RuntimeClassName != "gvisor" {
		t.Error("Expected gVisor RuntimeClass on build job")
	}

	// Verify backoff limit
	if job.Spec.BackoffLimit == nil || *job.Spec.BackoffLimit != 1 {
		t.Errorf("Expected backoffLimit=1, got %v", job.Spec.BackoffLimit)
	}

	// Verify active deadline
	if job.Spec.ActiveDeadlineSeconds == nil || *job.Spec.ActiveDeadlineSeconds != 900 {
		t.Errorf("Expected activeDeadlineSeconds=900, got %v", job.Spec.ActiveDeadlineSeconds)
	}

	// Verify kaniko container has destination with registry
	kanikoContainer := job.Spec.Template.Spec.Containers[0]
	foundDest := false
	for _, arg := range kanikoContainer.Args {
		if len(arg) > 14 && arg[:14] == "--destination=" {
			foundDest = true
			assertContains(t, arg, "project-abc123")
			assertContains(t, arg, "image-gen")
		}
	}
	if !foundDest {
		t.Error("Expected --destination flag in kaniko args")
	}
}

// --- Function Deployment tests ---

func TestDesiredFunctionDeployment_LightDeployment(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment)
	deploy := resources.DesiredFunctionDeployment(fn, "zot.example.com/project-abc123/my-func:1-abc")

	if deploy == nil {
		t.Fatal("Expected Deployment for light-deployment")
	}

	if *deploy.Spec.Replicas != 1 {
		t.Errorf("Expected 1 replica for light-deployment, got %d", *deploy.Spec.Replicas)
	}

	// Verify gVisor
	if deploy.Spec.Template.Spec.RuntimeClassName == nil || *deploy.Spec.Template.Spec.RuntimeClassName != "gvisor" {
		t.Error("Expected gVisor RuntimeClass")
	}

	// Verify security context
	sc := deploy.Spec.Template.Spec.Containers[0].SecurityContext
	if sc == nil {
		t.Fatal("Expected SecurityContext")
	}
	if sc.ReadOnlyRootFilesystem == nil || !*sc.ReadOnlyRootFilesystem {
		t.Error("Expected readOnlyRootFilesystem=true")
	}
	if sc.RunAsNonRoot == nil || !*sc.RunAsNonRoot {
		t.Error("Expected runAsNonRoot=true")
	}
	if len(sc.Capabilities.Drop) != 1 || sc.Capabilities.Drop[0] != "ALL" {
		t.Error("Expected capabilities drop ALL")
	}
}

func TestDesiredFunctionDeployment_HeavyDeployment(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindHeavyDeployment)
	deploy := resources.DesiredFunctionDeployment(fn, "zot.example.com/project-abc123/my-func:1-abc")

	if deploy == nil {
		t.Fatal("Expected Deployment for heavy-deployment")
	}

	if *deploy.Spec.Replicas != 0 {
		t.Errorf("Expected 0 replicas for heavy-deployment, got %d", *deploy.Spec.Replicas)
	}
}

func TestDesiredFunctionDeployment_HeavyJob_ReturnsNil(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindHeavyJob)
	deploy := resources.DesiredFunctionDeployment(fn, "zot.example.com/project-abc123/my-func:1-abc")

	if deploy != nil {
		t.Error("Expected nil Deployment for heavy-job")
	}
}

func TestDesiredFunctionDeployment_GPUSelfManaged(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindHeavyDeployment, func(f *etalbaasv1alpha1.Function) {
		f.Spec.GPU = &etalbaasv1alpha1.GPUSpec{
			Required: true,
			Type:     "A10G",
			Provider: "self-managed",
		}
	})
	deploy := resources.DesiredFunctionDeployment(fn, "zot.example.com/project-abc123/my-func:1-abc")

	if deploy == nil {
		t.Fatal("Expected Deployment for GPU self-managed")
	}

	// Verify nvidia RuntimeClass (not gVisor)
	if *deploy.Spec.Template.Spec.RuntimeClassName != "nvidia" {
		t.Errorf("Expected nvidia RuntimeClass, got %s", *deploy.Spec.Template.Spec.RuntimeClassName)
	}

	// Verify GPU nodeSelector
	if deploy.Spec.Template.Spec.NodeSelector["gpu"] != "true" {
		t.Error("Expected gpu=true nodeSelector")
	}

	// Verify GPU resource limit
	gpuLimit := deploy.Spec.Template.Spec.Containers[0].Resources.Limits["nvidia.com/gpu"]
	if gpuLimit.String() != "1" {
		t.Errorf("Expected nvidia.com/gpu=1, got %s", gpuLimit.String())
	}

	// Verify tolerations
	if len(deploy.Spec.Template.Spec.Tolerations) == 0 {
		t.Error("Expected GPU tolerations")
	}
}

// --- Env var tests ---

func TestDesiredFunctionDeployment_EnvVars(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment, func(f *etalbaasv1alpha1.Function) {
		f.Spec.Env = []etalbaasv1alpha1.EnvVar{
			{Name: "PLAIN_VAR", Value: "hello"},
			{Name: "SECRET_VAR", SecretName: "my-secret"},
		}
	})
	deploy := resources.DesiredFunctionDeployment(fn, "zot.example.com/project-abc123/my-func:1-abc")

	envVars := deploy.Spec.Template.Spec.Containers[0].Env

	if len(envVars) != 2 {
		t.Fatalf("Expected 2 env vars, got %d", len(envVars))
	}

	// Plain value
	if envVars[0].Name != "PLAIN_VAR" || envVars[0].Value != "hello" {
		t.Errorf("Expected PLAIN_VAR=hello, got %s=%s", envVars[0].Name, envVars[0].Value)
	}

	// Secret ref
	if envVars[1].Name != "SECRET_VAR" || envVars[1].ValueFrom == nil {
		t.Error("Expected SECRET_VAR with secretKeyRef")
	}
	if envVars[1].ValueFrom.SecretKeyRef.Name != "my-secret" {
		t.Errorf("Expected secret name my-secret, got %s", envVars[1].ValueFrom.SecretKeyRef.Name)
	}
}

// --- Autoscaling tests ---

func TestDesiredHPA_LightDeployment(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment)
	hpa := resources.DesiredHPA(fn)

	if hpa == nil {
		t.Fatal("Expected HPA for light-deployment")
	}

	if *hpa.Spec.MinReplicas != 1 {
		t.Errorf("Expected minReplicas=1, got %d", *hpa.Spec.MinReplicas)
	}

	if hpa.Spec.MaxReplicas != 10 {
		t.Errorf("Expected maxReplicas=10, got %d", hpa.Spec.MaxReplicas)
	}

	if len(hpa.Spec.Metrics) != 1 || hpa.Spec.Metrics[0].Resource.Name != "cpu" {
		t.Error("Expected CPU metric")
	}

	if *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization != 70 {
		t.Errorf("Expected CPU target 70%%, got %d%%", *hpa.Spec.Metrics[0].Resource.Target.AverageUtilization)
	}
}

func TestDesiredHPA_HeavyDeployment_ReturnsNil(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindHeavyDeployment)
	hpa := resources.DesiredHPA(fn)

	if hpa != nil {
		t.Error("Expected nil HPA for heavy-deployment (should use KEDA)")
	}
}

func TestDesiredKEDAScaledObject_HeavyDeployment(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindHeavyDeployment)
	so := resources.DesiredKEDAScaledObject(fn)

	if so == nil {
		t.Fatal("Expected KEDA ScaledObject for heavy-deployment")
	}

	if so.GetKind() != "ScaledObject" {
		t.Errorf("Expected kind ScaledObject, got %s", so.GetKind())
	}

	cooldown, _, _ := unstructured.NestedInt64(so.Object, "spec", "cooldownPeriod")
	if cooldown != 300 {
		t.Errorf("Expected cooldownPeriod=300, got %d", cooldown)
	}
}

func TestDesiredKEDAScaledObject_LightDeployment_ReturnsNil(t *testing.T) {
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment)
	so := resources.DesiredKEDAScaledObject(fn)

	if so != nil {
		t.Error("Expected nil ScaledObject for light-deployment")
	}
}

// --- Controller integration tests ---

func TestFunctionReconcile_CreatesBuildJob(t *testing.T) {
	scheme := functionTestScheme()
	project := newTestParentProject("abc123")
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment)

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, fn).
		WithStatusSubresource(fn).
		Build()

	r := &FunctionReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	result, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "my-func",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Should requeue for build
	if result.RequeueAfter == 0 {
		t.Error("Expected requeue for build")
	}

	// Verify build job was created
	jobList := &batchv1.JobList{}
	if err := fakeClient.List(ctx, jobList); err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(jobList.Items) != 1 {
		t.Fatalf("Expected 1 build job, got %d", len(jobList.Items))
	}

	// Verify job is in platform-system namespace
	if jobList.Items[0].Namespace != "platform-system" {
		t.Errorf("Expected job in platform-system, got %s", jobList.Items[0].Namespace)
	}

	// Verify function phase is Building
	var updated etalbaasv1alpha1.Function
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "my-func", Namespace: "project-abc123"}, &updated); err != nil {
		t.Fatalf("Failed to get function: %v", err)
	}
	if updated.Status.Phase != etalbaasv1alpha1.FunctionPhaseBuilding {
		t.Errorf("Expected phase Building, got %s", updated.Status.Phase)
	}
}

func TestFunctionReconcile_BuildSucceeded_CreatesDeployment(t *testing.T) {
	scheme := functionTestScheme()
	project := newTestParentProject("abc123")
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment, func(f *etalbaasv1alpha1.Function) {
		f.Status.Phase = etalbaasv1alpha1.FunctionPhaseReady
		f.Status.ObservedGeneration = 1
		f.Status.Build = &etalbaasv1alpha1.BuildStatus{
			Status:   etalbaasv1alpha1.BuildStatusSucceeded,
			ImageRef: "zot.platform-system.svc:5000/project-abc123/my-func:1-abc",
		}
	})

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, fn).
		WithStatusSubresource(fn).
		Build()

	r := &FunctionReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "my-func",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created
	deploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "func-my-func", Namespace: "project-abc123"}, deploy); err != nil {
		t.Fatalf("Expected Deployment to be created: %v", err)
	}

	// Verify image
	if deploy.Spec.Template.Spec.Containers[0].Image != "zot.platform-system.svc:5000/project-abc123/my-func:1-abc" {
		t.Errorf("Expected correct image, got %s", deploy.Spec.Template.Spec.Containers[0].Image)
	}

	// Verify HPA was created (light-deployment)
	hpa := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "func-my-func", Namespace: "project-abc123"}, hpa); err != nil {
		t.Fatalf("Expected HPA to be created: %v", err)
	}
}

func TestFunctionReconcile_HeavyDeployment_CreatesScaledObject(t *testing.T) {
	scheme := functionTestScheme()
	project := newTestParentProject("abc123")
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindHeavyDeployment, func(f *etalbaasv1alpha1.Function) {
		f.Status.Phase = etalbaasv1alpha1.FunctionPhaseReady
		f.Status.ObservedGeneration = 1
		f.Status.Build = &etalbaasv1alpha1.BuildStatus{
			Status:   etalbaasv1alpha1.BuildStatusSucceeded,
			ImageRef: "zot.platform-system.svc:5000/project-abc123/my-func:1-abc",
		}
	})

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, fn).
		WithStatusSubresource(fn).
		Build()

	r := &FunctionReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "my-func",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment was created with 0 replicas
	deploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "func-my-func", Namespace: "project-abc123"}, deploy); err != nil {
		t.Fatalf("Expected Deployment: %v", err)
	}
	if *deploy.Spec.Replicas != 0 {
		t.Errorf("Expected 0 replicas for heavy-deployment, got %d", *deploy.Spec.Replicas)
	}

	// Verify KEDA ScaledObject was created
	so := &unstructured.Unstructured{}
	so.SetGroupVersionKind(schema.GroupVersionKind{Group: "keda.sh", Version: "v1alpha1", Kind: "ScaledObject"})
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "func-my-func", Namespace: "project-abc123"}, so); err != nil {
		t.Fatalf("Expected KEDA ScaledObject: %v", err)
	}
}

func TestFunctionReconcile_Deletion_CleansUpBuildJobs(t *testing.T) {
	scheme := functionTestScheme()
	project := newTestParentProject("abc123")
	now := metav1.Now()
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment, func(f *etalbaasv1alpha1.Function) {
		f.DeletionTimestamp = &now
		f.Finalizers = []string{functionFinalizer}
	})

	// Create a build job
	buildJob := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "build-abc123-my-func-abc",
			Namespace: "platform-system",
			Labels: map[string]string{
				"etalbaas.io/project-id": "abc123",
				"etalbaas.io/function":   "my-func",
				"etalbaas.io/build":      "true",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{Name: "kaniko", Image: "kaniko:latest"},
					},
				},
			},
		},
	}

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, fn, buildJob).
		WithStatusSubresource(fn).
		Build()

	r := &FunctionReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "my-func",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify build job was deleted
	jobList := &batchv1.JobList{}
	if err := fakeClient.List(ctx, jobList); err != nil {
		t.Fatalf("Failed to list jobs: %v", err)
	}
	if len(jobList.Items) != 0 {
		t.Errorf("Expected build jobs to be cleaned up, got %d", len(jobList.Items))
	}
}

func TestFunctionReconcile_GPUSelfManaged(t *testing.T) {
	scheme := functionTestScheme()
	project := newTestParentProject("abc123")
	fn := newTestFunction("gpu-func", "abc123", etalbaasv1alpha1.FunctionKindHeavyDeployment, func(f *etalbaasv1alpha1.Function) {
		f.Spec.GPU = &etalbaasv1alpha1.GPUSpec{
			Required: true,
			Type:     "A10G",
			Provider: "self-managed",
		}
		f.Status.Phase = etalbaasv1alpha1.FunctionPhaseReady
		f.Status.ObservedGeneration = 1
		f.Status.Build = &etalbaasv1alpha1.BuildStatus{
			Status:   etalbaasv1alpha1.BuildStatusSucceeded,
			ImageRef: "zot.platform-system.svc:5000/project-abc123/gpu-func:1-abc",
		}
	})

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, fn).
		WithStatusSubresource(fn).
		Build()

	r := &FunctionReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "gpu-func",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify Deployment
	deploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "func-gpu-func", Namespace: "project-abc123"}, deploy); err != nil {
		t.Fatalf("Expected Deployment: %v", err)
	}

	// Verify nvidia RuntimeClass
	if *deploy.Spec.Template.Spec.RuntimeClassName != "nvidia" {
		t.Errorf("Expected nvidia RuntimeClass, got %s", *deploy.Spec.Template.Spec.RuntimeClassName)
	}

	// Verify GPU resource
	gpuLimit := deploy.Spec.Template.Spec.Containers[0].Resources.Limits[corev1.ResourceName("nvidia.com/gpu")]
	if gpuLimit.Cmp(resource.MustParse("1")) != 0 {
		t.Errorf("Expected nvidia.com/gpu=1, got %s", gpuLimit.String())
	}

	// Verify function status
	var updated etalbaasv1alpha1.Function
	_ = fakeClient.Get(ctx, types.NamespacedName{Name: "gpu-func", Namespace: "project-abc123"}, &updated)
	if updated.Status.Execution != nil && updated.Status.Execution.RuntimeClass != "nvidia" {
		t.Errorf("Expected status.execution.runtimeClass=nvidia, got %s", updated.Status.Execution.RuntimeClass)
	}
}

func TestFunctionReconcile_HTTPRoute(t *testing.T) {
	scheme := functionTestScheme()
	project := newTestParentProject("abc123")
	fn := newTestFunction("my-func", "abc123", etalbaasv1alpha1.FunctionKindLightDeployment, func(f *etalbaasv1alpha1.Function) {
		f.Status.Phase = etalbaasv1alpha1.FunctionPhaseReady
		f.Status.ObservedGeneration = 1
		f.Status.Build = &etalbaasv1alpha1.BuildStatus{
			Status:   etalbaasv1alpha1.BuildStatusSucceeded,
			ImageRef: "zot.platform-system.svc:5000/project-abc123/my-func:1-abc",
		}
	})

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project, fn).
		WithStatusSubresource(fn).
		Build()

	r := &FunctionReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "my-func",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify HTTPRoute was created
	route := &unstructured.Unstructured{}
	route.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   "gateway.networking.k8s.io",
		Version: "v1",
		Kind:    "HTTPRoute",
	})
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "func-my-func", Namespace: "project-abc123"}, route); err != nil {
		t.Fatalf("Expected HTTPRoute: %v", err)
	}

	// Verify trigger status
	var updated etalbaasv1alpha1.Function
	_ = fakeClient.Get(ctx, types.NamespacedName{Name: "my-func", Namespace: "project-abc123"}, &updated)
	if len(updated.Status.Triggers) != 1 {
		t.Fatalf("Expected 1 trigger status, got %d", len(updated.Status.Triggers))
	}
	if updated.Status.Triggers[0].Type != "Http" || updated.Status.Triggers[0].Status != "Active" {
		t.Errorf("Expected active Http trigger, got %+v", updated.Status.Triggers[0])
	}
}

func TestFunctionReconcile_NotFound(t *testing.T) {
	scheme := functionTestScheme()

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	r := &FunctionReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	result, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Expected no error for not found, got %v", err)
	}
	if result.Requeue {
		t.Error("Expected no requeue for not found")
	}
}

// --- Helper ---

func assertContains(t *testing.T, s, substr string) {
	t.Helper()
	if !containsStr(s, substr) {
		t.Errorf("Expected string to contain %q, but it didn't.\nString: %s", substr, s)
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Ensure unused imports are used
var (
	_ = errors.IsNotFound
	_ = resource.MustParse
)
