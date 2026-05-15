package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/kou-etal/etalbaas/pkg/metadb"
	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/server"
	tenantv1connect "github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/tenant/v1/tenantv1connect"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/handler"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/service"
	"github.com/kou-etal/etalbaas/services/tenant-user/internal/store"
)

func main() {
	ctx := context.Background()

	cfg, err := internal.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	shutdown, err := observability.Init(ctx, observability.Config{
		ServiceName:    "tenant-user",
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

	svc := service.New(store.New(pool))
	h := handler.New(svc)
	interceptors := server.DefaultInterceptors(slog.Default())
	path, hnd := tenantv1connect.NewTenantServiceHandler(h, interceptors)

	srv := server.New(
		server.Config{Port: cfg.Port, MetricsPort: cfg.MetricsPort},
		server.Handler{Pattern: path, Handler: hnd},
	)
	srv.RegisterHealthChecker(func(ctx context.Context) error {
		return metadb.HealthCheck(ctx, pool)
	})
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
