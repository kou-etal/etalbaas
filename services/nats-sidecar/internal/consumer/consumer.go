package consumer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// Consumer pulls messages from a NATS JetStream consumer and forwards them
// as HTTP POST requests to the function container at localhost.
type Consumer struct {
	sub         *nats.Subscription
	functionURL string
	httpClient  *http.Client
	maxRetries  int
	logger      *slog.Logger
}

// New creates a Consumer bound to the given JetStream pull subscription.
func New(sub *nats.Subscription, functionURL string, maxRetries int, logger *slog.Logger) *Consumer {
	return &Consumer{
		sub:         sub,
		functionURL: functionURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		maxRetries: maxRetries,
		logger:     logger,
	}
}

// Run starts the pull-subscribe loop. It blocks until the context is cancelled.
func (c *Consumer) Run(ctx context.Context) error {
	c.logger.Info("consumer started", "function_url", c.functionURL)

	for {
		if ctx.Err() != nil {
			return nil
		}

		// Fetch one message at a time with a short timeout so we can check context.
		// Note: nats.MaxWait and nats.Context cannot be used together.
		fetchCtx, fetchCancel := context.WithTimeout(ctx, 5*time.Second)
		msgs, err := c.sub.Fetch(1, nats.Context(fetchCtx))
		fetchCancel()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			if err == nats.ErrTimeout || err == context.DeadlineExceeded {
				continue
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		for _, msg := range msgs {
			if err := c.processMessage(ctx, msg); err != nil {
				c.logger.Error("process message failed", "error", err)
			}
		}
	}
}

// processMessage forwards a single NATS message to the function via HTTP POST.
func (c *Consumer) processMessage(ctx context.Context, msg *nats.Msg) error {
	eventID := msg.Header.Get(nats.MsgIdHdr)

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.functionURL, bytes.NewReader(msg.Data))
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
		if eventID != "" {
			req.Header.Set("X-Event-Id", eventID)
		}
		req.Header.Set("X-Event-Type", eventTypeFromSubject(msg.Subject))

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("http post (attempt %d): %w", attempt+1, err)
			c.logger.Warn("function invocation failed", "attempt", attempt+1, "error", err)
			backoff(ctx, attempt)
			continue
		}
		// Limit response body read to 1MB to avoid resource exhaustion from misbehaving functions.
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			if err := msg.AckSync(); err != nil {
				return fmt.Errorf("ack message: %w", err)
			}
			c.logger.Debug("message processed", "event_id", eventID, "status", resp.StatusCode)
			return nil
		}

		lastErr = fmt.Errorf("function returned status %d (attempt %d)", resp.StatusCode, attempt+1)
		c.logger.Warn("function returned error",
			"status", resp.StatusCode,
			"attempt", attempt+1,
			"event_id", eventID,
		)
		backoff(ctx, attempt)
	}

	// All retries exhausted — Nak so NATS will redeliver later.
	if err := msg.Nak(); err != nil {
		c.logger.Error("failed to nak message", "error", err)
	}
	return fmt.Errorf("exhausted retries for event %s: %w", eventID, lastErr)
}

// eventTypeFromSubject derives the X-Event-Type header value from the NATS
// message subject.  Subject formats:
//   - events.database.<op>.project-<id>.<table>  → "DatabaseChange"
//   - events.storage.<op>.project-<id>.<bucket>  → "StorageEvent"
//
// Falls back to "DatabaseChange" for unrecognised subjects.
func eventTypeFromSubject(subject string) string {
	parts := strings.SplitN(subject, ".", 3)
	if len(parts) >= 2 && parts[1] == "storage" {
		return "StorageEvent"
	}
	return "DatabaseChange"
}

// backoff sleeps for an exponentially increasing duration, respecting context cancellation.
func backoff(ctx context.Context, attempt int) {
	d := time.Duration(1<<uint(attempt)) * time.Second
	if d > 10*time.Second {
		d = 10 * time.Second
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
	case <-t.C:
	}
}
