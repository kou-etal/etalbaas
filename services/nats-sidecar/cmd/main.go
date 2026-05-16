package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/kou-etal/etalbaas/services/nats-sidecar/internal/config"
	"github.com/kou-etal/etalbaas/services/nats-sidecar/internal/consumer"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer cancel()

	logger := slog.Default()

	// 1. Load configuration.
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("config loaded",
		"stream", cfg.Stream,
		"consumer", cfg.Consumer,
		"function_url", cfg.FunctionURL,
	)

	// 2. Start health check server.
	healthSrv := startHealthServer(cfg.HealthPort)
	defer func() {
		shutdownCtx, c := context.WithTimeout(context.Background(), 5*time.Second)
		defer c()
		_ = healthSrv.Shutdown(shutdownCtx)
	}()

	// 3. Connect to NATS JetStream.
	nc, err := nats.Connect(cfg.NATSURL,
		nats.Name("nats-sidecar-"+cfg.Consumer),
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

	// 4. Bind to the durable pull consumer created by the Operator.
	sub, err := js.PullSubscribe("", cfg.Consumer,
		nats.Bind(cfg.Stream, cfg.Consumer),
	)
	if err != nil {
		log.Fatal(err)
	}
	logger.Info("subscribed to consumer", "stream", cfg.Stream, "consumer", cfg.Consumer)

	// 5. Run consumer loop.
	c := consumer.New(sub, cfg.FunctionURL, cfg.MaxRetries, logger)
	if err := c.Run(ctx); err != nil {
		log.Fatal(err)
	}
	logger.Info("nats-sidecar shutdown complete")
}

func startHealthServer(port int) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "error", err)
		}
	}()
	return srv
}
