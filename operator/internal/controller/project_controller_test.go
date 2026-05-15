package controller

import (
	"context"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/resources"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(s)
	_ = etalbaasv1alpha1.AddToScheme(s)
	_ = networkingv1.AddToScheme(s)
	return s
}

func testConfig() config.OperatorConfig {
	return config.DefaultConfig()
}

func newTestProject(name string, opts ...func(*etalbaasv1alpha1.Project)) *etalbaasv1alpha1.Project {
	replicas := int32(2)
	poolerInstances := int32(2)
	p := &etalbaasv1alpha1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "project-" + name,
			Labels: map[string]string{
				"etalbaas.io/user-id":    "user123",
				"etalbaas.io/project-id": name,
			},
		},
		Spec: etalbaasv1alpha1.ProjectSpec{
			DisplayName: "Test Project",
			Stack: etalbaasv1alpha1.ProjectStack{
				Postgres: &etalbaasv1alpha1.PostgresSpec{
					Enabled:    true,
					Version:    "16",
					Extensions: []string{"pgvector", "pgcrypto"},
					Storage:    "10Gi",
					Replicas:   &replicas,
					Pooler: &etalbaasv1alpha1.PoolerSpec{
						Enabled:   true,
						Instances: &poolerInstances,
					},
				},
				Redis: &etalbaasv1alpha1.RedisSpec{
					Enabled: true,
				},
				PostgREST: &etalbaasv1alpha1.PostgRESTSpec{
					Enabled:  true,
					AnonRole: "anon",
				},
			},
			Networking: etalbaasv1alpha1.NetworkingSpec{
				Subdomain:   name,
				DbSubdomain: name,
			},
			Plan: "free",
		},
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func TestProjectReconcile_CreatesNamespace(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// Verify namespace was created
	ns := &corev1.Namespace{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "project-abc123"}, ns); err != nil {
		t.Fatalf("Expected namespace to be created: %v", err)
	}

	// Verify PSS labels
	if ns.Labels["pod-security.kubernetes.io/enforce"] != "restricted" {
		t.Errorf("Expected PSS enforce=restricted, got %s", ns.Labels["pod-security.kubernetes.io/enforce"])
	}
}

func TestProjectReconcile_CreatesResourceQuota(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, _ = r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})

	// Verify ResourceQuota was created
	quota := &corev1.ResourceQuota{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "project-quota", Namespace: "project-abc123"}, quota); err != nil {
		t.Fatalf("Expected ResourceQuota to be created: %v", err)
	}

	cpuLimit := quota.Spec.Hard[corev1.ResourceLimitsCPU]
	if cpuLimit.String() != "2" {
		t.Errorf("Expected CPU limit 2, got %s", cpuLimit.String())
	}

	// Verify LimitRange was created
	lr := &corev1.LimitRange{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "project-limits", Namespace: "project-abc123"}, lr); err != nil {
		t.Fatalf("Expected LimitRange to be created: %v", err)
	}
}

func TestProjectReconcile_CreatesNetworkPolicies(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, _ = r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})

	expectedPolicies := []string{
		"default-deny-ingress",
		"allow-intra-namespace",
		"allow-platform",
		"egress-restrict",
	}

	for _, name := range expectedPolicies {
		np := &networkingv1.NetworkPolicy{}
		if err := fakeClient.Get(ctx, types.NamespacedName{Name: name, Namespace: "project-abc123"}, np); err != nil {
			t.Errorf("Expected NetworkPolicy %s to be created: %v", name, err)
		}
	}
}

func TestProjectReconcile_PostgresEnabled_CreatesCNPGAndCDC(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, _ = r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})

	// Verify CNPG Cluster was created (as unstructured)
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(resources.ClusterGVK())
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "db", Namespace: "project-abc123"}, cluster); err != nil {
		t.Fatalf("Expected CNPG Cluster to be created: %v", err)
	}

	// Verify CDC Deployment
	cdcDeploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "cdc", Namespace: "project-abc123"}, cdcDeploy); err != nil {
		t.Fatalf("Expected CDC Deployment to be created: %v", err)
	}

	// Verify CDC environment variables
	container := cdcDeploy.Spec.Template.Spec.Containers[0]
	var slotEnv string
	for _, env := range container.Env {
		if env.Name == "CDC_SLOT_NAME" {
			slotEnv = env.Value
		}
	}
	if slotEnv != "cdc_abc123" {
		t.Errorf("Expected CDC_SLOT_NAME=cdc_abc123, got %s", slotEnv)
	}
}

func TestProjectReconcile_PostgresDisabled_NoCNPGOrCDC(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123", func(p *etalbaasv1alpha1.Project) {
		p.Spec.Stack.Postgres = &etalbaasv1alpha1.PostgresSpec{Enabled: false}
	})

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, _ = r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})

	// Verify no CNPG Cluster
	cluster := &unstructured.Unstructured{}
	cluster.SetGroupVersionKind(resources.ClusterGVK())
	err := fakeClient.Get(ctx, types.NamespacedName{Name: "db", Namespace: "project-abc123"}, cluster)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected CNPG Cluster NOT to be created, but it was found or error: %v", err)
	}

	// Verify no CDC
	cdcDeploy := &appsv1.Deployment{}
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "cdc", Namespace: "project-abc123"}, cdcDeploy)
	if !errors.IsNotFound(err) {
		t.Errorf("Expected CDC Deployment NOT to be created, but it was found or error: %v", err)
	}
}

func TestProjectReconcile_Deletion_RemovesFinalizer(t *testing.T) {
	scheme := testScheme()
	now := metav1.Now()
	project := newTestProject("abc123", func(p *etalbaasv1alpha1.Project) {
		p.DeletionTimestamp = &now
		p.Finalizers = []string{projectFinalizer}
	})

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	// After removing the last finalizer on an object with DeletionTimestamp,
	// the fake client auto-deletes the object (simulating API server behavior).
	var updated etalbaasv1alpha1.Project
	err = fakeClient.Get(ctx, types.NamespacedName{Name: "abc123", Namespace: "project-abc123"}, &updated)
	if err == nil {
		// Object still exists - verify finalizer was removed
		for _, f := range updated.Finalizers {
			if f == projectFinalizer {
				t.Error("Expected finalizer to be removed, but it's still present")
			}
		}
	} else if !errors.IsNotFound(err) {
		t.Fatalf("Unexpected error getting project: %v", err)
	}
	// NotFound is acceptable - object was garbage collected after finalizer removal
}

func TestProjectReconcile_Idempotent(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	}

	// Run reconcile twice
	_, err := r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("First Reconcile failed: %v", err)
	}

	// Re-fetch the project since it was updated
	var updatedProject etalbaasv1alpha1.Project
	if err := fakeClient.Get(ctx, req.NamespacedName, &updatedProject); err != nil {
		t.Fatalf("Failed to get updated project: %v", err)
	}

	_, err = r.Reconcile(ctx, req)
	if err != nil {
		t.Fatalf("Second Reconcile failed: %v", err)
	}
}

func TestProjectReconcile_RedisEnabled(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, _ = r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})

	// Verify Redis Deployment
	deploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "redis", Namespace: "project-abc123"}, deploy); err != nil {
		t.Fatalf("Expected Redis Deployment to be created: %v", err)
	}

	// Verify Redis Service
	svc := &corev1.Service{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "redis", Namespace: "project-abc123"}, svc); err != nil {
		t.Fatalf("Expected Redis Service to be created: %v", err)
	}
}

func TestProjectReconcile_PostgRESTEnabled(t *testing.T) {
	scheme := testScheme()
	project := newTestProject("abc123")

	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(project).
		WithStatusSubresource(project).
		Build()

	r := &ProjectReconciler{
		Client: fakeClient,
		Scheme: scheme,
		Config: testConfig(),
	}

	ctx := context.Background()
	_, _ = r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "abc123",
			Namespace: "project-abc123",
		},
	})

	// Verify PostgREST resources
	deploy := &appsv1.Deployment{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "postgrest", Namespace: "project-abc123"}, deploy); err != nil {
		t.Fatalf("Expected PostgREST Deployment to be created: %v", err)
	}

	svc := &corev1.Service{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "postgrest", Namespace: "project-abc123"}, svc); err != nil {
		t.Fatalf("Expected PostgREST Service to be created: %v", err)
	}

	cm := &corev1.ConfigMap{}
	if err := fakeClient.Get(ctx, types.NamespacedName{Name: "postgrest-config", Namespace: "project-abc123"}, cm); err != nil {
		t.Fatalf("Expected PostgREST ConfigMap to be created: %v", err)
	}

	if cm.Data["PGRST_DB_ANON_ROLE"] != "anon" {
		t.Errorf("Expected PGRST_DB_ANON_ROLE=anon, got %s", cm.Data["PGRST_DB_ANON_ROLE"])
	}
}

func TestDesiredNamespace_Labels(t *testing.T) {
	project := newTestProject("abc123")
	ns := resources.DesiredNamespace(project)

	if ns.Name != "project-abc123" {
		t.Errorf("Expected namespace name project-abc123, got %s", ns.Name)
	}

	if ns.Labels[resources.LabelProjectID] != "abc123" {
		t.Errorf("Expected project-id label abc123, got %s", ns.Labels[resources.LabelProjectID])
	}

	if ns.Labels["pod-security.kubernetes.io/enforce"] != "restricted" {
		t.Errorf("Expected PSS enforce=restricted, got %s", ns.Labels["pod-security.kubernetes.io/enforce"])
	}
}

func TestDesiredResourceQuota_FreePlan(t *testing.T) {
	project := newTestProject("abc123")
	cfg := testConfig()
	quota := resources.DesiredResourceQuota(project, cfg)

	cpuLimit := quota.Spec.Hard[corev1.ResourceLimitsCPU]
	if cpuLimit.String() != "2" {
		t.Errorf("Expected CPU limit 2, got %s", cpuLimit.String())
	}

	memLimit := quota.Spec.Hard[corev1.ResourceLimitsMemory]
	if memLimit.String() != "4Gi" {
		t.Errorf("Expected Memory limit 4Gi, got %s", memLimit.String())
	}

	pods := quota.Spec.Hard[corev1.ResourcePods]
	if pods.Value() != 20 {
		t.Errorf("Expected Pods limit 20, got %d", pods.Value())
	}
}

func TestDesiredNetworkPolicies(t *testing.T) {
	project := newTestProject("abc123")

	deny := resources.DesiredDefaultDenyNetworkPolicy(project)
	if deny.Namespace != "project-abc123" {
		t.Errorf("Expected namespace project-abc123, got %s", deny.Namespace)
	}
	if len(deny.Spec.Ingress) != 0 {
		t.Error("Expected default-deny to have no ingress rules")
	}

	intra := resources.DesiredAllowIntraNamespaceNetworkPolicy(project)
	if len(intra.Spec.Ingress) != 1 {
		t.Errorf("Expected 1 ingress rule, got %d", len(intra.Spec.Ingress))
	}

	platform := resources.DesiredAllowPlatformNetworkPolicy(project, "platform-system")
	if len(platform.Spec.Ingress) != 1 {
		t.Errorf("Expected 1 ingress rule, got %d", len(platform.Spec.Ingress))
	}

	egress := resources.DesiredEgressNetworkPolicy(project, "platform-system")
	if len(egress.Spec.Egress) != 4 {
		t.Errorf("Expected 4 egress rules (DNS, HTTPS, intra-ns, NATS), got %d", len(egress.Spec.Egress))
	}
}

func TestDesiredCloudNativePGCluster(t *testing.T) {
	project := newTestProject("abc123")
	cluster := resources.DesiredCloudNativePGCluster(project)

	if cluster == nil {
		t.Fatal("Expected CNPG Cluster to be created")
	}

	instances, _, _ := unstructured.NestedInt64(cluster.Object, "spec", "instances")
	if instances != 2 {
		t.Errorf("Expected 2 instances, got %d", instances)
	}

	walLevel, _, _ := unstructured.NestedString(cluster.Object, "spec", "postgresql", "parameters", "wal_level")
	if walLevel != "logical" {
		t.Errorf("Expected wal_level=logical, got %s", walLevel)
	}
}

func TestDesiredCloudNativePGCluster_Disabled(t *testing.T) {
	project := newTestProject("abc123", func(p *etalbaasv1alpha1.Project) {
		p.Spec.Stack.Postgres = &etalbaasv1alpha1.PostgresSpec{Enabled: false}
	})

	cluster := resources.DesiredCloudNativePGCluster(project)
	if cluster != nil {
		t.Error("Expected nil Cluster when postgres is disabled")
	}
}

func TestDesiredHTTPRoute(t *testing.T) {
	project := newTestProject("abc123")
	cfg := testConfig()
	route := resources.DesiredHTTPRoute(project, cfg)

	if route == nil {
		t.Fatal("Expected HTTPRoute to be created")
	}

	hostnames, _, _ := unstructured.NestedStringSlice(route.Object, "spec", "hostnames")
	if len(hostnames) != 1 || hostnames[0] != "abc123.api.yourdomain.com" {
		t.Errorf("Expected hostname abc123.api.yourdomain.com, got %v", hostnames)
	}
}

func TestDesiredTLSRoute(t *testing.T) {
	project := newTestProject("abc123")
	cfg := testConfig()
	route := resources.DesiredTLSRoute(project, cfg)

	if route == nil {
		t.Fatal("Expected TLSRoute to be created")
	}

	hostnames, _, _ := unstructured.NestedStringSlice(route.Object, "spec", "hostnames")
	if len(hostnames) != 1 || hostnames[0] != "abc123.db.yourdomain.com" {
		t.Errorf("Expected hostname abc123.db.yourdomain.com, got %v", hostnames)
	}
}
