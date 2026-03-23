package history_test

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ensignwesley/svc/internal/history"
)

func openTestDB(t *testing.T) *history.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "history.db")
	db, err := history.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func strPtr(s string) *string { return &s }
func i64Ptr(i int64) *int64   { return &i }

func TestRecordUp(t *testing.T) {
	db := openTestDB(t)
	lat := int64(42)
	err := db.Record(history.CheckRow{
		ServiceID: "dead-drop",
		Host:      "localhost",
		CheckedAt: time.Now(),
		Status:    "up",
		LatencyMS: &lat,
	})
	if err != nil {
		t.Fatalf("Record up: %v", err)
	}
	// No open incidents for a service that just came up.
	open, err := db.QueryOpenIncidents()
	if err != nil {
		t.Fatalf("QueryOpenIncidents: %v", err)
	}
	if len(open) != 0 {
		t.Errorf("expected 0 open incidents, got %d", len(open))
	}
}

// TestIncidentOpensOnFirstFailure verifies that an incident row is created
// immediately on first failure — NOT deferred until recovery.
// This is the blind spot the design must avoid.
func TestIncidentOpensOnFirstFailure(t *testing.T) {
	db := openTestDB(t)

	err := db.Record(history.CheckRow{
		ServiceID: "dead-drop",
		Host:      "localhost",
		CheckedAt: time.Now(),
		Status:    "down",
		Error:     strPtr("connection refused"),
	})
	if err != nil {
		t.Fatalf("Record down: %v", err)
	}

	// Incident must be open NOW — not waiting for recovery.
	open, err := db.QueryOpenIncidents()
	if err != nil {
		t.Fatalf("QueryOpenIncidents: %v", err)
	}
	if len(open) != 1 {
		t.Fatalf("expected 1 open incident after first failure, got %d", len(open))
	}
	inc := open[0]
	if inc.RecoveredAt != nil {
		t.Error("expected recovered_at to be NULL for open incident")
	}
	if inc.ServiceID != "dead-drop" {
		t.Errorf("wrong service_id: %s", inc.ServiceID)
	}
	if inc.FirstError == nil || *inc.FirstError != "connection refused" {
		t.Errorf("wrong first_error: %v", inc.FirstError)
	}
}

func TestIncidentClosesOnRecovery(t *testing.T) {
	db := openTestDB(t)
	base := time.Now().Add(-10 * time.Minute)

	// Two failures.
	for i := 0; i < 2; i++ {
		db.Record(history.CheckRow{
			ServiceID: "dead-drop",
			CheckedAt: base.Add(time.Duration(i) * time.Minute),
			Status:    "down",
			Error:     strPtr("timeout"),
		})
	}

	// Recovery.
	err := db.Record(history.CheckRow{
		ServiceID: "dead-drop",
		CheckedAt: base.Add(5 * time.Minute),
		Status:    "up",
		LatencyMS: i64Ptr(38),
	})
	if err != nil {
		t.Fatalf("Record recovery: %v", err)
	}

	// No open incidents.
	open, err := db.QueryOpenIncidents()
	if err != nil {
		t.Fatalf("QueryOpenIncidents: %v", err)
	}
	if len(open) != 0 {
		t.Errorf("expected 0 open incidents after recovery, got %d", len(open))
	}

	// Closed incident exists with duration.
	incidents, err := db.QueryIncidents("dead-drop", 0, 10)
	if err != nil {
		t.Fatalf("QueryIncidents: %v", err)
	}
	if len(incidents) != 1 {
		t.Fatalf("expected 1 incident, got %d", len(incidents))
	}
	inc := incidents[0]
	if inc.IsOpen() {
		t.Error("expected closed incident")
	}
	if inc.CheckCount != 2 {
		t.Errorf("expected check_count 2, got %d", inc.CheckCount)
	}
	if inc.DurationSec == nil || *inc.DurationSec != 300 {
		t.Errorf("expected duration 300s, got %v", inc.DurationSec)
	}
}

func TestMultipleIncidents(t *testing.T) {
	db := openTestDB(t)
	base := time.Now().Add(-2 * time.Hour)

	// Incident 1: down then up.
	db.Record(history.CheckRow{ServiceID: "forth", CheckedAt: base, Status: "down", Error: strPtr("refused")})
	db.Record(history.CheckRow{ServiceID: "forth", CheckedAt: base.Add(5 * time.Minute), Status: "up", LatencyMS: i64Ptr(10)})

	// Incident 2: down, still open.
	db.Record(history.CheckRow{ServiceID: "forth", CheckedAt: base.Add(1 * time.Hour), Status: "down", Error: strPtr("timeout")})

	incidents, err := db.QueryIncidents("forth", 0, 10)
	if err != nil {
		t.Fatalf("QueryIncidents: %v", err)
	}
	if len(incidents) != 2 {
		t.Fatalf("expected 2 incidents, got %d", len(incidents))
	}

	// Most recent first — second incident (still open) should be first.
	if !incidents[0].IsOpen() {
		t.Error("expected most recent incident to be open")
	}
	if incidents[1].IsOpen() {
		t.Error("expected older incident to be closed")
	}
}

func TestUptimePct(t *testing.T) {
	db := openTestDB(t)
	base := time.Now().Add(-1 * time.Hour)

	// 9 up, 1 down = 90% uptime.
	for i := 0; i < 9; i++ {
		lat := int64(20)
		db.Record(history.CheckRow{
			ServiceID: "blog",
			CheckedAt: base.Add(time.Duration(i) * time.Minute),
			Status:    "up",
			LatencyMS: &lat,
		})
	}
	db.Record(history.CheckRow{
		ServiceID: "blog",
		CheckedAt: base.Add(9 * time.Minute),
		Status:    "down",
		Error:     strPtr("timeout"),
	})

	pct, incidents, err := db.UptimePct("blog", base.Add(-1*time.Minute).Unix())
	if err != nil {
		t.Fatalf("UptimePct: %v", err)
	}
	if pct < 89.9 || pct > 90.1 {
		t.Errorf("expected ~90%% uptime, got %.1f%%", pct)
	}
	if incidents != 1 {
		t.Errorf("expected 1 incident, got %d", incidents)
	}
}

func TestPrune(t *testing.T) {
	db := openTestDB(t)

	// Old check (2 days ago).
	old := time.Now().Add(-48 * time.Hour)
	lat := int64(10)
	db.Record(history.CheckRow{ServiceID: "blog", CheckedAt: old, Status: "up", LatencyMS: &lat})

	// Recent check.
	db.Record(history.CheckRow{ServiceID: "blog", CheckedAt: time.Now(), Status: "up", LatencyMS: &lat})

	// Prune anything older than 24h.
	deleted, err := db.Prune(int64(24 * time.Hour / time.Second))
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted row, got %d", deleted)
	}
}
