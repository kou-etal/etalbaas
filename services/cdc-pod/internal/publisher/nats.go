package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
)

// NATSPublisher publishes CDCEvents to NATS JetStream.
type NATSPublisher struct {
	js            nats.JetStreamContext
	subjectPrefix string
	projectID     string
}

// NewNATSPublisher creates a publisher that sends events to JetStream.
func NewNATSPublisher(js nats.JetStreamContext, subjectPrefix, projectID string) *NATSPublisher {
	return &NATSPublisher{
		js:            js,
		subjectPrefix: subjectPrefix,
		projectID:     projectID,
	}
}

// Publish sends a CDCEvent to NATS JetStream with exactly-once deduplication.
func (p *NATSPublisher) Publish(ctx context.Context, event *CDCEvent) error {
	subject := p.BuildSubject(event.Operation, event.Table)

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	msg := &nats.Msg{
		Subject: subject,
		Data:    data,
		Header:  nats.Header{},
	}
	// Nats-Msg-Id for exactly-once deduplication within the JetStream dedup window.
	msg.Header.Set(nats.MsgIdHdr, event.EventID)

	if _, err = p.js.PublishMsg(msg, nats.Context(ctx)); err != nil {
		return fmt.Errorf("publish to %s: %w", subject, err)
	}
	return nil
}

// BuildSubject constructs the NATS subject for an event.
// Format: {prefix}.{operation}.project-{project_id}.{table}
// Example: events.database.insert.project-abc123.orders
func (p *NATSPublisher) BuildSubject(operation, table string) string {
	op := OperationToSubject(operation)
	return fmt.Sprintf("%s.%s.project-%s.%s", p.subjectPrefix, op, p.projectID, table)
}

// OperationToSubject converts a WAL operation name to a lowercase NATS subject token.
func OperationToSubject(op string) string {
	switch strings.ToUpper(op) {
	case "INSERT":
		return "insert"
	case "UPDATE":
		return "update"
	case "DELETE":
		return "delete"
	default:
		return "unknown"
	}
}
