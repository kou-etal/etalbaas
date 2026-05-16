package main

import (
	"context"
	"log"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/config"
	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/health"
	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/publisher"
	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/replication"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger := slog.Default()

	// 1. Load configuration from environment variables.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("config loaded",
		"project_id", cfg.ProjectID,
		"slot_name", cfg.SlotName,
		"nats_url", cfg.NATSURL,
	)

	// 2. Start health check server.
	healthSrv := health.StartServer(cfg.HealthPort)
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = healthSrv.Shutdown(shutdownCtx)
	}()

	// 3. Connect to NATS JetStream.
	nc, err := nats.Connect(cfg.NATSURL,
		nats.Name("cdc-pod-"+cfg.ProjectID),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	js, err := nc.JetStream()
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("connected to NATS JetStream")

	// 4. Build publisher, handler, and replicator.
	pub := publisher.NewNATSPublisher(js, cfg.SubjectPrefix, cfg.ProjectID)
	hnd := replication.NewHandler(logger)
	repl := replication.NewReplicator(cfg, hnd, pub, logger)

	// 5. Run the replication loop (blocks until context cancellation or error).
	if err := repl.Run(ctx); err != nil {
		log.Fatal(err)
	}
	logger.Info("cdc-pod shutdown complete")
}
