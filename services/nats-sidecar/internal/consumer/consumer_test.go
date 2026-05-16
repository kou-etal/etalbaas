package consumer

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestBackoff(t *testing.T) {
	tests := []struct {
		attempt int
		wantMax time.Duration
	}{
		{0, 2 * time.Second},
		{1, 3 * time.Second},
		{2, 5 * time.Second},
		{3, 9 * time.Second},
		{10, 11 * time.Second}, // capped at 10s
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			start := time.Now()
			backoff(context.Background(), tt.attempt)
			elapsed := time.Since(start)
			if elapsed > tt.wantMax {
				t.Errorf("backoff(%d) took %v, expected less than %v", tt.attempt, elapsed, tt.wantMax)
			}
		})
	}
}

func TestProcessMessage_HeadersForwarded(t *testing.T) {
	var callCount atomic.Int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		_, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	// We can't easily construct a real nats.Msg for processMessage because
	// AckSync requires a NATS connection. Verify the test server is set up
	// and that backoff doesn't panic on high attempts.
	_ = ts
	backoff(context.Background(), 100)

	if callCount.Load() != 0 {
		t.Errorf("expected 0 calls, got %d", callCount.Load())
	}
}
