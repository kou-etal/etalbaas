package publisher

import "time"

// CDCEvent represents a single database change event published to NATS.
type CDCEvent struct {
	EventID   string         `json:"event_id"`
	ProjectID string         `json:"project_id"`
	Table     string         `json:"table"`
	Operation string         `json:"operation"`
	Timestamp time.Time      `json:"timestamp"`
	Old       map[string]any `json:"old"`
	New       map[string]any `json:"new"`
}
