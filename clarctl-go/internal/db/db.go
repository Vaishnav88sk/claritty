// Package db provides SQLite-backed persistence for incidents and snapshots.
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
	"github.com/Vaishnav88sk/claritty/clarctl-go/internal/incident"
)

// DB wraps the SQLite connection pool.
type DB struct {
	conn *sql.DB
}

// Open opens or creates the SQLite database at the given path.
func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	conn.SetMaxOpenConns(1) // SQLite is single-writer
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return d, nil
}

// Close closes the underlying database connection.
func (d *DB) Close() error { return d.conn.Close() }

// ─── Schema ─────────────────────────────────────────────────────────────────

func (d *DB) migrate() error {
	_, err := d.conn.Exec(`
		CREATE TABLE IF NOT EXISTS incidents (
			id          TEXT PRIMARY KEY,
			created_at  DATETIME NOT NULL,
			updated_at  DATETIME NOT NULL,
			severity    TEXT NOT NULL,
			title       TEXT NOT NULL,
			category    TEXT,
			status      TEXT NOT NULL,
			has_issue   INTEGER NOT NULL DEFAULT 1,
			payload     TEXT NOT NULL  -- full JSON blob
		);
		CREATE INDEX IF NOT EXISTS idx_incidents_created ON incidents(created_at DESC);
		CREATE INDEX IF NOT EXISTS idx_incidents_status  ON incidents(status);

		CREATE TABLE IF NOT EXISTS snapshots (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp  DATETIME NOT NULL,
			payload    TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_snapshots_ts ON snapshots(timestamp DESC);
	`)
	return err
}

// ─── Incidents ───────────────────────────────────────────────────────────────

// SaveIncident upserts a full incident report.
func (d *DB) SaveIncident(r *incident.Report) error {
	payload, err := json.Marshal(r)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(`
		INSERT INTO incidents(id, created_at, updated_at, severity, title, category, status, has_issue, payload)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			updated_at = excluded.updated_at,
			status     = excluded.status,
			payload    = excluded.payload
	`,
		r.ID, r.CreatedAt, r.UpdatedAt, string(r.Severity),
		r.Title, r.Category, string(r.Status),
		boolToInt(r.HasIssue), string(payload),
	)
	return err
}

// UpdateStatus updates only the status field of an incident.
func (d *DB) UpdateStatus(id string, status incident.Status) error {
	_, err := d.conn.Exec(
		`UPDATE incidents SET status=?, updated_at=? WHERE id=?`,
		string(status), time.Now().UTC(), id,
	)
	return err
}

// GetByID fetches a single incident report by ID.
func (d *DB) GetByID(id string) (*incident.Report, error) {
	row := d.conn.QueryRow(`SELECT payload FROM incidents WHERE id=?`, id)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return unmarshalReport(payload)
}

// GetIncidents retrieves a filtered list of incident reports.
func (d *DB) GetIncidents(severity, status string, hours, limit int) ([]*incident.Report, error) {
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	args := []any{since}
	q := `SELECT payload FROM incidents WHERE created_at >= ?`
	if severity != "" {
		q += ` AND severity=?`
		args = append(args, severity)
	}
	if status != "" {
		q += ` AND status=?`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT ?`
	args = append(args, limit)

	rows, err := d.conn.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*incident.Report
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		r, err := unmarshalReport(p)
		if err != nil {
			continue
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// OpenIncidentCount returns the count of open incidents.
func (d *DB) OpenIncidentCount() (int, error) {
	var n int
	err := d.conn.QueryRow(`SELECT COUNT(*) FROM incidents WHERE status='OPEN'`).Scan(&n)
	return n, err
}

// MTTRStats returns average MTTR in seconds for mitigated incidents.
func (d *DB) MTTRStats() (avg float64, count int, err error) {
	rows, err := d.conn.Query(
		`SELECT payload FROM incidents WHERE status IN ('MITIGATED','RESOLVED') ORDER BY created_at DESC LIMIT 50`,
	)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()

	var total float64
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			continue
		}
		r, err := unmarshalReport(p)
		if err != nil {
			continue
		}
		if r.MTTRSeconds != nil {
			total += float64(*r.MTTRSeconds)
			count++
		}
	}
	if count > 0 {
		avg = total / float64(count)
	}
	return avg, count, nil
}

// ─── Snapshots ───────────────────────────────────────────────────────────────

// SaveSnapshot persists a cluster health snapshot.
func (d *DB) SaveSnapshot(s *incident.ClusterSnapshot) error {
	payload, err := json.Marshal(s)
	if err != nil {
		return err
	}
	_, err = d.conn.Exec(
		`INSERT INTO snapshots(timestamp, payload) VALUES(?,?)`,
		s.Timestamp, string(payload),
	)
	return err
}

// LatestSnapshot returns the most recent cluster health snapshot.
func (d *DB) LatestSnapshot() (*incident.ClusterSnapshot, error) {
	row := d.conn.QueryRow(`SELECT payload FROM snapshots ORDER BY timestamp DESC LIMIT 1`)
	var payload string
	if err := row.Scan(&payload); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	var s incident.ClusterSnapshot
	return &s, json.Unmarshal([]byte(payload), &s)
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func unmarshalReport(payload string) (*incident.Report, error) {
	var r incident.Report
	return &r, json.Unmarshal([]byte(payload), &r)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
