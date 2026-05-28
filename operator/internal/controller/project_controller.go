package controller

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/hex"
	"fmt"
	"strings"
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

	jwtlib "github.com/golang-jwt/jwt/v5"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/natsadmin"
	"github.com/kou-etal/etalbaas/operator/internal/resources"
)

const projectFinalizer = "etalbaas.io/project-finalizer"

// ProjectReconciler reconciles a Project object.
type ProjectReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Config    config.OperatorConfig
	NATSAdmin *natsadmin.NATSAdmin
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

	// 3. Enforce global project total limit
	if r.Config.MaxTotalProjects > 0 && (project.Status.Phase == "" || project.Status.Phase == etalbaasv1alpha1.ProjectPhasePending) {
		var projectList etalbaasv1alpha1.ProjectList
		if err := r.List(ctx, &projectList, client.InNamespace(r.Config.PlatformNamespace)); err != nil {
			return ctrl.Result{}, fmt.Errorf("list projects for quota check: %w", err)
		}
		// Count non-deleting projects (exclude this one from the count)
		activeCount := 0
		for i := range projectList.Items {
			p := &projectList.Items[i]
			if p.DeletionTimestamp.IsZero() && p.Name != project.Name {
				activeCount++
			}
		}
		if activeCount >= r.Config.MaxTotalProjects {
			logger.Info("global project limit reached, queueing project",
				"active", activeCount, "limit", r.Config.MaxTotalProjects)
			project.Status.Phase = etalbaasv1alpha1.ProjectPhaseQueued
			if err := r.Status().Update(ctx, &project); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}
	}

	// 4. Set phase to Provisioning if Pending
	if project.Status.Phase == "" || project.Status.Phase == etalbaasv1alpha1.ProjectPhasePending || project.Status.Phase == etalbaasv1alpha1.ProjectPhaseQueued {
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

	if err := r.reconcileCDCSecret(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile CDC secret")
		return r.setFailed(ctx, &project, "CDCSecretFailed", err)
	}

	if err := r.reconcilePostgres(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile postgres")
		return r.setFailed(ctx, &project, "PostgresFailed", err)
	}

	// NATS stream must exist before CDC deployment starts publishing.
	if err := r.reconcileNATSStream(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile NATS stream")
		return r.setFailed(ctx, &project, "NATSStreamFailed", err)
	}

	if err := r.reconcileCDC(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile CDC")
		return r.setFailed(ctx, &project, "CDCFailed", err)
	}

	if err := r.reconcileRedis(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile redis")
		return r.setFailed(ctx, &project, "RedisFailed", err)
	}

	if err := r.reconcileJWTSecret(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile JWT secret")
		return r.setFailed(ctx, &project, "JWTSecretFailed", err)
	}

	if err := r.reconcileJWTKeys(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile JWT keys")
		return r.setFailed(ctx, &project, "JWTKeysFailed", err)
	}

	if err := r.reconcilePostgREST(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile PostgREST")
		return r.setFailed(ctx, &project, "PostgRESTFailed", err)
	}

	if err := r.reconcilePostgresMeta(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile postgres-meta")
		return r.setFailed(ctx, &project, "PostgresMetaFailed", err)
	}

	if err := r.reconcileGoTrue(ctx, &project); err != nil {
		logger.Error(err, "failed to reconcile GoTrue")
		return r.setFailed(ctx, &project, "GoTrueFailed", err)
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
		// Clean up NATS stream — block finalizer removal on failure to prevent orphan streams.
		if r.NATSAdmin != nil {
			if err := r.NATSAdmin.DeleteStream(project.Name); err != nil {
				logger.Error(err, "failed to cleanup NATS stream, requeueing")
				return ctrl.Result{RequeueAfter: 10 * time.Second}, err
			}
		}

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
		resources.DesiredAllowCNPGNetworkPolicy(project),
		resources.DesiredAllowEnvoyGatewayNetworkPolicy(project, "envoy-gateway-system"),
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

func (r *ProjectReconciler) reconcilePostgresMeta(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name

	deploy := resources.DesiredPostgresMetaDeployment(project, r.Config)
	if deploy != nil {
		if err := r.reconcileNamespacedResource(ctx, project, deploy); err != nil {
			return err
		}
	} else {
		r.deleteIfExists(ctx, &appsv1.Deployment{}, namespace, "postgres-meta")
	}

	svc := resources.DesiredPostgresMetaService(project)
	if svc != nil {
		return r.reconcileNamespacedResource(ctx, project, svc)
	}
	r.deleteIfExists(ctx, &corev1.Service{}, namespace, "postgres-meta")
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

// reconcileNATSStream ensures the NATS JetStream stream exists for a project with Postgres.
func (r *ProjectReconciler) reconcileNATSStream(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	if r.NATSAdmin == nil {
		return nil
	}

	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return r.NATSAdmin.DeleteStream(project.Name)
	}
	return r.NATSAdmin.EnsureStream(project.Name)
}

// reconcileCDCSecret creates the CDC user password Secret if it doesn't exist.
// CNPG managed.roles references this Secret to set the CDC PostgreSQL user's password.
func (r *ProjectReconciler) reconcileCDCSecret(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		r.deleteIfExists(ctx, &corev1.Secret{}, namespace, "db-cdc")
		return nil
	}

	// Idempotent: only create if absent.
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: "db-cdc", Namespace: namespace}, existing)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("check CDC secret: %w", err)
	}

	password, err := generateRandomPassword(32)
	if err != nil {
		return fmt.Errorf("generate CDC password: %w", err)
	}

	userID := project.Labels[resources.LabelUserID]
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-cdc",
			Namespace: namespace,
			Labels:    resources.ComponentLabels(project.Name, userID, project.Spec.Plan, "cdc"),
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"password": password,
			"username": "cdc",
			"host":     "db-rw",
			"dbname":   "postgres",
		},
	}

	return r.Create(ctx, secret)
}

// reconcileJWTSecret creates the project-jwt-secret Secret for GoTrue's GOTRUE_JWT_SECRET.
// Each project gets a unique random symmetric key to prevent cross-project JWT forgery.
func (r *ProjectReconciler) reconcileJWTSecret(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		r.deleteIfExists(ctx, &corev1.Secret{}, namespace, "project-jwt-secret")
		return nil
	}

	// Idempotent: only create if absent.
	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: "project-jwt-secret", Namespace: namespace}, existing)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("check JWT secret: %w", err)
	}

	// Generate a unique 32-byte random secret per project.
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return fmt.Errorf("generate JWT secret: %w", err)
	}

	userID := project.Labels[resources.LabelUserID]
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "project-jwt-secret",
			Namespace: namespace,
			Labels:    resources.ComponentLabels(project.Name, userID, project.Spec.Plan, "postgrest"),
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"secret": hex.EncodeToString(secretBytes),
		},
	}

	return r.Create(ctx, secret)
}

// reconcileJWTKeys generates RSA key pair and pre-signed JWTs for GoTrue RS256 authentication.
// Creates four Secrets in the project namespace (idempotent, skips if gotrue-jwt-keys already exists):
//   - gotrue-jwt-keys: JWK Set (private key) for GoTrue GOTRUE_JWT_KEYS
//   - jwt-verification-key: JWK (public key) for PostgREST PGRST_JWT_SECRET
//   - service-role-key: RS256 signed JWT with role=service_role
//   - anon-key: RS256 signed JWT with role=anon
func (r *ProjectReconciler) reconcileJWTKeys(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name
	pg := project.Spec.Stack.Postgres
	if pg == nil || !pg.Enabled {
		return nil
	}

	// Check if gotrue-jwt-keys already exists (idempotent).
	existing := &corev1.Secret{}
	var rsaKey *rsa.PrivateKey
	err := r.Get(ctx, types.NamespacedName{Name: "gotrue-jwt-keys", Namespace: namespace}, existing)
	if err == nil {
		// Keys already exist. Ensure derived Secrets also exist.
		jwkSetData := existing.Data["jwk-set"]
		if len(jwkSetData) == 0 {
			return fmt.Errorf("gotrue-jwt-keys Secret exists but jwk-set is empty")
		}
		rsaKey, err = resources.ParseRSAPrivateKeyFromJWKSet(jwkSetData)
		if err != nil {
			return fmt.Errorf("parse RSA key from existing Secret: %w", err)
		}
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("check gotrue-jwt-keys: %w", err)
	} else {
		// Generate new RSA key pair.
		rsaKey, err = resources.GenerateRSAKeyPair()
		if err != nil {
			return fmt.Errorf("generate RSA key pair: %w", err)
		}

		jwkSet, err := resources.PrivateKeyToJWKSet(rsaKey)
		if err != nil {
			return fmt.Errorf("convert key to JWK Set: %w", err)
		}

		userID := project.Labels[resources.LabelUserID]
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "gotrue-jwt-keys",
				Namespace: namespace,
				Labels:    resources.ComponentLabels(project.Name, userID, project.Spec.Plan, "gotrue"),
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"jwk-set": jwkSet,
			},
		}
		if err := r.Create(ctx, secret); err != nil {
			return fmt.Errorf("create gotrue-jwt-keys: %w", err)
		}
	}

	// Ensure jwt-verification-key (public key for PostgREST).
	if err := r.ensureJWTVerificationKey(ctx, project, rsaKey); err != nil {
		return err
	}

	// Ensure service-role-key.
	if err := r.ensureRoleKeySecret(ctx, project, rsaKey, "service-role-key", resources.BuildServiceRoleClaims()); err != nil {
		return err
	}

	// Ensure anon-key.
	if err := r.ensureRoleKeySecret(ctx, project, rsaKey, "anon-key", resources.BuildAnonClaims()); err != nil {
		return err
	}

	return nil
}

func (r *ProjectReconciler) ensureJWTVerificationKey(ctx context.Context, project *etalbaasv1alpha1.Project, rsaKey *rsa.PrivateKey) error {
	namespace := "project-" + project.Name

	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: "jwt-verification-key", Namespace: namespace}, existing)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return fmt.Errorf("check jwt-verification-key: %w", err)
	}

	jwk, err := resources.PublicKeyToJWK(&rsaKey.PublicKey)
	if err != nil {
		return fmt.Errorf("convert public key to JWK: %w", err)
	}

	userID := project.Labels[resources.LabelUserID]
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jwt-verification-key",
			Namespace: namespace,
			Labels:    resources.ComponentLabels(project.Name, userID, project.Spec.Plan, "gotrue"),
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"jwk": jwk,
		},
	}
	return r.Create(ctx, secret)
}

func (r *ProjectReconciler) ensureRoleKeySecret(ctx context.Context, project *etalbaasv1alpha1.Project, rsaKey *rsa.PrivateKey, secretName string, claims map[string]interface{}) error {
	namespace := "project-" + project.Name

	existing := &corev1.Secret{}
	err := r.Get(ctx, types.NamespacedName{Name: secretName, Namespace: namespace}, existing)
	if err == nil {
		// Check if the existing JWT is expiring soon (within 30 days).
		if !r.isJWTExpiringSoon(existing.Data["key"], 30*24*time.Hour) {
			return nil
		}
		// Delete the expiring secret so we regenerate it below.
		log.FromContext(ctx).Info("regenerating expiring JWT", "secret", secretName, "namespace", namespace)
		if err := r.Delete(ctx, existing); err != nil {
			return fmt.Errorf("delete expiring %s: %w", secretName, err)
		}
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("check %s: %w", secretName, err)
	}

	jwtStr, err := resources.SignRS256JWT(claims, rsaKey, "key1")
	if err != nil {
		return fmt.Errorf("sign JWT for %s: %w", secretName, err)
	}

	userID := project.Labels[resources.LabelUserID]
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels:    resources.ComponentLabels(project.Name, userID, project.Spec.Plan, "gotrue"),
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"key": jwtStr,
		},
	}
	return r.Create(ctx, secret)
}

// isJWTExpiringSoon parses a JWT without verification and checks if exp is within the threshold.
func (r *ProjectReconciler) isJWTExpiringSoon(tokenData []byte, threshold time.Duration) bool {
	if len(tokenData) == 0 {
		return true
	}
	parser := jwtlib.NewParser(jwtlib.WithoutClaimsValidation())
	claims := &jwtlib.RegisteredClaims{}
	_, _, err := parser.ParseUnverified(string(tokenData), claims)
	if err != nil {
		return true // can't parse → treat as expired
	}
	if claims.ExpiresAt == nil {
		return false // no expiry → never expires
	}
	return time.Until(claims.ExpiresAt.Time) < threshold
}

func (r *ProjectReconciler) reconcileGoTrue(ctx context.Context, project *etalbaasv1alpha1.Project) error {
	namespace := "project-" + project.Name

	// Create gotrue-db-url Secret with search_path=auth appended to the CNPG db-app URI.
	// GoTrue needs search_path=auth so its unqualified DDL creates objects in the auth schema.
	if project.Spec.Stack.Postgres != nil && project.Spec.Stack.Postgres.Enabled {
		if err := r.ensureGoTrueDBURL(ctx, namespace); err != nil {
			return err
		}
	}

	deploy := resources.DesiredGoTrueDeployment(project, r.Config)
	if deploy != nil {
		if err := r.reconcileNamespacedResource(ctx, project, deploy); err != nil {
			return err
		}
	} else {
		r.deleteIfExists(ctx, &appsv1.Deployment{}, namespace, "gotrue")
	}

	svc := resources.DesiredGoTrueService(project)
	if svc != nil {
		return r.reconcileNamespacedResource(ctx, project, svc)
	}
	r.deleteIfExists(ctx, &corev1.Service{}, namespace, "gotrue")
	return nil
}

// ensureGoTrueDBURL reads the CNPG db-app Secret and creates a gotrue-db-url Secret
// with ?search_path=auth appended to the connection URI. GoTrue requires search_path=auth
// so that its migrations create enum types and objects in the auth schema, not public.
func (r *ProjectReconciler) ensureGoTrueDBURL(ctx context.Context, namespace string) error {
	dbAppSecret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: "db-app", Namespace: namespace}, dbAppSecret); err != nil {
		if apierrors.IsNotFound(err) {
			// db-app not yet created by CNPG; will be retried on next reconcile
			return nil
		}
		return fmt.Errorf("get db-app secret: %w", err)
	}

	uri := string(dbAppSecret.Data["uri"])
	if !strings.Contains(uri, "search_path") {
		if strings.Contains(uri, "?") {
			uri += "&search_path=auth"
		} else {
			uri += "?search_path=auth"
		}
	}

	gotrueDBSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gotrue-db-url",
			Namespace: namespace,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, gotrueDBSecret, func() error {
		gotrueDBSecret.Data = map[string][]byte{
			"uri": []byte(uri),
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("create gotrue-db-url secret: %w", err)
	}
	return nil
}

// generateRandomPassword creates a random alphanumeric password.
func generateRandomPassword(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b), nil
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
		spec, found, nestedErr := unstructured.NestedMap(desired.Object, "spec")
		if nestedErr != nil {
			return fmt.Errorf("read spec from desired: %w", nestedErr)
		}
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

	// Check PostgresMeta
	if pg := project.Spec.Stack.Postgres; pg != nil && pg.Enabled {
		metaStatus := &etalbaasv1alpha1.ComponentStatus{Status: "Provisioning"}
		deploy := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: "postgres-meta", Namespace: namespace}, deploy); err == nil {
			if deploy.Status.ReadyReplicas > 0 {
				metaStatus.Status = "Ready"
			} else {
				allReady = false
			}
		} else {
			allReady = false
		}
		components.PostgresMeta = metaStatus
	}

	// Check GoTrue
	if pg := project.Spec.Stack.Postgres; pg != nil && pg.Enabled {
		gotrueStatus := &etalbaasv1alpha1.ComponentStatus{Status: "Provisioning"}
		deploy := &appsv1.Deployment{}
		if err := r.Get(ctx, types.NamespacedName{Name: "gotrue", Namespace: namespace}, deploy); err == nil {
			if deploy.Status.ReadyReplicas > 0 {
				gotrueStatus.Status = "Ready"
			} else {
				allReady = false
			}
		} else {
			allReady = false
		}
		components.GoTrue = gotrueStatus
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
			AuthApi:            "https://" + subdomain + ".api." + r.Config.BaseDomain + "/auth",
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
// Conflict errors are transient and should be retried, not marked as Failed.
func (r *ProjectReconciler) setFailed(ctx context.Context, project *etalbaasv1alpha1.Project, reason string, err error) (ctrl.Result, error) {
	if apierrors.IsConflict(err) {
		return ctrl.Result{Requeue: true}, nil
	}
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
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&networkingv1.NetworkPolicy{}).
		Named("project").
		Complete(r)
}
