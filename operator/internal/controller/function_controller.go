package controller

import (
	"context"
	"fmt"
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
	"github.com/kou-etal/etalbaas/operator/internal/resources"
)

const functionFinalizer = "etalbaas.io/function-finalizer"

// FunctionReconciler reconciles a Function object.
type FunctionReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Config    config.OperatorConfig
	NATSAdmin *natsadmin.NATSAdmin
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

	// 4. Verify parent Project exists
	var project etalbaasv1alpha1.Project
	if err := r.Get(ctx, types.NamespacedName{
		Name:      fn.Spec.ProjectRef.Name,
		Namespace: fn.Namespace,
	}, &project); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("parent Project not found, requeueing")
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	// 4. Build phase: determine if a build is needed
	needsBuild := fn.Status.ObservedGeneration < fn.Generation

	if needsBuild || (fn.Status.Phase == etalbaasv1alpha1.FunctionPhaseBuilding) {
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

	// 7. Reconcile Service
	if err := r.reconcileFunctionService(ctx, &fn); err != nil {
		logger.Error(err, "failed to reconcile function service")
		return r.setFunctionFailed(ctx, &fn, "ServiceFailed", err)
	}

	// 8. Reconcile NATS consumer (for DatabaseChange triggers)
	if err := r.reconcileNATSConsumer(ctx, &fn); err != nil {
		logger.Error(err, "failed to reconcile NATS consumer")
		return r.setFunctionFailed(ctx, &fn, "NATSConsumerFailed", err)
	}

	// 9. Reconcile autoscaling (HPA, KEDA ScaledObject, or HTTPScaledObject)
	if err := r.reconcileAutoscaling(ctx, &fn); err != nil {
		logger.Error(err, "failed to reconcile autoscaling")
		return r.setFunctionFailed(ctx, &fn, "AutoscaleFailed", err)
	}

	// 10. Reconcile HTTPRoute
	if err := r.reconcileFunctionHTTPRoute(ctx, &fn); err != nil {
		logger.Error(err, "failed to reconcile function HTTPRoute")
		return r.setFunctionFailed(ctx, &fn, "HTTPRouteFailed", err)
	}

	// 10. Update status to Ready
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
			// Build succeeded
			logger.Info("build succeeded", "job", job.Name)
			fn.Status.Phase = etalbaasv1alpha1.FunctionPhaseReady
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

	// No active job - create new build job
	logger.Info("creating build job", "function", funcName)
	buildJob := build.KanikoBuildJob(fn, r.Config)

	if err := r.Create(ctx, buildJob); err != nil {
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
func (r *FunctionReconciler) reconcileWorkload(ctx context.Context, fn *etalbaasv1alpha1.Function, imageRef string) error {
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

// reconcileNATSConsumer ensures the NATS JetStream consumer exists for DatabaseChange triggers.
func (r *FunctionReconciler) reconcileNATSConsumer(ctx context.Context, fn *etalbaasv1alpha1.Function) error {
	if r.NATSAdmin == nil {
		if resources.HasDatabaseChangeTrigger(fn) {
			logger := ctrl.LoggerFrom(ctx)
			logger.Info("WARNING: NATS admin not configured, skipping consumer reconciliation for DatabaseChange trigger",
				"function", fn.Name)
		}
		return nil
	}

	projectID := fn.Spec.ProjectRef.Name
	consumerName := natsadmin.ConsumerName(fn.Name)
	streamName := natsadmin.StreamName(projectID)

	if !resources.HasDatabaseChangeTrigger(fn) {
		// No DatabaseChange triggers — delete consumer if it exists.
		return r.NATSAdmin.DeleteConsumer(streamName, consumerName)
	}

	// Build filter subjects from triggers.
	var triggers []natsadmin.TriggerInfo
	for _, t := range fn.Spec.Triggers {
		if t.Type == "DatabaseChange" && t.DatabaseChange != nil {
			triggers = append(triggers, natsadmin.TriggerInfo{
				Table:      t.DatabaseChange.Table,
				Operations: t.DatabaseChange.Operations,
			})
		}
	}

	filterSubjects := natsadmin.BuildFilterSubjects("events.database", projectID, triggers)

	return r.NATSAdmin.EnsureConsumer(natsadmin.ConsumerConfig{
		StreamName:     streamName,
		ConsumerName:   consumerName,
		FilterSubjects: filterSubjects,
	})
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
		spec, found, _ := unstructured.NestedMap(desired.Object, "spec")
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
	// Verify Deployment readiness before marking Ready (skip for heavy-job which has no Deployment).
	if fn.Spec.Kind != etalbaasv1alpha1.FunctionKindHeavyJob {
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
	fn.Status.Execution = &etalbaasv1alpha1.ExecutionStatus{
		RuntimeClass: "gvisor",
	}

	if isGPUSelfManaged(fn) {
		fn.Status.Execution.RuntimeClass = "nvidia"
		fn.Status.Execution.GpuProvider = "self-managed"
	}

	switch fn.Spec.Kind {
	case etalbaasv1alpha1.FunctionKindHeavyJob:
		fn.Status.Execution.Mode = "Job"
		fn.Status.Execution.JobTemplate = "func-" + fn.Name + "-template"
	default:
		fn.Status.Execution.Mode = "Deployment"
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

// SetupWithManager sets up the controller with the Manager.
func (r *FunctionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etalbaasv1alpha1.Function{}).
		Named("function").
		Complete(r)
}
