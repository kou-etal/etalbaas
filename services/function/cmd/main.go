package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/kou-etal/etalbaas/pkg/k8s"
	"github.com/kou-etal/etalbaas/pkg/metadb"
	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/server"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/function/v1/functionv1connect"
	"github.com/kou-etal/etalbaas/services/function/internal"
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

	shutdown, err := observability.Init(ctx, observability.Config{
		ServiceName:    "function",
		ServiceVersion: "0.1.0",
		OTELEndpoint:   cfg.OTELEndpoint,
		Environment:    cfg.Environment,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := shutdown(ctx); err != nil {
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

	var crdMgr k8s.FunctionCRDManager
	if cfg.K8sEnabled {
		dynClient, err := k8s.NewDynamicClient()
		if err != nil {
			log.Fatal("create k8s dynamic client:", err)
		}
		crdMgr = k8s.NewFunctionCRDManager(dynClient)
		slog.Info("K8s CRD manager enabled")
	}

	q := store.New(pool)
	functionSvc := service.NewFunctionService(q, crdMgr)
	functionHandler := handler.NewFunctionHandler(functionSvc)

	interceptors := server.DefaultInterceptors(slog.Default())
	fnPath, fnHnd := functionv1connect.NewFunctionServiceHandler(functionHandler, interceptors)

	srv := server.New(
		server.Config{Port: cfg.Port, MetricsPort: cfg.MetricsPort},
		server.Handler{Pattern: fnPath, Handler: fnHnd},
	)
	srv.RegisterHealthChecker(func(ctx context.Context) error {
		return metadb.HealthCheck(ctx, pool)
	})
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
