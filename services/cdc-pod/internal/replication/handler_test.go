package replication

import (
	"log/slog"
	"testing"
	"time"

	"github.com/jackc/pglogrepl"

	"github.com/kou-etal/etalbaas/services/cdc-pod/internal/publisher"
)

func TestHandler_HandleInsertMessage(t *testing.T) {
	h := NewHandler(slog.Default())

	// Register a relation (table metadata).
	h.HandleRelationMessage(&pglogrepl.RelationMessage{
		RelationID:   10001,
		Namespace:    "public",
		RelationName: "orders",
		Columns: []*pglogrepl.RelationMessageColumn{
			{Name: "id", DataType: oidInt4},
			{Name: "status", DataType: 25}, // text
			{Name: "total", DataType: oidFloat8},
			{Name: "active", DataType: oidBool},
		},
	})

	// Set transaction context.
	txTime := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	h.HandleBeginMessage(&pglogrepl.BeginMessage{
		Xid:        42,
		CommitTime: txTime,
	})

	msg := &pglogrepl.InsertMessage{
		RelationID: 10001,
		Tuple: &pglogrepl.TupleData{
			Columns: []*pglogrepl.TupleDataColumn{
				{DataType: 't', Data: []byte("1")},
				{DataType: 't', Data: []byte("paid")},
				{DataType: 't', Data: []byte("100.5")},
				{DataType: 't', Data: []byte("t")},
			},
		},
	}

	event, err := h.HandleInsertMessage(msg, "abc123", pglogrepl.LSN(0x16B3798))
	if err != nil {
		t.Fatalf("HandleInsertMessage: %v", err)
	}

	assertEvent(t, event, "INSERT", "orders", "abc123")

	// Verify typed values.
	assertFieldInt(t, event.New, "id", 1)
	assertFieldString(t, event.New, "status", "paid")
	assertFieldFloat(t, event.New, "total", 100.5)
	assertFieldBool(t, event.New, "active", true)

	if event.Old != nil {
		t.Error("expected Old to be nil for INSERT")
	}

	// Verify event ID format: {project_id}-{lsn}-{txid}.
	expectedID := "abc123-0/16B3798-42"
	if event.EventID != expectedID {
		t.Errorf("event ID = %q, want %q", event.EventID, expectedID)
	}

	if !event.Timestamp.Equal(txTime) {
		t.Errorf("timestamp = %v, want %v", event.Timestamp, txTime)
	}
}

func TestHandler_HandleUpdateMessage(t *testing.T) {
	h := NewHandler(slog.Default())

	h.HandleRelationMessage(&pglogrepl.RelationMessage{
		RelationID:   10001,
		Namespace:    "public",
		RelationName: "orders",
		Columns: []*pglogrepl.RelationMessageColumn{
			{Name: "id", DataType: oidInt4},
			{Name: "status", DataType: 25},
		},
	})
	h.HandleBeginMessage(&pglogrepl.BeginMessage{Xid: 43, CommitTime: time.Now()})

	msg := &pglogrepl.UpdateMessage{
		RelationID: 10001,
		OldTuple: &pglogrepl.TupleData{
			Columns: []*pglogrepl.TupleDataColumn{
				{DataType: 't', Data: []byte("1")},
				{DataType: 't', Data: []byte("pending")},
			},
		},
		NewTuple: &pglogrepl.TupleData{
			Columns: []*pglogrepl.TupleDataColumn{
				{DataType: 't', Data: []byte("1")},
				{DataType: 't', Data: []byte("paid")},
			},
		},
	}

	event, err := h.HandleUpdateMessage(msg, "abc123", pglogrepl.LSN(0x1000))
	if err != nil {
		t.Fatalf("HandleUpdateMessage: %v", err)
	}

	assertEvent(t, event, "UPDATE", "orders", "abc123")
	assertFieldString(t, event.Old, "status", "pending")
	assertFieldString(t, event.New, "status", "paid")
}

func TestHandler_HandleDeleteMessage(t *testing.T) {
	h := NewHandler(slog.Default())

	h.HandleRelationMessage(&pglogrepl.RelationMessage{
		RelationID:   10001,
		Namespace:    "public",
		RelationName: "orders",
		Columns: []*pglogrepl.RelationMessageColumn{
			{Name: "id", DataType: oidInt4},
		},
	})
	h.HandleBeginMessage(&pglogrepl.BeginMessage{Xid: 44, CommitTime: time.Now()})

	msg := &pglogrepl.DeleteMessage{
		RelationID: 10001,
		OldTuple: &pglogrepl.TupleData{
			Columns: []*pglogrepl.TupleDataColumn{
				{DataType: 't', Data: []byte("1")},
			},
		},
	}

	event, err := h.HandleDeleteMessage(msg, "abc123", pglogrepl.LSN(0x2000))
	if err != nil {
		t.Fatalf("HandleDeleteMessage: %v", err)
	}

	assertEvent(t, event, "DELETE", "orders", "abc123")
	assertFieldInt(t, event.Old, "id", 1)
	if event.New != nil {
		t.Error("expected New to be nil for DELETE")
	}
}

func TestHandler_NullColumn(t *testing.T) {
	h := NewHandler(slog.Default())

	h.HandleRelationMessage(&pglogrepl.RelationMessage{
		RelationID:   10002,
		Namespace:    "public",
		RelationName: "users",
		Columns: []*pglogrepl.RelationMessageColumn{
			{Name: "id", DataType: oidInt4},
			{Name: "email", DataType: 25},
		},
	})
	h.HandleBeginMessage(&pglogrepl.BeginMessage{Xid: 50, CommitTime: time.Now()})

	msg := &pglogrepl.InsertMessage{
		RelationID: 10002,
		Tuple: &pglogrepl.TupleData{
			Columns: []*pglogrepl.TupleDataColumn{
				{DataType: 't', Data: []byte("1")},
				{DataType: 'n', Data: nil}, // NULL
			},
		},
	}

	event, err := h.HandleInsertMessage(msg, "proj1", pglogrepl.LSN(0x3000))
	if err != nil {
		t.Fatalf("HandleInsertMessage: %v", err)
	}

	if event.New["email"] != nil {
		t.Errorf("expected email to be nil, got %v", event.New["email"])
	}
}

func TestHandler_JSONBColumn(t *testing.T) {
	h := NewHandler(slog.Default())

	h.HandleRelationMessage(&pglogrepl.RelationMessage{
		RelationID:   10003,
		Namespace:    "public",
		RelationName: "metadata",
		Columns: []*pglogrepl.RelationMessageColumn{
			{Name: "data", DataType: oidJSONB},
		},
	})
	h.HandleBeginMessage(&pglogrepl.BeginMessage{Xid: 51, CommitTime: time.Now()})

	jsonData := `{"key":"value","num":42}`
	msg := &pglogrepl.InsertMessage{
		RelationID: 10003,
		Tuple: &pglogrepl.TupleData{
			Columns: []*pglogrepl.TupleDataColumn{
				{DataType: 't', Data: []byte(jsonData)},
			},
		},
	}

	event, err := h.HandleInsertMessage(msg, "proj1", pglogrepl.LSN(0x4000))
	if err != nil {
		t.Fatalf("HandleInsertMessage: %v", err)
	}

	dataVal, ok := event.New["data"].(map[string]any)
	if !ok {
		t.Fatalf("expected data to be map, got %T", event.New["data"])
	}
	if dataVal["key"] != "value" {
		t.Errorf("data.key = %v, want \"value\"", dataVal["key"])
	}
}

func TestHandler_UnknownRelation(t *testing.T) {
	h := NewHandler(slog.Default())
	h.HandleBeginMessage(&pglogrepl.BeginMessage{Xid: 60, CommitTime: time.Now()})

	msg := &pglogrepl.InsertMessage{
		RelationID: 99999,
		Tuple:      &pglogrepl.TupleData{},
	}

	_, err := h.HandleInsertMessage(msg, "proj1", pglogrepl.LSN(0))
	if err == nil {
		t.Error("expected error for unknown relation")
	}
}

func TestDecodeColumnValue(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		typeOID uint32
		want    any
	}{
		{"int4", []byte("42"), oidInt4, int64(42)},
		{"int8", []byte("-1000000"), oidInt8, int64(-1000000)},
		{"float8", []byte("3.14"), oidFloat8, 3.14},
		{"numeric", []byte("99.99"), oidNumeric, 99.99},
		{"bool_true", []byte("t"), oidBool, true},
		{"bool_false", []byte("f"), oidBool, false},
		{"text", []byte("hello"), 25, "hello"},
		{"uuid", []byte("550e8400-e29b-41d4-a716-446655440000"), 2950, "550e8400-e29b-41d4-a716-446655440000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeColumnValue(tt.data, tt.typeOID)
			if got != tt.want {
				t.Errorf("decodeColumnValue(%q, %d) = %v (%T), want %v (%T)",
					tt.data, tt.typeOID, got, got, tt.want, tt.want)
			}
		})
	}
}

// --- test helpers ---

func assertEvent(t *testing.T, event *publisher.CDCEvent, op, table, projectID string) {
	t.Helper()
	if event.Operation != op {
		t.Errorf("operation = %q, want %q", event.Operation, op)
	}
	if event.Table != table {
		t.Errorf("table = %q, want %q", event.Table, table)
	}
	if event.ProjectID != projectID {
		t.Errorf("project_id = %q, want %q", event.ProjectID, projectID)
	}
}

func assertFieldInt(t *testing.T, row map[string]any, key string, want int64) {
	t.Helper()
	v, ok := row[key]
	if !ok {
		t.Errorf("field %q not found", key)
		return
	}
	if v != want {
		t.Errorf("field %q = %v (%T), want %v", key, v, v, want)
	}
}

func assertFieldString(t *testing.T, row map[string]any, key, want string) {
	t.Helper()
	v, ok := row[key]
	if !ok {
		t.Errorf("field %q not found", key)
		return
	}
	if v != want {
		t.Errorf("field %q = %v, want %q", key, v, want)
	}
}

func assertFieldFloat(t *testing.T, row map[string]any, key string, want float64) {
	t.Helper()
	v, ok := row[key]
	if !ok {
		t.Errorf("field %q not found", key)
		return
	}
	if v != want {
		t.Errorf("field %q = %v, want %v", key, v, want)
	}
}

func assertFieldBool(t *testing.T, row map[string]any, key string, want bool) {
	t.Helper()
	v, ok := row[key]
	if !ok {
		t.Errorf("field %q not found", key)
		return
	}
	if v != want {
		t.Errorf("field %q = %v, want %v", key, v, want)
	}
}
