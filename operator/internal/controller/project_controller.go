package controller

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/resources"
)

const projectFinalizer = "etalbaas.io/project-finalizer"

// ProjectReconciler reconciles a Project object.
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Config config.OperatorConfig
}

// +kubebuilder:rbac:groups=etalbaas.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=etalbaas.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=etalbaas.io,resources=projects/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=resourcequotas;limitranges;services;configmaps;secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=networkpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes;tlsroutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=postgresql.cnpg.io,resources=clusters;poolers,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles the reconciliation loop for Project resources.
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch Project CR
	var project etalbaasv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Handle finalizer and deletion
	if !project.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &project)
	}

	if !controllerutil.ContainsFinalizer(&project, projectFinalizer) {
		controllerutil.AddFinalizer(&project, projectFinalizer)
		if err := r.Update(ctx, &project); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. Set phase to Provisioning if Pending
	if project.Status.Phase == "" || project.Status.Phase == etalbaasv1alpha1.ProjectPhasePending {
		project.Status.Phase = etalbaasv1alpha1.ProjectPhaseProvisioning
		if err := r.Status().Update(ctx, &project); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 4. Reconcile all sub-resources
	if err := r.reconcileNamespace(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile namespace")
		return r.setFailed(ctx, &project, "NamespaceFailed", err)
	}

	if err := r.reconcileResourceQuota(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile resource quota")
		return r.setFailed(ctx, &project, "ResourceQuotaFailed", err)
	}

	if err := r.reconcileNetworkPolicies(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile network policies")
		return r.setFailed(ctx, &project, "NetworkPolicyFailed", err)
	}

	if err := r.reconcilePostgres(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile postgres")
		return r.setFailed(ctx, &project, "PostgresFailed", err)
	}

	if err := r.reconcileCDC(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile CDC")
		return r.setFailed(ctx, &project, "CDCFailed", err)
	}

	if err := r.reconcileRedis(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile redis")
		return r.setFailed(ctx, &project, "RedisFailed", err)
	}

	if err := r.reconcilePostgREST(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile PostgREST")
		return r.setFailed(ctx, &project, "PostgRESTFailed", err)
	}

	if err := r.reconcileHTTPRoute(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile HTTPRoute")
		return r.setFailed(ctx, &project, "HTTPRouteFailed", err)
	}

	if err := r.reconcileTLSRoute(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile TLSRoute")
		return r.setFailed(ctx, &project, "TLSRouteFailed", err)
	}

	// 13. Update status
	return r.updateStatus(ctx, &project)
}

// handleDeletion processes the finalizer cleanup.
func (r *ProjectReconciler) handleDeletion(ctx context.Context, project *etalbaasv1alpha1.Project) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(project, projectFinalizer) {
		// Delete the namespace (cascade deletes all child resources)
		ns := &corev1.Namespace{}
		nsName := "project-" + project.Name
		if err := r.Get(ctx, types.NamespacedName{Name: nsName}, ns); err == nil {
			logger.Info("deleting project namespace", "namespace", nsName)
			if err := r.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("delete namespace %s: %w", nsName, err)
			}
		}

		controllerutil.RemoveFinalizer(project, projectFinalizer)
		if err := r.Update(ctx, project); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (r *ProjectReconciler) reconcileNamespace(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	desired := resources.DesiredNamespace(project)

	existing := &corev1.Namespace{}
	err := r.Get(ctx, types.NamespacedName{Name: desired.Name}, existing)
	if apierrors.IsNotFound(err) {
		return r.Create(ctx, desired)
	}
	if err != nil {
		return err
	}

	// Update labels
	existing.Labels = desired.Labels
	return r.Update(ctx, existing)
}

func (r *ProjectReconciler) reconcileResourceQuota(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	quota := resources.DesiredResourceQuota(project, r.Config)
	if err := r.reconcileNamespacedResource(ctx, project, quota); err != nil {
		return err
	}

	limitRange := resources.DesiredLimitRange(project)
	return r.reconcileNamespacedResource(ctx, project, limitRange)
}

func (r *ProjectReconciler) reconcileNetworkPolicies(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	policies := []*networkingv1.NetworkPolicy{
		resources.DesiredDefaultDenyNetworkPolicy(project),
		resources.DesiredAllowIntraNamespaceNetworkPolicy(project),
		resources.DesiredAllowPlatformNetworkPolicy(project, r.Config.PlatformNamespace),
		resources.DesiredEgressNetworkPolicy(project, r.Config.PlatformNamespace),
	}

	for _, policy := range policies {
		if err := r.reconcileNamespacedResource(ctx, project, policy); err != nil {
			return err
		}
	}
	return nil
}

func (r *ProjectReconciler) reconcilePostgres(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name

	cluster := resources.DesiredCloudNativePGCluster(project)
	if cluster != nil {
		if err := r.reconcileUnstructured(ctx, cluster); err != nil {
			return fmt.Errorf("reconcile CNPG Cluster: %w", err)
		}
	} else {
		// Postgres disabled: clean up existing CNPG Cluster and Pooler
		if err := r.deleteUnstructuredIfExists(ctx, resources.PoolerGVK(), namespace, "db-pooler"); err != nil {
			return fmt.Errorf("cleanup CNPG Pooler: %w", err)
		}
		if err := r.deleteUnstructuredIfExists(ctx, resources.ClusterGVK(), namespace, "db"); err != nil {
			return fmt.Errorf("cleanup CNPG Cluster: %w", err)
		}
		return nil
	}

	pooler := resources.DesiredCloudNativePGPooler(project)
	if pooler != nil {
		if err := r.reconcileUnstructured(ctx, pooler); err != nil {
			return fmt.Errorf("reconcile CNPG Pooler: %w", err)
		}
	} else {
		// Pooler disabled: clean up existing Pooler
		if err := r.deleteUnstructuredIfExists(ctx, resources.PoolerGVK(), namespace, "db-pooler"); err != nil {
			return fmt.Errorf("cleanup CNPG Pooler: %w", err)
		}
	}

	return nil
}

func (r *ProjectReconciler) reconcileCDC(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	deploy := resources.DesiredCDCDeployment(project, r.Config)
	if deploy == nil {
		// Postgres disabled: clean up existing CDC deployment
		namespace := "project-" + project.Name
		r.deleteIfExists(ctx, &appsv1.Deployment{}, namespace, "cdc")
		return nil
	}
	return r.reconcileNamespacedResource(ctx, project, deploy)
}

func (r *ProjectReconciler) reconcileRedis(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name
	deploy := resources.DesiredRedisDeployment(project, r.Config)
	if deploy != nil {
		if err := r.reconcileNamespacedResource(ctx, project, deploy); err != nil {
			return err
		}
	} else {
		// Redis disabled: clean up existing resources
		r.deleteIfExists(ctx, &appsv1.Deployment{}, namespace, "redis")
	}

	svc := resources.DesiredRedisService(project)
	if svc != nil {
		return r.reconcileNamespacedResource(ctx, project, svc)
	}
	r.deleteIfExists(ctx, &corev1.Service{}, namespace, "redis")
	return nil
}

func (r *ProjectReconciler) reconcilePostgREST(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name
	cm := resources.DesiredPostgRESTConfigMap(project)
	if cm != nil {
		if err := r.reconcileNamespacedResource(ctx, project, cm); err != nil {
			return err
		}
	} else {
		r.deleteIfExists(ctx, &corev1.ConfigMap{}, namespace, "postgrest-config")
	}

	deploy := resources.DesiredPostgRESTDeployment(project, r.Config)
	if deploy != nil {
		if err := r.reconcileNamespacedResource(ctx, project, deploy); err != nil {
			return err
		}
	} else {
		r.deleteIfExists(ctx, &appsv1.Deployment{}, namespace, "postgrest")
	}

	svc := resources.DesiredPostgRESTService(project)
	if svc != nil {
		return r.reconcileNamespacedResource(ctx, project, svc)
	}
	r.deleteIfExists(ctx, &corev1.Service{}, namespace, "postgrest")
	return nil
}

func (r *ProjectReconciler) reconcileHTTPRoute(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	route := resources.DesiredHTTPRoute(project, r.Config)
	if route == nil {
		// PostgREST disabled: clean up existing HTTPRoute
		namespace := "project-" + project.Name
		return r.deleteUnstructuredIfExists(ctx, resources.HTTPRouteGVK(), namespace, "api-route")
	}
	return r.reconcileUnstructured(ctx, route)
}

func (r *ProjectReconciler) reconcileTLSRoute(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	route := resources.DesiredTLSRoute(project, r.Config)
	if route == nil {
		// Postgres disabled: clean up existing TLSRoute
		namespace := "project-" + project.Name
		return r.deleteUnstructuredIfExists(ctx, resources.TLSRouteGVK(), namespace, "db-route")
	}
	return r.reconcileUnstructured(ctx, route)
}

// reconcileNamespacedResource creates or updates a typed namespaced resource.
func (r *ProjectReconciler) reconcileNamespacedResource(ctx context.Context, project *etalbaasv1alpha1.Project, obj client.Object) error {
	existing := obj.DeepCopyObject().(client.Object)

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, existing, func() error {
		// Copy spec-like fields from desired to existing.
		// For typed objects, CreateOrUpdate merges the mutation.
		return r.copyResourceFields(obj, existing)
	})
	return err
}

// reconcileUnstructured creates or updates an unstructured resource.
func (r *ProjectReconciler) reconcileUnstructured(ctx context.Context, desired *unstructured.Unstructured) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(desired.GroupVersionKind())
	existing.SetName(desired.GetName())
	existing.SetNamespace(desired.GetNamespace())

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, existing, func() error {
		// Copy spec from desired to existing
		spec, found, _ := unstructured.NestedMap(desired.Object, "spec")
		if found {
			if err := unstructured.SetNestedMap(existing.Object, spec, "spec"); err != nil {
				return err
			}
		}

		// Copy labels
		existing.SetLabels(desired.GetLabels())
		return nil
	})
	return err
}

// copyResourceFields copies mutable fields from desired to existing.
func (r *ProjectReconciler) copyResourceFields(desired, existing client.Object) error {
	// Use JSON or type-specific copy. For simplicity, we copy labels and let
	// CreateOrUpdate handle the object identity (name, namespace, GVK).
	existing.SetLabels(desired.GetLabels())
	existing.SetAnnotations(desired.GetAnnotations())

	switch d := desired.(type) {
	case *corev1.ResourceQuota:
		e := existing.(*corev1.ResourceQuota)
		e.Spec = d.Spec
	case *corev1.LimitRange:
		e := existing.(*corev1.LimitRange)
		e.Spec = d.Spec
	case *networkingv1.NetworkPolicy:
		e := existing.(*networkingv1.NetworkPolicy)
		e.Spec = d.Spec
	case *appsv1.Deployment:
		e := existing.(*appsv1.Deployment)
		e.Spec = d.Spec
	case *corev1.Service:
		e := existing.(*corev1.Service)
		// Preserve ClusterIP on update
		e.Spec.Selector = d.Spec.Selector
		e.Spec.Ports = d.Spec.Ports
		e.Spec.Type = d.Spec.Type
	case *corev1.ConfigMap:
		e := existing.(*corev1.ConfigMap)
		e.Data = d.Data
	}
	return nil
}

// updateStatus checks component readiness and updates the Project status.
func (r *ProjectReconciler) updateStatus(ctx context.Context, project *etalbaasv1alpha1.Project) (ctrl.Result, error) {
	namespace := "project-" + project.Name
	components := &etalbaasv1alpha1.ComponentStatuses{}
	allReady := true

	// Check Postgres
	if pg := project.Spec.Stack.Postgres; pg != nil && pg.Enabled {
		pgStatus := &etalbaasv1alpha1.PostgresComponentStatus{Status: "Provisioning"}
		// Check if CNPG Cluster exists and has status
		cluster := &unstructured.Unstructured{}
		cluster.SetGroupVersionKind(resources.ClusterGVK())
		if err := r.Get(ctx, types.NamespacedName{Name: "db", Namespace: namespace}, cluster); err == nil {
			phase, _, _ := unstructured.NestedString(cluster.Object, "status", "phase")
			if phase == "Cluster in healthy state" {
				pgStatus.Status = "Ready"
				pgStatus.PrimaryEndpoint = "db-rw." + namespace + ".svc.cluster.local:5432"
				if pg.Pooler != nil && pg.Pooler.Enabled {
					pgStatus.PoolerEndpoint = "db-pooler-rw." + namespace + ".svc.cluster.local:5432"
				}
			} else {
				allReady = false
			}
		} else {
			allReady = false
		}
		components.Postgres = pgStatus
	}

	// Check Redis
	if redis := project.Spec.Stack.Redis; redis != nil && redis.Enabled {
		redisStatus := &etalbaasv1alpha1.ComponentStatus{Status: "Provisioning"}
		deploy := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: "redis", Namespace: namespace}, deploy); err == nil {
			if deploy.Status.ReadyReplicas > 0 {
				redisStatus.Status = "Ready"
			} else {
				allReady = false
			}
		} else {
			allReady = false
		}
		components.Redis = redisStatus
	}

	// Check PostgREST
	if pr := project.Spec.Stack.PostgREST; pr != nil && pr.Enabled {
		prStatus := &etalbaasv1alpha1.ComponentStatus{Status: "Provisioning"}
		deploy := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: "postgrest", Namespace: namespace}, deploy); err == nil {
			if deploy.Status.ReadyReplicas > 0 {
				prStatus.Status = "Ready"
			} else {
				allReady = false
			}
		} else {
			allReady = false
		}
		components.PostgREST = prStatus
	}

	// Check CDC
	if pg := project.Spec.Stack.Postgres; pg != nil && pg.Enabled {
		cdcStatus := &etalbaasv1alpha1.ComponentStatus{Status: "Provisioning"}
		deploy := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: "cdc", Namespace: namespace}, deploy); err == nil {
			if deploy.Status.ReadyReplicas > 0 {
				cdcStatus.Status = "Ready"
			} else {
				allReady = false
			}
		} else {
			allReady = false
		}
		components.CDC = cdcStatus
	}

	// Determine phase
	if allReady {
		project.Status.Phase = etalbaasv1alpha1.ProjectPhaseReady
	} else {
		project.Status.Phase = etalbaasv1alpha1.ProjectPhaseProvisioning
	}

	project.Status.Components = components
	project.Status.ObservedGeneration = project.Generation

	// Set endpoints
	if allReady {
		subdomain := project.Spec.Networking.Subdomain
		dbSubdomain := project.Spec.Networking.DbSubdomain
		if dbSubdomain == "" {
			dbSubdomain = subdomain
		}
		project.Status.Endpoints = &etalbaasv1alpha1.EndpointStatus{
			RestApi:            "https://" + subdomain + ".api." + r.Config.BaseDomain + "/rest/v1",
			StorageApi:         "https://" + subdomain + ".api." + r.Config.BaseDomain + "/storage",
			DbHost:             dbSubdomain + ".db." + r.Config.BaseDomain,
			DbConnectionString: "postgresql://app@" + dbSubdomain + ".db." + r.Config.BaseDomain + ":5432/postgres?sslmode=require&sslnegotiation=direct",
		}
	}

	// Update conditions
	condition := metav1.Condition{
		Type:               "Provisioned",
		ObservedGeneration: project.Generation,
		LastTransitionTime: metav1.Now(),
	}
	if allReady {
		condition.Status = metav1.ConditionTrue
		condition.Reason = "AllComponentsReady"
		condition.Message = "All project components are ready"
	} else {
		condition.Status = metav1.ConditionFalse
		condition.Reason = "ComponentsProvisioning"
		condition.Message = "Some components are still provisioning"
	}
	meta.SetStatusCondition(&project.Status.Conditions, condition)

	if err := r.Status().Update(ctx, project); err != nil {
		return ctrl.Result{}, err
	}

	if !allReady {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}
	return ctrl.Result{}, nil
}

// setFailed sets the project phase to Failed and returns an appropriate result.
func (r *ProjectReconciler) setFailed(ctx context.Context, project *etalbaasv1alpha1.Project, reason string, err error) (ctrl.Result, error) {
	project.Status.Phase = etalbaasv1alpha1.ProjectPhaseFailed
	condition := metav1.Condition{
		Type:               "Provisioned",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: project.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	}
	meta.SetStatusCondition(&project.Status.Conditions, condition)

	if statusErr := r.Status().Update(ctx, project); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// deleteIfExists deletes a typed resource if it exists, ignoring NotFound errors.
func (r *ProjectReconciler) deleteIfExists(ctx context.Context, obj client.Object, namespace, name string) {
	key := types.NamespacedName{Name: name, Namespace: namespace}
	if err := r.Get(ctx, key, obj); err == nil {
		_ = r.Delete(ctx, obj)
	}
}

// deleteUnstructuredIfExists deletes an unstructured resource if it exists.
func (r *ProjectReconciler) deleteUnstructuredIfExists(ctx context.Context, gvk schema.GroupVersionKind, namespace, name string) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(gvk)
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing); err != nil {
		return client.IgnoreNotFound(err)
	}
	return r.Delete(ctx, existing)
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etalbaasv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
