package database

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// AuditRepo persists the audit trail (spec section 36). Secret values are
// redacted upstream before reaching this layer.
type AuditRepo struct{ db *DB }

func NewAuditRepo(db *DB) *AuditRepo { return &AuditRepo{db: db} }

// AuditEntry is one audit record.
type AuditEntry struct {
	ID            string
	Timestamp     time.Time
	Actor         string
	SourceIP      string
	Action        string
	Target        string
	BeforeSummary string
	AfterSummary  string
	Success       bool
	RequestID     string
	Detail        string
}

// Insert writes one entry.
func (r *AuditRepo) Insert(e AuditEntry) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if e.Timestamp.IsZero() {
		e.Timestamp = time.Now().UTC()
	}
	_, err := r.db.SQL().Exec(`INSERT INTO audit_log
		(id, ts, actor, source_ip, action, target, before_summary, after_summary, success, request_id, detail)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		e.ID, e.Timestamp.Format(time.RFC3339Nano), e.Actor, e.SourceIP, e.Action, e.Target,
		e.BeforeSummary, e.AfterSummary, boolInt(e.Success), e.RequestID, e.Detail)
	return err
}

// List returns the most recent entries with pagination (spec section 48:
// never load the entire audit log at once).
func (r *AuditRepo) List(limit, offset int) ([]AuditEntry, error) {
	if limit < 1 || limit > 500 {
		limit = 100
	}
	rows, err := r.db.SQL().Query(`SELECT id, ts, actor, source_ip, action, target,
		before_summary, after_summary, success, request_id, detail
		FROM audit_log ORDER BY ts DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var e AuditEntry
		var ts string
		var success int
		var srcIP, target, before, after, reqID, detail sql.NullString
		if err := rows.Scan(&e.ID, &ts, &e.Actor, &srcIP, &e.Action, &target,
			&before, &after, &success, &reqID, &detail); err != nil {
			return nil, err
		}
		e.Timestamp = parseTime(ts)
		e.Success = success == 1
		e.SourceIP = srcIP.String
		e.Target = target.String
		e.BeforeSummary = before.String
		e.AfterSummary = after.String
		e.RequestID = reqID.String
		e.Detail = detail.String
		out = append(out, e)
	}
	return out, rows.Err()
}

// NotificationRepo persists deduplicated notifications (spec sections 10.2,
// 12). Dedup key is code+component+context fingerprint.
type NotificationRepo struct{ db *DB }

func NewNotificationRepo(db *DB) *NotificationRepo { return &NotificationRepo{db: db} }

// Notification is a persisted user-visible issue.
type Notification struct {
	ID          string
	Code        string
	Severity    string // info|warning|error|critical
	Category    string
	Component   string
	Summary     string
	Details     string
	Remediation string
	Fingerprint string
	FirstSeen   time.Time
	LastSeen    time.Time
	Occurrences int
	Resolved    bool
	ResolvedAt  *time.Time
	Read        bool
}

// Upsert inserts or deduplicates by fingerprint, incrementing occurrences
// and refreshing last_seen (spec section 10.2: no duplicate notifications
// every boot).
func (r *NotificationRepo) Upsert(n Notification) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.SQL().Exec(`INSERT INTO notifications
		(id, code, severity, category, component, summary, details, remediation, fingerprint,
		 first_seen_at, last_seen_at, occurrences, resolved, read)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,1,0,0)
		ON CONFLICT(fingerprint) DO UPDATE SET
			last_seen_at = excluded.last_seen_at,
			occurrences = occurrences + 1,
			severity = excluded.severity,
			summary = excluded.summary,
			details = excluded.details,
			remediation = excluded.remediation`,
		uuid.NewString(), n.Code, n.Severity, n.Category, n.Component, n.Summary,
		n.Details, n.Remediation, n.Fingerprint, now, now)
	return err
}

// Resolve marks a fingerprint resolved (issue disappeared, spec section 54
// step 14).
func (r *NotificationRepo) Resolve(fingerprint string) error {
	_, err := r.db.SQL().Exec(`UPDATE notifications
		SET resolved = 1, resolved_at = ? WHERE fingerprint = ? AND resolved = 0`,
		time.Now().UTC().Format(time.RFC3339), fingerprint)
	return err
}

// ListUnresolved returns active notifications by severity.
func (r *NotificationRepo) ListUnresolved() ([]Notification, error) {
	rows, err := r.db.SQL().Query(`SELECT id, code, severity, category, component, summary,
		details, remediation, fingerprint, first_seen_at, last_seen_at, occurrences, resolved, read
		FROM notifications WHERE resolved = 0
		ORDER BY CASE severity WHEN 'critical' THEN 0 WHEN 'error' THEN 1
			WHEN 'warning' THEN 2 ELSE 3 END, first_seen_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanNotifications(rows)
}

// MarkRead toggles read state.
func (r *NotificationRepo) MarkRead(id string, read bool) error {
	_, err := r.db.SQL().Exec(`UPDATE notifications SET read = ? WHERE id = ?`, boolInt(read), id)
	return err
}

func scanNotifications(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Notification, error) {
	var out []Notification
	for rows.Next() {
		var n Notification
		var first, last string
		var resolved, read int
		if err := rows.Scan(&n.ID, &n.Code, &n.Severity, &n.Category, &n.Component, &n.Summary,
			&n.Details, &n.Remediation, &n.Fingerprint, &first, &last, &n.Occurrences, &resolved, &read); err != nil {
			return nil, err
		}
		n.FirstSeen = parseTime(first)
		n.LastSeen = parseTime(last)
		n.Resolved = resolved == 1
		n.Read = read == 1
		out = append(out, n)
	}
	return out, rows.Err()
}
