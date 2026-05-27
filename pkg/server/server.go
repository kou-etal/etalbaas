package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

type Handler struct {
	Pattern string
	Handler http.Handler
}

type Config struct {
	Port        int
	MetricsPort int
}

type Server struct {
	cfg            Config
	handlers       []Handler
	healthCheckers []HealthChecker
	metricsHandler http.Handler
}

func New(cfg Config, handlers ...Handler) *Server {
	return &Server{
		cfg:      cfg,
		handlers: handlers,
	}
}

func (s *Server) SetMetricsHandler(handler http.Handler) {
	s.metricsHandler = handler
}

func (s *Server) Run(ctx context.Context) error {
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", s.healthzHandler)

	for _, h := range s.handlers {
		mux.Handle(h.Pattern, h.Handler)
	}

	mainServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", s.cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	var metricsServer *http.Server
	if s.cfg.MetricsPort > 0 {
		metricsMux := http.NewServeMux()
		if s.metricsHandler != nil {
			metricsMux.Handle("/metrics", s.metricsHandler)
		}
		metricsServer = &http.Server{
			Addr:              fmt.Sprintf(":%d", s.cfg.MetricsPort),
			Handler:           metricsMux,
			ReadHeaderTimeout: 10 * time.Second,
		}
	}

	errCh := make(chan error, 2)

	go func() {
		slog.Info("starting main server", slog.Int("port", s.cfg.Port))
		if err := mainServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("main server: %w", err)
		}
	}()

	if metricsServer != nil {
		go func() {
			slog.Info("starting metrics server", slog.Int("port", s.cfg.MetricsPort))
			if err := metricsServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				errCh <- fmt.Errorf("metrics server: %w", err)
			}
		}()
	}

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		slog.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var shutdownErrs []error
	if err := mainServer.Shutdown(shutdownCtx); err != nil {
		shutdownErrs = append(shutdownErrs, fmt.Errorf("main server shutdown: %w", err))
	}
	if metricsServer != nil {
		if err := metricsServer.Shutdown(shutdownCtx); err != nil {
			shutdownErrs = append(shutdownErrs, fmt.Errorf("metrics server shutdown: %w", err))
		}
	}

	if len(shutdownErrs) > 0 {
		return errors.Join(shutdownErrs...)
	}

	slog.Info("server stopped gracefully")
	return nil
}

func (s *Server) Addr() string {
	return net.JoinHostPort("", fmt.Sprintf("%d", s.cfg.Port))
}
