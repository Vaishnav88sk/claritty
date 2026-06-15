// Package db implements PostgreSQL-backed persistence for the hub.
package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Store is the PostgreSQL-backed data store.
type Store struct {
	db *sql.DB
}

// New opens a PostgreSQL connection and runs migrations.
func New(databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping db: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return s, nil
}

// NewWithDB creates a Store with an existing sql.DB connection.
// Useful for testing with tools like go-sqlmock.
func NewWithDB(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
	CREATE TABLE IF NOT EXISTS clusters (
		name         TEXT PRIMARY KEY,
		last_seen    TIMESTAMP NOT NULL,
		health_score FLOAT,
		ready_nodes  INT,
		total_nodes  INT,
		running_pods INT,
		pending_pods INT,
		failed_pods  INT,
		crashloop    INT,
		namespaces   TEXT
	);

	CREATE TABLE IF NOT EXISTS incidents (
		id                   TEXT PRIMARY KEY,
		cluster              TEXT NOT NULL REFERENCES clusters(name),
		namespace            TEXT,
		severity             TEXT,
		status               TEXT NOT NULL DEFAULT 'INVESTIGATING',
		title                TEXT,
		category             TEXT,
		root_cause           TEXT,
		contributing_factors JSONB,
		affected_services    JSONB,
		affected_namespaces  JSONB,
		remediation_plan     JSONB,
		llm_model            TEXT,
		confidence           INT,
		has_issue            BOOLEAN,
		detected_at          TIMESTAMP NOT NULL,
		resolved_at          TIMESTAMP,
		scan_duration_secs   FLOAT
	);

	CREATE TABLE IF NOT EXISTS remediation_log (
		id           SERIAL PRIMARY KEY,
		incident_id  TEXT REFERENCES incidents(id),
		cluster      TEXT,
		step_number  INT,
		description  TEXT,
		command      TEXT,
		status       TEXT,
		executed_at  TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_incidents_cluster ON incidents(cluster);
	CREATE INDEX IF NOT EXISTS idx_incidents_detected_at ON incidents(detected_at DESC);
	CREATE INDEX IF NOT EXISTS idx_incidents_severity ON incidents(severity);
	`)
	return err
}

// ─── Cluster operations ───────────────────────────────────────────────────────

type ClusterRow struct {
	Name        string    `json:"name"`
	LastSeen    time.Time `json:"last_seen"`
	HealthScore float64   `json:"health_score"`
	ReadyNodes  int       `json:"ready_nodes"`
	TotalNodes  int       `json:"total_nodes"`
	RunningPods int       `json:"running_pods"`
	PendingPods int       `json:"pending_pods"`
	FailedPods  int       `json:"failed_pods"`
	Crashloop   int       `json:"crashloop"`
	Namespaces  []string  `json:"namespaces"`
}

func (s *Store) UpsertCluster(c *ClusterRow) error {
	nsJSON, _ := json.Marshal(c.Namespaces)
	_, err := s.db.Exec(`
		INSERT INTO clusters (name, last_seen, health_score, ready_nodes, total_nodes, running_pods, pending_pods, failed_pods, crashloop, namespaces)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (name) DO UPDATE SET
			last_seen=EXCLUDED.last_seen, health_score=EXCLUDED.health_score,
			ready_nodes=EXCLUDED.ready_nodes, total_nodes=EXCLUDED.total_nodes,
			running_pods=EXCLUDED.running_pods, pending_pods=EXCLUDED.pending_pods,
			failed_pods=EXCLUDED.failed_pods, crashloop=EXCLUDED.crashloop,
			namespaces=EXCLUDED.namespaces`,
		c.Name, c.LastSeen, c.HealthScore, c.ReadyNodes, c.TotalNodes,
		c.RunningPods, c.PendingPods, c.FailedPods, c.Crashloop, string(nsJSON))
	return err
}

func (s *Store) ListClusters() ([]ClusterRow, error) {
	rows, err := s.db.Query(`SELECT name, last_seen, health_score, ready_nodes, total_nodes, running_pods, pending_pods, failed_pods, crashloop, namespaces FROM clusters ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ClusterRow
	for rows.Next() {
		var c ClusterRow
		var nsJSON string
		if err := rows.Scan(&c.Name, &c.LastSeen, &c.HealthScore, &c.ReadyNodes, &c.TotalNodes, &c.RunningPods, &c.PendingPods, &c.FailedPods, &c.Crashloop, &nsJSON); err != nil {
			continue
		}
		_ = json.Unmarshal([]byte(nsJSON), &c.Namespaces)
		out = append(out, c)
	}
	return out, nil
}

// ─── Incident operations ──────────────────────────────────────────────────────

type IncidentRow struct {
	ID                  string          `json:"id"`
	Cluster             string          `json:"cluster"`
	Namespace           string          `json:"namespace"`
	Severity            string          `json:"severity"`
	Status              string          `json:"status"`
	Title               string          `json:"title"`
	Category            string          `json:"category"`
	RootCause           string          `json:"root_cause"`
	ContributingFactors json.RawMessage `json:"contributing_factors"`
	AffectedServices    json.RawMessage `json:"affected_services"`
	AffectedNamespaces  json.RawMessage `json:"affected_namespaces"`
	RemediationPlan     json.RawMessage `json:"remediation_plan"`
	LLMModel            string          `json:"llm_model"`
	Confidence          int             `json:"confidence_score"`
	HasIssue            bool            `json:"has_issue"`
	DetectedAt          time.Time       `json:"detected_at"`
	ResolvedAt          *time.Time      `json:"resolved_at,omitempty"`
	ScanDurationSecs    float64         `json:"scan_duration_secs"`
}

func (s *Store) InsertIncident(inc *IncidentRow) error {
	cf, _ := json.Marshal(inc.ContributingFactors)
	as, _ := json.Marshal(inc.AffectedServices)
	an, _ := json.Marshal(inc.AffectedNamespaces)
	rp, _ := json.Marshal(inc.RemediationPlan)

	// Ensure cluster exists
	_, _ = s.db.Exec(`INSERT INTO clusters (name, last_seen, health_score) VALUES ($1, NOW(), 0) ON CONFLICT DO NOTHING`, inc.Cluster)

	_, err := s.db.Exec(`
		INSERT INTO incidents (id, cluster, namespace, severity, status, title, category, root_cause,
			contributing_factors, affected_services, affected_namespaces, remediation_plan,
			llm_model, confidence, has_issue, detected_at, scan_duration_secs)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		ON CONFLICT (id) DO NOTHING`,
		inc.ID, inc.Cluster, inc.Namespace, inc.Severity, inc.Status, inc.Title, inc.Category, inc.RootCause,
		string(cf), string(as), string(an), string(rp),
		inc.LLMModel, inc.Confidence, inc.HasIssue, inc.DetectedAt, inc.ScanDurationSecs)
	return err
}

// ListIncidentsFilter holds filter options for listing incidents.
type ListIncidentsFilter struct {
	Cluster   string
	Namespace string
	Severity  string
	Status    string
	Limit     int
}

func (s *Store) ListIncidents(f ListIncidentsFilter) ([]IncidentRow, error) {
	if f.Limit == 0 {
		f.Limit = 100
	}
	q := `SELECT id, cluster, namespace, severity, status, title, category, root_cause,
		contributing_factors, affected_services, affected_namespaces, remediation_plan,
		llm_model, confidence, has_issue, detected_at, resolved_at, scan_duration_secs
		FROM incidents WHERE 1=1`
	args := []interface{}{}
	n := 1
	if f.Cluster != "" {
		q += fmt.Sprintf(" AND cluster=$%d", n)
		args = append(args, f.Cluster)
		n++
	}
	if f.Namespace != "" {
		q += fmt.Sprintf(" AND namespace=$%d", n)
		args = append(args, f.Namespace)
		n++
	}
	if f.Severity != "" {
		q += fmt.Sprintf(" AND severity=$%d", n)
		args = append(args, f.Severity)
		n++
	}
	if f.Status != "" {
		q += fmt.Sprintf(" AND status=$%d", n)
		args = append(args, f.Status)
		n++
	}
	q += fmt.Sprintf(" ORDER BY detected_at DESC LIMIT $%d", n)
	args = append(args, f.Limit)

	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanIncidentRows(rows)
}

func (s *Store) GetIncident(id string) (*IncidentRow, error) {
	row := s.db.QueryRow(`SELECT id, cluster, namespace, severity, status, title, category, root_cause,
		contributing_factors, affected_services, affected_namespaces, remediation_plan,
		llm_model, confidence, has_issue, detected_at, resolved_at, scan_duration_secs
		FROM incidents WHERE id=$1`, id)
	rows, err := scanIncidentRows(asRows(row))
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	return &rows[0], nil
}

func (s *Store) UpdateIncidentStatus(id, status string) error {
	var resolvedAt interface{}
	if status == "RESOLVED" || status == "MITIGATED" {
		resolvedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`UPDATE incidents SET status=$1, resolved_at=$2 WHERE id=$3`, status, resolvedAt, id)
	return err
}

func (s *Store) GetStats() (map[string]interface{}, error) {
	stats := map[string]interface{}{}
	row := s.db.QueryRow(`SELECT COUNT(*) FROM incidents`)
	var total int
	_ = row.Scan(&total)
	stats["total_incidents"] = total

	row = s.db.QueryRow(`SELECT COUNT(*) FROM incidents WHERE status='INVESTIGATING'`)
	var open int
	_ = row.Scan(&open)
	stats["open_incidents"] = open

	row = s.db.QueryRow(`
		SELECT COALESCE(AVG(EXTRACT(EPOCH FROM (resolved_at - detected_at))), 0)
		FROM incidents WHERE resolved_at IS NOT NULL`)
	var mttr float64
	_ = row.Scan(&mttr)
	stats["avg_mttr_secs"] = mttr

	return stats, nil
}

// ─── helpers ──────────────────────────────────────────────────────────────────

type singleRow struct {
	row  *sql.Row
	done bool
}

func (s *singleRow) Next() bool {
	if s.done {
		return false
	}
	s.done = true
	return true
}

func (s *singleRow) Scan(dest ...interface{}) error {
	return s.row.Scan(dest...)
}

func (s *singleRow) Close() error { return nil }

type scanner interface {
	Next() bool
	Scan(dest ...interface{}) error
	Close() error
}

func asRows(row *sql.Row) scanner {
	return &singleRow{row: row}
}

func scanIncidentRows(rows scanner) ([]IncidentRow, error) {
	defer rows.Close()
	var out []IncidentRow
	for rows.Next() {
		var inc IncidentRow
		var cf, as, an, rp []byte
		if err := rows.Scan(&inc.ID, &inc.Cluster, &inc.Namespace, &inc.Severity, &inc.Status,
			&inc.Title, &inc.Category, &inc.RootCause, &cf, &as, &an, &rp,
			&inc.LLMModel, &inc.Confidence, &inc.HasIssue, &inc.DetectedAt, &inc.ResolvedAt, &inc.ScanDurationSecs); err != nil {
			continue
		}
		inc.ContributingFactors = json.RawMessage(cf)
		inc.AffectedServices = json.RawMessage(as)
		inc.AffectedNamespaces = json.RawMessage(an)
		inc.RemediationPlan = json.RawMessage(rp)
		out = append(out, inc)
	}
	return out, nil
}
