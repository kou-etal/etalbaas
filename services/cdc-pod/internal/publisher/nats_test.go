package publisher

import "testing"

func TestBuildSubject(t *testing.T) {
	p := NewNATSPublisher(nil, "events.database", "abc123")

	tests := []struct {
		operation string
		table     string
		want      string
	}{
		{"INSERT", "orders", "events.database.insert.project-abc123.orders"},
		{"UPDATE", "users", "events.database.update.project-abc123.users"},
		{"DELETE", "items", "events.database.delete.project-abc123.items"},
		{"UNKNOWN", "test", "events.database.unknown.project-abc123.test"},
	}

	for _, tt := range tests {
		t.Run(tt.operation+"_"+tt.table, func(t *testing.T) {
			got := p.BuildSubject(tt.operation, tt.table)
			if got != tt.want {
				t.Errorf("BuildSubject(%q, %q) = %q, want %q", tt.operation, tt.table, got, tt.want)
			}
		})
	}
}

func TestOperationToSubject(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"INSERT", "insert"},
		{"UPDATE", "update"},
		{"DELETE", "delete"},
		{"insert", "insert"},
		{"TRUNCATE", "unknown"},
		{"", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := OperationToSubject(tt.input)
			if got != tt.want {
				t.Errorf("OperationToSubject(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
