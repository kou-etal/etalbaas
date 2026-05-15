package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
)

type HealthChecker func(ctx context.Context) error

func (s *Server) RegisterHealthChecker(checker HealthChecker) {
	s.healthCheckers = append(s.healthCheckers, checker)
}

func (s *Server) healthzHandler(w http.ResponseWriter, r *http.Request) {
	for _, checker := range s.healthCheckers {
		if err := checker(r.Context()); err != nil {
			slog.ErrorContext(r.Context(), "health check failed", slog.Any("error", err))
			http.Error(w, "unhealthy", http.StatusServiceUnavailable)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}
