package health

import (
	"fmt"
	"net/http"
)

// StartServer starts a lightweight HTTP health check server on the given port.
// Returns the server so the caller can shut it down gracefully.
func StartServer(port int) *http.Server {
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
			// Health server failure is non-fatal; the main loop will continue.
			fmt.Printf("health server error: %v\n", err)
		}
	}()
	return srv
}
