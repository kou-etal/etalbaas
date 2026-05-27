package natsadmin

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
)

// NATSAdmin manages NATS JetStream streams and consumers for the operator.
type NATSAdmin struct {
	js nats.JetStreamContext
}

// New creates a NATSAdmin from an existing JetStream context.
func New(js nats.JetStreamContext) *NATSAdmin {
	return &NATSAdmin{js: js}
}

// StreamName returns the JetStream stream name for a project's CDC events.
func StreamName(projectID string) string {
	return "CDC-project-" + projectID
}

// ConsumerName returns the JetStream consumer name for a function.
func ConsumerName(functionName string) string {
	return "func-" + functionName
}

// TriggerInfo holds the minimal info needed to build NATS filter subjects.
type TriggerInfo struct {
	Table      string
	Operations []string
}

// BuildFilterSubjects constructs NATS subject filters from trigger definitions.
// Subject format: {prefix}.{op}.project-{project_id}.{table}
func BuildFilterSubjects(subjectPrefix, projectID string, triggers []TriggerInfo) []string {
	var subjects []string
	for _, t := range triggers {
		table := sanitizeSubjectToken(t.Table)
		if len(t.Operations) > 0 {
			for _, op := range t.Operations {
				subjects = append(subjects, fmt.Sprintf("%s.%s.project-%s.%s",
					subjectPrefix, sanitizeSubjectToken(strings.ToLower(op)), projectID, table))
			}
		} else {
			// No operations filter → wildcard for all operations.
			subjects = append(subjects, fmt.Sprintf("%s.*.project-%s.%s",
				subjectPrefix, projectID, table))
		}
	}
	return subjects
}

// sanitizeSubjectToken removes NATS wildcard characters from a subject token.
func sanitizeSubjectToken(s string) string {
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "*", "_")
	s = strings.ReplaceAll(s, ">", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

// EnsureStream creates or updates the JetStream stream for a project.
// Stream captures all event-driven events: database CDC + storage object events.
func (a *NATSAdmin) EnsureStream(projectID string) error {
	name := StreamName(projectID)
	subjects := []string{
		fmt.Sprintf("events.database.*.project-%s.>", projectID),
		fmt.Sprintf("events.storage.*.project-%s.>", projectID),
	}

	cfg := &nats.StreamConfig{
		Name:       name,
		Subjects:   subjects,
		Retention:  nats.LimitsPolicy,
		MaxAge:     7 * 24 * time.Hour,
		MaxBytes:   5 * 1024 * 1024 * 1024, // 5 GiB
		Duplicates: 2 * time.Minute,
	}

	_, err := a.js.StreamInfo(name)
	if err == nil {
		_, err = a.js.UpdateStream(cfg)
		return err
	}
	if !errors.Is(err, nats.ErrStreamNotFound) {
		return fmt.Errorf("check stream %s: %w", name, err)
	}

	_, err = a.js.AddStream(cfg)
	return err
}

// DeleteStream removes the JetStream stream for a project. No-op if not found.
func (a *NATSAdmin) DeleteStream(projectID string) error {
	err := a.js.DeleteStream(StreamName(projectID))
	if errors.Is(err, nats.ErrStreamNotFound) {
		return nil
	}
	return err
}

// ConsumerConfig specifies a durable pull consumer to create/update.
type ConsumerConfig struct {
	StreamName     string
	ConsumerName   string
	FilterSubjects []string
}

// EnsureConsumer creates or updates a durable pull consumer for a function.
func (a *NATSAdmin) EnsureConsumer(cfg ConsumerConfig) error {
	if len(cfg.FilterSubjects) == 0 {
		return nil
	}

	consumerCfg := &nats.ConsumerConfig{
		Durable:       cfg.ConsumerName,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       60 * time.Second,
		MaxDeliver:    3,
		BackOff:       []time.Duration{5 * time.Second, 10 * time.Second, 20 * time.Second},
		DeliverPolicy: nats.DeliverNewPolicy,
	}

	// Use FilterSubject (singular) when possible for broader NATS server compat.
	if len(cfg.FilterSubjects) == 1 {
		consumerCfg.FilterSubject = cfg.FilterSubjects[0]
	} else {
		consumerCfg.FilterSubjects = cfg.FilterSubjects
	}

	_, err := a.js.ConsumerInfo(cfg.StreamName, cfg.ConsumerName)
	if err == nil {
		_, err = a.js.UpdateConsumer(cfg.StreamName, consumerCfg)
		return err
	}
	if !errors.Is(err, nats.ErrConsumerNotFound) {
		return fmt.Errorf("check consumer %s/%s: %w", cfg.StreamName, cfg.ConsumerName, err)
	}

	_, err = a.js.AddConsumer(cfg.StreamName, consumerCfg)
	return err
}

// DeleteConsumer removes a consumer. No-op if not found.
func (a *NATSAdmin) DeleteConsumer(streamName, consumerName string) error {
	err := a.js.DeleteConsumer(streamName, consumerName)
	if errors.Is(err, nats.ErrConsumerNotFound) || errors.Is(err, nats.ErrStreamNotFound) {
		return nil
	}
	return err
}
