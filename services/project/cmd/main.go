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
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/project/v1/projectv1connect"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/secret/v1/secretv1connect"
	"github.com/kou-etal/etalbaas/services/project/internal"
	"github.com/kou-etal/etalbaas/services/project/internal/handler"
	"github.com/kou-etal/etalbaas/services/project/internal/service"
	"github.com/kou-etal/etalbaas/services/project/internal/store"
)

func main() {
	ctx := context.Background()

	cfg, err := internal.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	shutdown, err := observability.Init(ctx, observability.Config{
		ServiceName:    "project",
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

	k8sClient, err := k8s.NewClient()
	if err != nil {
		log.Fatal(err)
	}
	secMgr := k8s.NewSecretManager(k8sClient)

	q := store.New(pool)
	projectSvc := service.NewProjectService(q)
	secretSvc := service.NewSecretService(q, secMgr)

	projectHandler := handler.NewProjectHandler(projectSvc)
	secretHandler := handler.NewSecretHandler(secretSvc)

	interceptors := server.DefaultInterceptors(slog.Default())
	projPath, projHnd := projectv1connect.NewProjectServiceHandler(projectHandler, interceptors)
	secPath, secHnd := secretv1connect.NewSecretServiceHandler(secretHandler, interceptors)

	srv := server.New(
		server.Config{Port: cfg.Port, MetricsPort: cfg.MetricsPort},
		server.Handler{Pattern: projPath, Handler: projHnd},
		server.Handler{Pattern: secPath, Handler: secHnd},
	)
	srv.RegisterHealthChecker(func(ctx context.Context) error {
		return metadb.HealthCheck(ctx, pool)
	})
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
