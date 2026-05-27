package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/kou-etal/etalbaas/pkg/k8s"
	"github.com/kou-etal/etalbaas/pkg/metadb"
	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/server"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1/functionv1connect"
	"github.com/kou-etal/etalbaas/services/function/internal"
	"github.com/kou-etal/etalbaas/services/function/internal/gpuinvoke"
	"github.com/kou-etal/etalbaas/services/function/internal/handler"
	"github.com/kou-etal/etalbaas/services/function/internal/service"
	"github.com/kou-etal/etalbaas/services/function/internal/store"
)

func main() {
	ctx := context.Background()

	cfg, err := internal.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	obs, err := observability.Init(ctx, observability.Config{
		ServiceName:    "function",
		ServiceVersion: "0.1.0",
		OTELEndpoint:   cfg.OTELEndpoint,
		Environment:    cfg.Environment,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := obs.Shutdown(ctx); err != nil {
			slog.Error("observability shutdown failed", "error", err)
		}
	}()

	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	pool, err := metadb.NewPool(initCtx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	var dynClient dynamic.Interface
	if cfg.K8sEnabled {
		dc, err := k8s.NewDynamicClient()
		if err != nil {
			log.Fatal("create k8s dynamic client:", err)
		}
		dynClient = dc
		slog.Info("K8s CRD manager enabled")
	}

	var crdMgr k8s.FunctionCRDManager
	if dynClient != nil {
		crdMgr = k8s.NewFunctionCRDManager(dynClient)
	}

	q := store.New(pool)
	functionSvc := service.NewFunctionService(q, crdMgr)
	functionHandler := handler.NewFunctionHandler(functionSvc)

	interceptors := server.DefaultInterceptors(slog.Default())
	fnPath, fnHnd := functionv1connect.NewFunctionServiceHandler(functionHandler, interceptors)

	handlers := []server.Handler{
		{Pattern: fnPath, Handler: fnHnd},
	}

	// Register GPU invoke handler if GPU is enabled and K8s is available.
	if cfg.GPUEnabled && cfg.K8sEnabled {
		restCfg, err := rest.InClusterConfig()
		if err != nil {
			log.Fatal("GPU enabled but in-cluster config failed:", err)
		}
		k8sClient, err := kubernetes.NewForConfig(restCfg)
		if err != nil {
			log.Fatal("create k8s client for GPU invoke:", err)
		}

		gpuHandler := gpuinvoke.NewHandler(k8sClient, dynClient, gpuinvoke.Config{
			PlatformNamespace:   cfg.PlatformNamespace,
			DispatcherImage:     cfg.DispatcherImage,
			SandboxRuntimeClass: cfg.SandboxRuntimeClass,
			GPUAPIKeySecret:     cfg.GPUAPIKeySecret,
		})
		handlers = append(handlers, server.Handler{Pattern: "/gpu-invoke", Handler: gpuHandler})
		slog.Info("GPU invoke handler registered",
			"platformNamespace", cfg.PlatformNamespace,
			"dispatcherImage", cfg.DispatcherImage,
		)
	}

	srv := server.New(
		server.Config{Port: cfg.Port, MetricsPort: cfg.MetricsPort},
		handlers...,
	)
	srv.SetMetricsHandler(obs.MetricsHandler)
	srv.RegisterHealthChecker(func(ctx context.Context) error {
		return metadb.HealthCheck(ctx, pool)
	})
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
