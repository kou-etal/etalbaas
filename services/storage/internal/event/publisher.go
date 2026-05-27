package event

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/nats-io/nats.go"
)

// StorageEvent represents a storage object event published to NATS JetStream.
type StorageEvent struct {
	EventID   string    `json:"event_id"`
	ProjectID string    `json:"project_id"`
	Bucket    string    `json:"bucket"`
	ObjectKey string    `json:"object_key"`
	Operation string    `json:"operation"` // "created" | "deleted"
	Size      int64     `json:"size"`
	MimeType  string    `json:"mime_type"`
	Timestamp time.Time `json:"timestamp"`
}

// Publisher publishes storage events to a message broker.
type Publisher interface {
	PublishObjectCreated(ctx context.Context, ev StorageEvent) error
	PublishObjectDeleted(ctx context.Context, ev StorageEvent) error
}

// NATSPublisher publishes StorageEvents to NATS JetStream.
type NATSPublisher struct {
	js            nats.JetStreamContext
	subjectPrefix string
}

// NewNATSPublisher creates a publisher that sends storage events to JetStream.
func NewNATSPublisher(js nats.JetStreamContext, subjectPrefix string) *NATSPublisher {
	return &NATSPublisher{
		js:            js,
		subjectPrefix: subjectPrefix,
	}
}

// PublishObjectCreated publishes an object-created event.
func (p *NATSPublisher) PublishObjectCreated(ctx context.Context, ev StorageEvent) error {
	ev.Operation = "created"
	return p.publish(ctx, ev)
}

// PublishObjectDeleted publishes an object-deleted event.
func (p *NATSPublisher) PublishObjectDeleted(ctx context.Context, ev StorageEvent) error {
	ev.Operation = "deleted"
	return p.publish(ctx, ev)
}

func (p *NATSPublisher) publish(ctx context.Context, ev StorageEvent) error {
	subject := p.buildSubject(ev.Operation, ev.ProjectID, ev.Bucket)

	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("marshal storage event: %w", err)
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}
	// Nats-Msg-Id for exactly-once deduplication within the JetStream dedup window.
	msg.Header.Set(nats.MsgIdHdr, ev.EventID)

	if _, err = p.js.PublishMsg(msg, nats.Context(ctx)); err != nil {
		slog.Error("failed to publish storage event",
			"subject", subject,
			"event_id", ev.EventID,
			"project_id", ev.ProjectID,
			"bucket", ev.Bucket,
			"operation", ev.Operation,
			"error", err,
		)
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}

// buildSubject constructs the NATS subject for a storage event.
// Format: {prefix}.{operation}.project-{projectID}.{bucket}
// Example: events.storage.created.project-abc123.uploads
func (p *NATSPublisher) buildSubject(operation, projectID, bucket string) string {
	return fmt.Sprintf("%s.%s.project-%s.%s", p.subjectPrefix, operation, projectID, bucket)
}
