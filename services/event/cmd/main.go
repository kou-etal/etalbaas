package main

import (
	"context"
	"log"
	"log/slog"
	"time"

	"github.com/kou-etal/etalbaas/pkg/metadb"
	"github.com/kou-etal/etalbaas/pkg/observability"
	"github.com/kou-etal/etalbaas/pkg/server"
	"github.com/kou-etal/etalbaas/proto/gen/go/etalbaas/event/v1/eventv1connect"
	"github.com/kou-etal/etalbaas/services/event/internal"
	"github.com/kou-etal/etalbaas/services/event/internal/handler"
	"github.com/kou-etal/etalbaas/services/event/internal/service"
	"github.com/kou-etal/etalbaas/services/event/internal/store"
)

func main() {
	ctx := context.Background()

	cfg, err := internal.NewConfig()
	if err != nil {
		log.Fatal(err)
	}

	obs, err := observability.Init(ctx, observability.Config{
		ServiceName:    "event",
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

	q := store.New(pool)
	eventSvc := service.NewEventService(q)
	eventHandler := handler.NewEventHandler(eventSvc)

	interceptors := server.DefaultInterceptors(slog.Default())
	evtPath, evtHnd := eventv1connect.NewEventServiceHandler(eventHandler, interceptors)

	srv := server.New(
		server.Config{Port: cfg.Port, MetricsPort: cfg.MetricsPort},
		server.Handler{Pattern: evtPath, Handler: evtHnd},
	)
	srv.SetMetricsHandler(obs.MetricsHandler)
	srv.RegisterHealthChecker(func(ctx context.Context) error {
		return metadb.HealthCheck(ctx, pool)
	})
	if err := srv.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
