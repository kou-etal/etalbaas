package main

import (
	"flag"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	etalbaasv1alpha1 "github.com/kou-etal/etalbaas/operator/api/v1alpha1"
	"github.com/kou-etal/etalbaas/operator/internal/config"
	"github.com/kou-etal/etalbaas/operator/internal/controller"
	"github.com/kou-etal/etalbaas/operator/internal/natsadmin"
	gpuprovider "github.com/kou-etal/etalbaas/operator/internal/provider/gpu"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(etalbaasv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. Use :8443 for HTTPS or :8080 for HTTP.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")

	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "etalbaas-operator.etalbaas.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	operatorConfig := config.DefaultConfig()

	// Override defaults from environment variables (set by Helm chart).
	if v := os.Getenv("PLATFORM_NAMESPACE"); v != "" {
		operatorConfig.PlatformNamespace = v
	}
	if v := os.Getenv("BASE_DOMAIN"); v != "" {
		operatorConfig.BaseDomain = v
	}
	if v := os.Getenv("REGISTRY_ENDPOINT"); v != "" {
		operatorConfig.RegistryEndpoint = v
	}
	if v := os.Getenv("REGISTRY_INSECURE"); v == "true" {
		operatorConfig.RegistryInsecure = true
	}
	if v := os.Getenv("NATS_ENDPOINT"); v != "" {
		operatorConfig.NATSEndpoint = v
	}
	if v := os.Getenv("NATS_MONITORING_ENDPOINT"); v != "" {
		operatorConfig.NATSMonitoringEndpoint = v
	}
	if v := os.Getenv("GATEWAY_NAME"); v != "" {
		operatorConfig.GatewayName = v
	}
	if v := os.Getenv("GATEWAY_NAMESPACE"); v != "" {
		operatorConfig.GatewayNamespace = v
	}
	if v := os.Getenv("CDC_IMAGE"); v != "" {
		operatorConfig.CDCImage = v
	}
	if v := os.Getenv("NATS_SIDECAR_IMAGE"); v != "" {
		operatorConfig.NATSSidecarImage = v
	}
	if v := os.Getenv("POSTGREST_IMAGE"); v != "" {
		operatorConfig.PostgRESTImage = v
	}
	if v := os.Getenv("REDIS_IMAGE"); v != "" {
		operatorConfig.RedisImage = v
	}
	if v := os.Getenv("KANIKO_IMAGE"); v != "" {
		operatorConfig.KanikoImage = v
	}
	if v := os.Getenv("DISPATCHER_IMAGE"); v != "" {
		operatorConfig.DispatcherImage = v
	}
	if v := os.Getenv("POSTGRES_META_IMAGE"); v != "" {
		operatorConfig.PostgresMetaImage = v
	}
	if v := os.Getenv("GOTRUE_IMAGE"); v != "" {
		operatorConfig.GoTrueImage = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		operatorConfig.JWTSecret = v
	}
	if v, ok := os.LookupEnv("SANDBOX_RUNTIME_CLASS"); ok {
		operatorConfig.SandboxRuntimeClass = v
	}
	if v := os.Getenv("BUILD_TIMEOUT_MINUTES"); v != "" {
		if minutes, err := strconv.Atoi(v); err == nil {
			operatorConfig.BuildTimeout = time.Duration(minutes) * time.Minute
		}
	}
	if v := os.Getenv("MAX_TOTAL_PROJECTS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			operatorConfig.MaxTotalProjects = n
		}
	}
	if v := os.Getenv("MAX_CONCURRENT_BUILDS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			operatorConfig.MaxConcurrentBuilds = n
		}
	}
	if v := os.Getenv("FREE_PLAN_CPU"); v != "" {
		operatorConfig.FreePlanQuota.CPU = v
	}
	if v := os.Getenv("FREE_PLAN_MEMORY"); v != "" {
		operatorConfig.FreePlanQuota.Memory = v
	}
	if v := os.Getenv("FREE_PLAN_PODS"); v != "" {
		if pods, err := strconv.ParseInt(v, 10, 32); err == nil {
			operatorConfig.FreePlanQuota.Pods = int32(pods)
		}
	}

	// GPU configuration from Helm values (operator deployment.yaml L89-104).
	if v := os.Getenv("GPU_ENABLED"); v == "true" {
		operatorConfig.GPU.Enabled = true
	}
	if v := os.Getenv("GPU_DEFAULT_PROVIDER"); v != "" {
		operatorConfig.GPU.DefaultProvider = v
	}
	if v := os.Getenv("GPU_RUNPOD_API_KEY_SECRET"); v != "" {
		if operatorConfig.GPU.Providers.RunPod == nil {
			operatorConfig.GPU.Providers.RunPod = &gpuprovider.RunPodConfig{}
		}
		operatorConfig.GPU.Providers.RunPod.Enabled = true
		operatorConfig.GPU.Providers.RunPod.APIKeySecret = v
	}
	if v := os.Getenv("GPU_SELF_MANAGED_RUNTIME_CLASS"); v != "" {
		if operatorConfig.GPU.Providers.SelfManaged == nil {
			operatorConfig.GPU.Providers.SelfManaged = &gpuprovider.SelfManagedConfig{}
		}
		operatorConfig.GPU.Providers.SelfManaged.Enabled = true
		operatorConfig.GPU.Providers.SelfManaged.RuntimeClass = v
	}
	if v := os.Getenv("GPU_SELF_MANAGED_RESOURCE_LIMIT"); v != "" {
		if operatorConfig.GPU.Providers.SelfManaged == nil {
			operatorConfig.GPU.Providers.SelfManaged = &gpuprovider.SelfManagedConfig{}
		}
		operatorConfig.GPU.Providers.SelfManaged.ResourceLimit = v
	}

	// Initialize NATS connection for stream/consumer management.
	var natsAdmin *natsadmin.NATSAdmin
	if operatorConfig.NATSEndpoint != "" {
		natsURL := "nats://" + operatorConfig.NATSEndpoint
		nc, err := nats.Connect(natsURL,
			nats.Name("etalbaas-operator"),
			nats.RetryOnFailedConnect(true),
			nats.MaxReconnects(-1),
		)
		if err != nil {
			setupLog.Error(err, "unable to connect to NATS", "url", natsURL)
			os.Exit(1)
		}
		defer nc.Close()

		js, err := nc.JetStream()
		if err != nil {
			setupLog.Error(err, "unable to create JetStream context")
			os.Exit(1)
		}
		natsAdmin = natsadmin.New(js)
		setupLog.Info("connected to NATS JetStream", "endpoint", operatorConfig.NATSEndpoint)
	}

	if err := (&controller.ProjectReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Config:    operatorConfig,
		NATSAdmin: natsAdmin,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Project")
		os.Exit(1)
	}

	// Initialize GPU Factory if GPU features are enabled.
	var gpuFactory *gpuprovider.Factory
	if operatorConfig.GPU.Enabled {
		k8sClient, err := kubernetes.NewForConfig(ctrl.GetConfigOrDie())
		if err != nil {
			setupLog.Error(err, "unable to create kubernetes client for GPU factory")
			os.Exit(1)
		}
		gpuFactory = gpuprovider.NewFactory(
			operatorConfig.GPU,
			&http.Client{Timeout: 30 * time.Second},
			k8sClient,
			operatorConfig.PlatformNamespace,
		)
		setupLog.Info("GPU provider factory initialized", "defaultProvider", operatorConfig.GPU.DefaultProvider)
	}

	if err := (&controller.FunctionReconciler{
		Client:     mgr.GetClient(),
		Scheme:     mgr.GetScheme(),
		Config:     operatorConfig,
		NATSAdmin:  natsAdmin,
		GPUFactory: gpuFactory,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Function")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
