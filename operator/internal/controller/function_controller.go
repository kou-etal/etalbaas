package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/build"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/natsadmin"
	gpuprovider "github.com/kou-etal/etalbaas/operator/internal/provider/gpu"
	"github.com/kou-etal/etalbaas/operator/internal/resources"
)

const functionFinalizer = "etalbaas.io/function-finalizer"

// FunctionReconciler reconciles a Function object.
type FunctionReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Config     config.OperatorConfig
	NATSAdmin  *natsadmin.NATSAdmin
	GPUFactory *gpuprovider.Factory
}

// +kubebuilder:rbac:groups=etalbaas.io,resources=functions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=etalbaas.io,resources=functions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=etalbaas.io,resources=functions/finalizers,verbs=update
// +kubebuilder:rbac:groups=etalbaas.io,resources=projects,verbs=get;list;watch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services;configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=autoscaling,resources=horizontalpodautoscalers,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=referencegrants,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=gateway.envoyproxy.io,resources=backendtrafficpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=keda.sh,resources=scaledobjects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=http.keda.sh,resources=httpscaledobjects,verbs=get;list;watch;create;update;patch;delete

// Reconcile handles the reconciliation loop for Function resources.
func (r *FunctionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch Function CR
	var fn etalbaasv1alpha1.Function
	if err := r.Get(ctx, req.NamespacedName, &fn); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// 2. Handle finalizer and deletion
	if !fn.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, &fn)
	}

	if !controllerutil.ContainsFinalizer(&fn, functionFinalizer) {
		controllerutil.AddFinalizer(&fn, functionFinalizer)
		if err := r.Update(ctx, &fn); err != nil {
			return ctrl.Result{}, err
		}
	}

	// 3. Enforce namespace boundary: Function must reside in project-{projectRef.name}
	expectedNamespace := "project-" + fn.Spec.ProjectRef.Name
	if fn.Namespace != expectedNamespace {
		err := fmt.Errorf("function namespace %q does not match expected %q for project %q",
			fn.Namespace, expectedNamespace, fn.Spec.ProjectRef.Name)
		logger.Error(err, "namespace boundary violation")
		return r.setFunctionFailed(ctx, &fn, "NamespaceMismatch", err)
	}

	// 4. Verify parent Project exists (Project CRs live in the platform namespace)
	var project etalbaasv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{
		Name:      fn.Spec.ProjectRef.Name,
		Namespace: r.Config.PlatformNamespace,
	}, &project); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("parent Project not found, requeueing")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// 4. Validate GPU configuration (before build to fail fast)
	if err := r.validateGPUConfig(&fn); err != nil {
		logger.Error(err, "GPU configuration invalid")
		return r.setFunctionFailed(ctx, &fn, "GPUConfigInvalid", err)
	}

	// 5. Build phase: determine if a build is needed
	needsBuild := fn.Status.ObservedGeneration < fn.Generation

	buildSucceeded := fn.Status.Build != nil && fn.Status.Build.Status == etalbaasv1alpha1.BuildStatusSucceeded
	if needsBuild || (fn.Status.Phase == etalbaasv1alpha1.FunctionPhaseBuilding && !buildSucceeded) {
		return r.reconcileBuild(ctx, &fn)
	}

	// 5. Workload reconcile (only if image is ready)
	if fn.Status.Build == nil || fn.Status.Build.ImageRef == "" {
		logger.Info("no image available, waiting for build")
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	imageRef := fn.Status.Build.ImageRef

	// 6. Reconcile workload (Deployment for light/heavy-deployment)
	if err := r.reconcileWorkload(ctx, &fn, imageRef); err != nil {
		logger.Error(err, "failed to reconcile workload")
		return r.setFunctionFailed(ctx, &fn, "WorkloadFailed", err)
	}

	// External GPU functions route invocations through Function MS → Dispatcher Job.
	// No in-cluster Deployment/Service/autoscaling, but need HTTPRoute → Function MS.
	if resources.IsExternalGPU(&fn) {
		logger.Info("external GPU function, creating GPU invoker route via Function MS")
		if err := r.reconcileGPUHTTPRoute(ctx, &fn); err != nil {
			return r.setFunctionFailed(ctx, &fn, "GPUHTTPRouteFailed", err)
		}
		if err := r.reconcileGPUReferenceGrant(ctx, &fn); err != nil {
			return r.setFunctionFailed(ctx, &fn, "GPUReferenceGrantFailed", err)
		}
		if err := r.reconcileGPUBackendTrafficPolicy(ctx, &fn); err != nil {
			return r.setFunctionFailed(ctx, &fn, "GPUBackendTrafficPolicyFailed", err)
		}
		return r.updateFunctionStatus(ctx, &fn)
	}

	// 7. Reconcile Service
	if err := r.reconcileFunctionService(ctx, &fn); err != nil {
		logger.Error(err, "failed to reconcile function service")
		return r.setFunctionFailed(ctx, &fn, "ServiceFailed", err)
	}

	// 8. Reconcile NATS consumer (for event-driven triggers: DatabaseChange + ObjectStorage)
	if err := r.reconcileNATSConsumer(ctx, &fn); err != nil {
		logger.Error(err, "failed to reconcile NATS consumer")
		return r.setFunctionFailed(ctx, &fn, "NATSConsumerFailed", err)
	}

	// 9. Reconcile autoscaling (HPA, KEDA ScaledObject, or HTTPScaledObject)
	if err := r.reconcileAutoscaling(ctx, &fn); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("autoscaling conflict, requeueing", "error", err.Error())
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		logger.Error(err, "failed to reconcile autoscaling")
		return r.setFunctionFailed(ctx, &fn, "AutoscaleFailed", err)
	}

	// 10. Reconcile HTTPRoute
	if err := r.reconcileFunctionHTTPRoute(ctx, &fn); err != nil {
		if apierrors.IsConflict(err) {
			logger.Info("HTTPRoute conflict, requeueing", "error", err.Error())
			return ctrl.Result{RequeueAfter: 2 * time.Second}, nil
		}
		logger.Error(err, "failed to reconcile function HTTPRoute")
		return r.setFunctionFailed(ctx, &fn, "HTTPRouteFailed", err)
	}

	// 11. Update status to Ready
	return r.updateFunctionStatus(ctx, &fn)
}

// handleDeletion processes the finalizer cleanup for Function.
func (r *FunctionReconciler) handleDeletion(ctx context.Context, fn *etalbaasv1alpha1.Function) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if controllerutil.ContainsFinalizer(fn, functionFinalizer) {
		// Clean up cross-namespace build jobs in platform-system
		if err := r.cleanupBuildJobs(ctx, fn); err != nil {
			logger.Error(err, "failed to cleanup build jobs")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, err
		}

		// Clean up NATS consumer
		if r.NATSAdmin != nil {
			projectID := fn.Spec.ProjectRef.Name
			if err := r.NATSAdmin.DeleteConsumer(
				natsadmin.StreamName(projectID),
				natsadmin.ConsumerName(fn.Name),
			); err != nil {
				logger.Error(err, "failed to cleanup NATS consumer")
			}
		}

		controllerutil.RemoveFinalizer(fn, functionFinalizer)
		if err := r.Update(ctx, fn); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// cleanupBuildJobs deletes build jobs and configmaps from platform-system namespace.
func (r *FunctionReconciler) cleanupBuildJobs(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	// Delete build jobs by label
	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, client.InNamespace(r.Config.PlatformNamespace),
		client.MatchingLabels{
			"etalbaas.io/project-id": projectID,
			"etalbaas.io/function":   funcName,
			"etalbaas.io/build":      "true",
		},
	); err != nil {
		return err
	}

	propagation := metav1.DeletePropagationBackground
	for i := range jobList.Items {
		if err := r.Delete(ctx, &jobList.Items[i], &client.DeleteOptions{
			PropagationPolicy: &propagation,
		}); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	// Delete dockerfile configmaps
	cmName := fmt.Sprintf("build-%s-%s-dockerfile", projectID, funcName)
	cm := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      cmName,
		Namespace: r.Config.PlatformNamespace,
	}, cm); err == nil {
		if err := r.Delete(ctx, cm); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	// Delete source configmaps (inline/zip)
	sourceCMName := build.SourceConfigMapName(projectID, funcName)
	sourceCM := &corev1.ConfigMap{}
	if err := r.Get(ctx, types.NamespacedName{
		Name:      sourceCMName,
		Namespace: r.Config.PlatformNamespace,
	}, sourceCM); err == nil {
		if err := r.Delete(ctx, sourceCM); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	return nil
}

// reconcileBuild handles the build lifecycle.
func (r *FunctionReconciler) reconcileBuild(ctx context.Context, fn *etalbaasv1alpha1.Function) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	projectID := fn.Spec.ProjectRef.Name
	funcName := fn.Name

	// Generate Dockerfile
	dockerfile, err := build.GenerateDockerfile(
		fn.Spec.Runtime.Preset,
		fn.Spec.Runtime.Requirements,
		fn.Spec.Runtime.Dockerfile,
	)
	if err != nil {
		return r.setFunctionFailed(ctx, fn, "DockerfileFailed", err)
	}

	// Create/update Dockerfile ConfigMap
	desiredCM := build.BuildDockerfileConfigMap(fn, dockerfile, r.Config)
	existingCM := &corev1.ConfigMap{}
	cmKey := types.NamespacedName{Name: desiredCM.Name, Namespace: desiredCM.Namespace}
	if err := r.Get(ctx, cmKey, existingCM); apierrors.IsNotFound(err) {
		if err := r.Create(ctx, desiredCM); err != nil {
			return ctrl.Result{}, fmt.Errorf("create dockerfile configmap: %w", err)
		}
	} else if err != nil {
		return ctrl.Result{}, err
	} else {
		existingCM.Data = desiredCM.Data
		if err := r.Update(ctx, existingCM); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Create/update source ConfigMap for inline sources.
	// Zip source requires object storage download (not yet implemented).
	if fn.Spec.Source.Type == "inline" {
		desiredSourceCM := build.BuildSourceConfigMap(fn, r.Config)
		existingSourceCM := &corev1.ConfigMap{}
		sourceCMKey := types.NamespacedName{Name: desiredSourceCM.Name, Namespace: desiredSourceCM.Namespace}
		if err := r.Get(ctx, sourceCMKey, existingSourceCM); apierrors.IsNotFound(err) {
			if err := r.Create(ctx, desiredSourceCM); err != nil {
				return ctrl.Result{}, fmt.Errorf("create source configmap: %w", err)
			}
		} else if err != nil {
			return ctrl.Result{}, err
		} else {
			existingSourceCM.Data = desiredSourceCM.Data
			if err := r.Update(ctx, existingSourceCM); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Check for existing build job matching current generation
	jobList := &batchv1.JobList{}
	if err := r.List(ctx, jobList, client.InNamespace(r.Config.PlatformNamespace),
		client.MatchingLabels{
			"etalbaas.io/project-id": projectID,
			"etalbaas.io/function":   funcName,
			"etalbaas.io/build":      "true",
			"etalbaas.io/generation": fmt.Sprintf("%d", fn.Generation),
		},
	); err != nil {
		return ctrl.Result{}, err
	}

	// Find an active or completed build job for current generation
	expectedImage := build.ImageDestination(fn, r.Config)
	var activeJob *batchv1.Job
	for i := range jobList.Items {
		job := &jobList.Items[i]
		if job.Status.Succeeded > 0 {
			// Build succeeded — update build status but leave phase as Building.
			// Phase transitions to Ready after workload deployment is verified.
			logger.Info("build succeeded", "job", job.Name)
			fn.Status.Build = &etalbaasv1alpha1.BuildStatus{
				Status:   etalbaasv1alpha1.BuildStatusSucceeded,
				ImageRef: expectedImage,
			}
			fn.Status.ObservedGeneration = fn.Generation
			if err := r.Status().Update(ctx, fn); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		// Check if the Job has truly failed (condition type=Failed, not just a pod retry).
		if isJobFailed(job) {
			logger.Info("build failed", "job", job.Name)
			fn.Status.Phase = etalbaasv1alpha1.FunctionPhaseFailed
			fn.Status.Build = &etalbaasv1alpha1.BuildStatus{
				Status: etalbaasv1alpha1.BuildStatusFailed,
			}
			fn.Status.ObservedGeneration = fn.Generation
			if err := r.Status().Update(ctx, fn); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
		// Job is still running
		activeJob = job
	}

	if activeJob != nil {
		// Build in progress
		if fn.Status.Phase != etalbaasv1alpha1.FunctionPhaseBuilding {
			fn.Status.Phase = etalbaasv1alpha1.FunctionPhaseBuilding
			fn.Status.Build = &etalbaasv1alpha1.BuildStatus{
				Status: etalbaasv1alpha1.BuildStatusBuilding,
			}
			if err := r.Status().Update(ctx, fn); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
	}

	// Enforce concurrent build limit per project before creating a new job
	if r.Config.MaxConcurrentBuilds > 0 {
		allBuildJobs := &batchv1.JobList{}
		if err := r.List(ctx, allBuildJobs, client.InNamespace(r.Config.PlatformNamespace),
			client.MatchingLabels{
				"etalbaas.io/project-id": projectID,
				"etalbaas.io/build":      "true",
			},
		); err != nil {
			return ctrl.Result{}, fmt.Errorf("list build jobs for concurrency check: %w", err)
		}
		runningCount := 0
		for i := range allBuildJobs.Items {
			j := &allBuildJobs.Items[i]
			if j.Status.Succeeded == 0 && !isJobFailed(j) {
				runningCount++
			}
		}
		if runningCount >= r.Config.MaxConcurrentBuilds {
			logger.Info("concurrent build limit reached, requeueing",
				"running", runningCount, "limit", r.Config.MaxConcurrentBuilds)
			return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
		}
	}

	// No active job - create new build job
	logger.Info("creating build job", "function", funcName)
	buildJob, err := build.KanikoBuildJob(fn, r.Config)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("build job spec: %w", err)
	}

	if err := r.Create(ctx, buildJob); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("build job already exists, requeueing", "job", buildJob.Name)
			return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
		}
		return ctrl.Result{}, fmt.Errorf("create build job: %w", err)
	}

	// Set phase to Building
	fn.Status.Phase = etalbaasv1alpha1.FunctionPhaseBuilding
	fn.Status.Build = &etalbaasv1alpha1.BuildStatus{
		Status: etalbaasv1alpha1.BuildStatusBuilding,
	}
	if err := r.Status().Update(ctx, fn); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 15 * time.Second}, nil
}

// reconcileWorkload creates or updates the Deployment for the function.
// For heavy-job kind, cleans up stale Deployment/Service/HTTPRoute from previous kind.
// For external GPU functions, skips Deployment creation (workload runs on external provider).
func (r *FunctionReconciler) reconcileWorkload(ctx context.Context, fn *etalbaasv1alpha1.Function, imageRef string) error {
	// External GPU functions don't create a local Deployment; workload runs on the provider.
	if resources.IsExternalGPU(fn) {
		resourceName := "func-" + fn.Name
		r.deleteStaleDeployment(ctx, fn.Namespace, resourceName)
		r.deleteStaleService(ctx, fn.Namespace, resourceName)
		r.deleteStaleHTTPRoute(ctx, fn.Namespace, resourceName)
		return nil
	}

	deploy := resources.DesiredFunctionDeployment(fn, imageRef, r.Config)
	if deploy == nil {
		// heavy-job: no Deployment. Clean up stale resources from a previous kind.
		resourceName := "func-" + fn.Name
		r.deleteStaleDeployment(ctx, fn.Namespace, resourceName)
		r.deleteStaleService(ctx, fn.Namespace, resourceName)
		r.deleteStaleHTTPRoute(ctx, fn.Namespace, resourceName)
		return nil
	}

	existing := &appsv1.Deployment{}
	key := types.NamespacedName{Name: deploy.Name, Namespace: deploy.Namespace}
	if err := r.Get(ctx, key, existing); apierrors.IsNotFound(err) {
		return r.Create(ctx, deploy)
	} else if err != nil {
		return err
	}

	// Update existing deployment
	existing.Spec = deploy.Spec
	existing.Labels = deploy.Labels
	return r.Update(ctx, existing)
}

// reconcileFunctionService creates or updates the Service for the function.
// Cleans up stale Services when kind changes or HTTP triggers are removed.
func (r *FunctionReconciler) reconcileFunctionService(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	svc := resources.DesiredFunctionService(fn)
	if svc == nil {
		// No service needed: clean up if exists (kind changed or no HTTP trigger)
		r.deleteStaleService(ctx, fn.Namespace, "func-"+fn.Name)
		return nil
	}

	existing := &corev1.Service{}
	key := types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}
	if err := r.Get(ctx, key, existing); apierrors.IsNotFound(err) {
		return r.Create(ctx, svc)
	} else if err != nil {
		return err
	}

	existing.Spec.Selector = svc.Spec.Selector
	existing.Spec.Ports = svc.Spec.Ports
	existing.Labels = svc.Labels
	return r.Update(ctx, existing)
}

// reconcileNATSConsumer ensures the NATS JetStream consumer exists for event-driven triggers
// (DatabaseChange and/or ObjectStorage).
func (r *FunctionReconciler) reconcileNATSConsumer(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	if r.NATSAdmin == nil {
		if resources.HasEventTrigger(fn) {
			logger := ctrl.LoggerFrom(ctx)
			logger.Info("WARNING: NATS admin not configured, skipping consumer reconciliation for event trigger",
				"function", fn.Name)
		}
		return nil
	}

	projectID := fn.Spec.ProjectRef.Name
	consumerName := natsadmin.ConsumerName(fn.Name)
	streamName := natsadmin.StreamName(projectID)

	if !resources.HasEventTrigger(fn) {
		// No event triggers — delete consumer if it exists.
		return r.NATSAdmin.DeleteConsumer(streamName, consumerName)
	}

	// Build filter subjects from all event-driven triggers.
	var filterSubjects []string

	// DatabaseChange triggers
	var dbTriggers []natsadmin.TriggerInfo
	for _, t := range fn.Spec.Triggers {
		if t.Type == "DatabaseChange" && t.DatabaseChange != nil {
			dbTriggers = append(dbTriggers, natsadmin.TriggerInfo{
				Table:      t.DatabaseChange.Table,
				Operations: t.DatabaseChange.Operations,
			})
		}
	}
	filterSubjects = append(filterSubjects, natsadmin.BuildFilterSubjects("events.database", projectID, dbTriggers)...)

	// ObjectStorage triggers
	for _, t := range fn.Spec.Triggers {
		if t.Type == "ObjectStorage" && t.ObjectStorage != nil {
			for _, ev := range t.ObjectStorage.Events {
				op := storageEventToOp(ev)
				filterSubjects = append(filterSubjects, fmt.Sprintf(
					"events.storage.%s.project-%s.%s",
					op, projectID, t.ObjectStorage.Bucket,
				))
			}
		}
	}

	return r.NATSAdmin.EnsureConsumer(natsadmin.ConsumerConfig{
		StreamName:     streamName,
		ConsumerName:   consumerName,
		FilterSubjects: filterSubjects,
	})
}

// storageEventToOp converts an ObjectStorage event name to a NATS subject operation token.
// e.g. "ObjectCreated" → "created", "ObjectDeleted" → "deleted"
func storageEventToOp(event string) string {
	lower := strings.ToLower(event)
	lower = strings.TrimPrefix(lower, "object")
	if lower == "" {
		return "unknown"
	}
	return lower
}

// reconcileAutoscaling creates or updates HPA, KEDA ScaledObject, or HTTPScaledObject.
// Also cleans up stale autoscalers when kind or trigger type changes.
func (r *FunctionReconciler) reconcileAutoscaling(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	resourceName := "func-" + fn.Name

	// HPA for light-deployment
	hpa := resources.DesiredHPA(fn)
	if hpa != nil {
		// Delete stale KEDA resources if they exist (kind changed from heavy→light)
		_ = r.deleteStaleScaledObject(ctx, fn.Namespace, resourceName)
		_ = r.deleteStaleHTTPScaledObject(ctx, fn.Namespace, resourceName)

		existing := &autoscalingv2.HorizontalPodAutoscaler{}
		key := types.NamespacedName{Name: hpa.Name, Namespace: hpa.Namespace}
		if err := r.Get(ctx, key, existing); apierrors.IsNotFound(err) {
			return r.Create(ctx, hpa)
		} else if err != nil {
			return err
		}
		existing.Spec = hpa.Spec
		existing.Labels = hpa.Labels
		return r.Update(ctx, existing)
	}

	// KEDA ScaledObject for heavy-deployment with DatabaseChange trigger
	scaledObj := resources.DesiredKEDAScaledObject(fn, r.Config)
	if scaledObj != nil {
		_ = r.deleteStaleHPA(ctx, fn.Namespace, resourceName)
		_ = r.deleteStaleHTTPScaledObject(ctx, fn.Namespace, resourceName)
		return r.reconcileUnstructuredFunction(ctx, scaledObj)
	}

	// HTTPScaledObject for heavy-deployment with Http trigger (no DatabaseChange)
	httpScaledObj := resources.DesiredHTTPScaledObject(fn, r.Config)
	if httpScaledObj != nil {
		_ = r.deleteStaleHPA(ctx, fn.Namespace, resourceName)
		_ = r.deleteStaleScaledObject(ctx, fn.Namespace, resourceName)
		return r.reconcileUnstructuredFunction(ctx, httpScaledObj)
	}

	// Neither needed (heavy-job or no applicable triggers): clean up all.
	_ = r.deleteStaleHPA(ctx, fn.Namespace, resourceName)
	_ = r.deleteStaleScaledObject(ctx, fn.Namespace, resourceName)
	_ = r.deleteStaleHTTPScaledObject(ctx, fn.Namespace, resourceName)

	return nil
}

// deleteStaleHPA removes an HPA that is no longer needed.
func (r *FunctionReconciler) deleteStaleHPA(ctx context.Context, namespace, name string) error {
	existing := &autoscalingv2.HorizontalPodAutoscaler{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing); err != nil {
		return client.IgnoreNotFound(err)
	}
	return r.Delete(ctx, existing)
}

// deleteStaleDeployment removes a Deployment that is no longer needed (e.g., kind changed to heavy-job).
func (r *FunctionReconciler) deleteStaleDeployment(ctx context.Context, namespace, name string) {
	existing := &appsv1.Deployment{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing); err == nil {
		_ = r.Delete(ctx, existing)
	}
}

// deleteStaleService removes a Service that is no longer needed.
func (r *FunctionReconciler) deleteStaleService(ctx context.Context, namespace, name string) {
	existing := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing); err == nil {
		_ = r.Delete(ctx, existing)
	}
}

// deleteStaleHTTPRoute removes an HTTPRoute that is no longer needed.
func (r *FunctionReconciler) deleteStaleHTTPRoute(ctx context.Context, namespace, name string) {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(resources.HTTPRouteGVK())
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing); err == nil {
		_ = r.Delete(ctx, existing)
	}
}

// deleteStaleHTTPScaledObject removes a KEDA HTTPScaledObject that is no longer needed.
func (r *FunctionReconciler) deleteStaleHTTPScaledObject(ctx context.Context, namespace, name string) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(resources.KEDAHTTPScaledObjectGVK())
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing); err != nil {
		return client.IgnoreNotFound(err)
	}
	return r.Delete(ctx, existing)
}

// deleteStaleScaledObject removes a KEDA ScaledObject that is no longer needed.
func (r *FunctionReconciler) deleteStaleScaledObject(ctx context.Context, namespace, name string) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(resources.KEDAScaledObjectGVK())
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, existing); err != nil {
		return client.IgnoreNotFound(err)
	}
	return r.Delete(ctx, existing)
}

// reconcileFunctionHTTPRoute creates or updates the HTTPRoute for the function.
// Cleans up stale HTTPRoutes when kind changes or HTTP triggers are removed.
func (r *FunctionReconciler) reconcileFunctionHTTPRoute(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	route := resources.DesiredFunctionHTTPRoute(fn, r.Config)
	if route == nil {
		// No route needed: clean up if exists
		r.deleteStaleHTTPRoute(ctx, fn.Namespace, "func-"+fn.Name)
		return nil
	}
	return r.reconcileUnstructuredFunction(ctx, route)
}

// reconcileUnstructuredFunction creates or updates an unstructured resource for functions.
func (r *FunctionReconciler) reconcileUnstructuredFunction(ctx context.Context, desired *unstructured.Unstructured) error {
	existing := &unstructured.Unstructured{}
	existing.SetGroupVersionKind(desired.GroupVersionKind())
	existing.SetName(desired.GetName())
	existing.SetNamespace(desired.GetNamespace())

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, existing, func() error {
		spec, found, nestedErr := unstructured.NestedMap(desired.Object, "spec")
		if nestedErr != nil {
			return fmt.Errorf("read spec from desired: %w", nestedErr)
		}
		if found {
			if err := unstructured.SetNestedMap(existing.Object, spec, "spec"); err != nil {
				return err
			}
		}
		existing.SetLabels(desired.GetLabels())
		return nil
	})
	return err
}

// updateFunctionStatus sets the function status to Ready after verifying Deployment readiness.
func (r *FunctionReconciler) updateFunctionStatus(ctx context.Context, fn *etalbaasv1alpha1.Function) (ctrl.Result, error) {
	// External GPU: no local Deployment to check.
	// Self-managed GPU + heavy-job: no Deployment to check.
	// All others: verify Deployment readiness.
	isExtGPU := resources.IsExternalGPU(fn)
	if !isExtGPU && fn.Spec.Kind != etalbaasv1alpha1.FunctionKindHeavyJob {
		deploy := &appsv1.Deployment{}
		deployKey := types.NamespacedName{Name: "func-" + fn.Name, Namespace: fn.Namespace}
		if err := r.Get(ctx, deployKey, deploy); err != nil {
			if apierrors.IsNotFound(err) {
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}
			return ctrl.Result{}, err
		}
		// For heavy-deployment (min=0), ReadyReplicas may be 0 — that's expected.
		// For light-deployment (min>=1), at least 1 replica must be available.
		if fn.Spec.Kind == etalbaasv1alpha1.FunctionKindLightDeployment && deploy.Status.ReadyReplicas < 1 {
			logger := log.FromContext(ctx)
			logger.Info("deployment not ready yet, requeueing", "readyReplicas", deploy.Status.ReadyReplicas)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
	}

	fn.Status.Phase = etalbaasv1alpha1.FunctionPhaseReady
	fn.Status.ObservedGeneration = fn.Generation

	// Set execution status
	rc := r.Config.SandboxRuntimeClass
	if rc == "" {
		rc = "none"
	}
	fn.Status.Execution = &etalbaasv1alpha1.ExecutionStatus{
		RuntimeClass: rc,
	}

	if isExtGPU {
		fn.Status.Execution.RuntimeClass = rc // Dispatcher is CPU only
		fn.Status.Execution.GpuProvider = fn.Spec.GPU.Provider
		fn.Status.Execution.Mode = "ExternalGPU"
	} else if isGPUSelfManaged(fn) {
		smCfg := resources.ResolveSelfManagedConfig(r.Config)
		fn.Status.Execution.RuntimeClass = smCfg.RuntimeClass
		fn.Status.Execution.GpuProvider = "self-managed"
	}

	// Set execution mode if not already set by GPU handling above
	if fn.Status.Execution.Mode == "" {
		switch fn.Spec.Kind {
		case etalbaasv1alpha1.FunctionKindHeavyJob:
			fn.Status.Execution.Mode = "Job"
			fn.Status.Execution.JobTemplate = "func-" + fn.Name + "-template"
		default:
			fn.Status.Execution.Mode = "Deployment"
		}
	}

	// Set trigger statuses
	fn.Status.Triggers = nil
	for _, trigger := range fn.Spec.Triggers {
		ts := etalbaasv1alpha1.TriggerStatus{
			Type:   trigger.Type,
			Status: "Active",
		}
		if trigger.Type == "Http" {
			projectID := fn.Spec.ProjectRef.Name
			ts.Endpoint = "https://" + projectID + ".api." + r.Config.BaseDomain + "/functions/" + fn.Name + "/invoke"
		}
		fn.Status.Triggers = append(fn.Status.Triggers, ts)
	}

	// Set condition
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		ObservedGeneration: fn.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             "WorkloadReady",
		Message:            "Function workload is ready",
	}
	meta.SetStatusCondition(&fn.Status.Conditions, condition)

	if err := r.Status().Update(ctx, fn); err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// setFunctionFailed sets the function phase to Failed.
func (r *FunctionReconciler) setFunctionFailed(ctx context.Context, fn *etalbaasv1alpha1.Function, reason string, err error) (ctrl.Result, error) {
	fn.Status.Phase = etalbaasv1alpha1.FunctionPhaseFailed
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionFalse,
		ObservedGeneration: fn.Generation,
		LastTransitionTime: metav1.Now(),
		Reason:             reason,
		Message:            err.Error(),
	}
	meta.SetStatusCondition(&fn.Status.Conditions, condition)

	if statusErr := r.Status().Update(ctx, fn); statusErr != nil {
		return ctrl.Result{}, statusErr
	}
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// isJobFailed checks whether a Job has truly failed (the Job controller has given up),
// as opposed to having a failed pod that is still being retried under backoffLimit.
func isJobFailed(job *batchv1.Job) bool {
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func isGPUSelfManaged(fn *etalbaasv1alpha1.Function) bool {
	return fn.Spec.GPU != nil && fn.Spec.GPU.Required && fn.Spec.GPU.Provider == "self-managed"
}

// validateGPUConfig validates that the function's GPU configuration is compatible
// with the enabled GPU providers. Returns nil if GPU is not required.
// Self-managed GPU is always allowed (handled directly in workload builder).
// External GPU providers require the factory to be configured and the provider enabled.
func (r *FunctionReconciler) validateGPUConfig(fn *etalbaasv1alpha1.Function) error {
	if fn.Spec.GPU == nil || !fn.Spec.GPU.Required {
		return nil
	}
	// Self-managed GPU is always allowed; it doesn't need an external provider.
	if fn.Spec.GPU.Provider == "self-managed" {
		return nil
	}
	// External GPU provider: requires factory to be configured.
	if r.GPUFactory == nil || !r.GPUFactory.IsEnabled() {
		return fmt.Errorf("function requires external GPU provider %q but GPU is disabled in platform configuration", fn.Spec.GPU.Provider)
	}
	_, err := r.GPUFactory.ResolveProvider(fn.Spec.GPU.Provider, fn.Spec.GPU.Product)
	if err != nil {
		return fmt.Errorf("GPU provider unavailable: %w", err)
	}
	return nil
}

// reconcileGPUHTTPRoute creates or updates the HTTPRoute that routes GPU function
// invocations to Function MS (cross-namespace via ReferenceGrant).
func (r *FunctionReconciler) reconcileGPUHTTPRoute(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	route := resources.DesiredGPUFunctionHTTPRoute(fn, r.Config)
	return r.reconcileUnstructuredFunction(ctx, route)
}

// reconcileGPUReferenceGrant ensures a ReferenceGrant exists in the platform namespace
// allowing this project's HTTPRoute to reference the Function MS Service.
func (r *FunctionReconciler) reconcileGPUReferenceGrant(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	grant := resources.DesiredGPUReferenceGrant(fn.Namespace, r.Config)
	return r.reconcileUnstructuredFunction(ctx, grant)
}

// reconcileGPUBackendTrafficPolicy creates or updates the BackendTrafficPolicy
// that extends Envoy Gateway timeout to 300s for GPU function HTTPRoutes.
func (r *FunctionReconciler) reconcileGPUBackendTrafficPolicy(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	btp := resources.DesiredGPUBackendTrafficPolicy(fn)
	return r.reconcileUnstructuredFunction(ctx, btp)
}

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etalbaasv1alpha1.Function{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&autoscalingv2.HorizontalPodAutoscaler{}).
		Named("function").
		Complete(r)
}
