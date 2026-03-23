package history

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the history SQLite database.
type DB struct {
	db *sql.DB
}

// DefaultPath returns the default history database path.
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".svc-history.db"
	}
	return filepath.Join(home, ".local", "share", "svc", "history.db")
}

// Open opens (or creates) the history database at path.
func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("creating history dir: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening history db: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite: one writer at a time
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("creating schema: %w", err)
	}
	return &DB{db: db}, nil
}

// Close closes the database.
func (h *DB) Close() error {
	return h.db.Close()
}

// CheckRow is a single poll result to be recorded.
type CheckRow struct {
	ServiceID string
	Host      string
	CheckedAt time.Time
	Status    string // "up" | "down" | "timeout" | "error"
	LatencyMS *int64 // nil when down
	Error     *string
}

// Record inserts a check result and updates the incident table.
func (h *DB) Record(c CheckRow) error {
	host := c.Host
	if host == "" {
		host = "localhost"
	}
	ts := c.CheckedAt.Unix()

	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. Append to checks (immutable).
	_, err = tx.Exec(
		`INSERT INTO checks (service_id, host, checked_at, status, latency_ms, error)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		c.ServiceID, host, ts, c.Status, c.LatencyMS, c.Error,
	)
	if err != nil {
		return fmt.Errorf("inserting check: %w", err)
	}

	// 2. Update incidents table.
	if c.Status != "up" {
		// Failure: open a new incident or increment check_count on open one.
		var incidentID int64
		err = tx.QueryRow(
			`SELECT id FROM incidents
			 WHERE service_id = ? AND host = ? AND recovered_at IS NULL
			 LIMIT 1`,
			c.ServiceID, host,
		).Scan(&incidentID)

		if err == sql.ErrNoRows {
			// First failure in a new incident — open it now.
			_, err = tx.Exec(
				`INSERT INTO incidents (service_id, host, started_at, check_count, first_error)
				 VALUES (?, ?, ?, 1, ?)`,
				c.ServiceID, host, ts, c.Error,
			)
			if err != nil {
				return fmt.Errorf("opening incident: %w", err)
			}
		} else if err == nil {
			// Existing open incident — increment check count.
			_, err = tx.Exec(
				`UPDATE incidents SET check_count = check_count + 1 WHERE id = ?`,
				incidentID,
			)
			if err != nil {
				return fmt.Errorf("updating incident: %w", err)
			}
		} else {
			return fmt.Errorf("querying open incident: %w", err)
		}
	} else {
		// Recovery: close any open incident.
		var incidentID int64
		var startedAt int64
		err = tx.QueryRow(
			`SELECT id, started_at FROM incidents
			 WHERE service_id = ? AND host = ? AND recovered_at IS NULL
			 LIMIT 1`,
			c.ServiceID, host,
		).Scan(&incidentID, &startedAt)

		if err == nil {
			// Close it.
			duration := ts - startedAt
			_, err = tx.Exec(
				`UPDATE incidents
				 SET recovered_at = ?, duration_sec = ?
				 WHERE id = ?`,
				ts, duration, incidentID,
			)
			if err != nil {
				return fmt.Errorf("closing incident: %w", err)
			}
		}
		// sql.ErrNoRows here is fine — no open incident to close.
	}

	return tx.Commit()
}

// Incident represents a completed or open down event.
type Incident struct {
	ID          int64
	ServiceID   string
	Host        string
	StartedAt   time.Time
	RecoveredAt *time.Time // nil = still open
	DurationSec *int64
	CheckCount  int
	FirstError  *string
}

// IsOpen returns true if the incident has not yet recovered.
func (i Incident) IsOpen() bool {
	return i.RecoveredAt == nil
}

// QueryIncidents returns incidents for a service, most recent first.
// sinceTS=0 means no time filter. limit=0 means use default (20).
func (h *DB) QueryIncidents(serviceID string, sinceTS int64, limit int) ([]Incident, error) {
	if limit <= 0 {
		limit = 20
	}
	args := []any{serviceID}
	where := "WHERE service_id = ?"
	if sinceTS > 0 {
		where += " AND started_at >= ?"
		args = append(args, sinceTS)
	}
	args = append(args, limit)

	rows, err := h.db.Query(
		`SELECT id, service_id, host, started_at, recovered_at, duration_sec, check_count, first_error
		 FROM incidents `+where+`
		 ORDER BY started_at DESC LIMIT ?`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIncidents(rows)
}

// QueryOpenIncidents returns all currently-open incidents (service still down).
func (h *DB) QueryOpenIncidents() ([]Incident, error) {
	rows, err := h.db.Query(
		`SELECT id, service_id, host, started_at, recovered_at, duration_sec, check_count, first_error
		 FROM incidents WHERE recovered_at IS NULL
		 ORDER BY started_at ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIncidents(rows)
}

// UptimePct returns the uptime percentage for a service over a time window.
func (h *DB) UptimePct(serviceID string, sinceTS int64) (float64, int, error) {
	var total, down int
	err := h.db.QueryRow(
		`SELECT COUNT(*), SUM(CASE WHEN status != 'up' THEN 1 ELSE 0 END)
		 FROM checks WHERE service_id = ? AND checked_at >= ?`,
		serviceID, sinceTS,
	).Scan(&total, &down)
	if err != nil || total == 0 {
		return 100.0, 0, err
	}

	var incidents int
	h.db.QueryRow(
		`SELECT COUNT(*) FROM incidents WHERE service_id = ? AND started_at >= ?`,
		serviceID, sinceTS,
	).Scan(&incidents)

	return float64(total-down) / float64(total) * 100, incidents, nil
}

// Prune deletes checks older than keepSecs. Incidents are never pruned.
func (h *DB) Prune(keepSecs int64) (int64, error) {
	cutoff := time.Now().Unix() - keepSecs
	res, err := h.db.Exec(`DELETE FROM checks WHERE checked_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanIncidents(rows *sql.Rows) ([]Incident, error) {
	var incidents []Incident
	for rows.Next() {
		var inc Incident
		var startedTS int64
		var recoveredTS *int64
		err := rows.Scan(
			&inc.ID, &inc.ServiceID, &inc.Host,
			&startedTS, &recoveredTS,
			&inc.DurationSec, &inc.CheckCount, &inc.FirstError,
		)
		if err != nil {
			return nil, err
		}
		inc.StartedAt = time.Unix(startedTS, 0).UTC()
		if recoveredTS != nil {
			t := time.Unix(*recoveredTS, 0).UTC()
			inc.RecoveredAt = &t
		}
		incidents = append(incidents, inc)
	}
	return incidents, rows.Err()
}
