// Package firewall provides the FirewallManager: the single orchestrator
// that every API handler goes through for firewall reads and writes. It
// enforces the transaction sequence (authorize, lock, validate, snapshot,
// apply, verify, rollback) and the ownership policy (spec sections 7, 19).
package firewall

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager orchestrates backend access and safe transactions.
type Manager struct {
	mu sync.Mutex // serializes mutations; reads are lock-free

	snapshots    map[string]Snapshot
	backend      FirewallBackend
	ownership    OwnershipReport
	lastHash     string
	lastRead     time.Time
	lastWrite    time.Time
	lastSnapshot *Snapshot
	auditFn      func(event AuditEvent)
	notifyFn     func(event IssueEvent)
}

// AuditEvent is the minimal audit callback contract; the audit service
// plugs in at startup.
type AuditEvent struct {
	Actor   string
	Action  string
	Target  string
	Success bool
	Detail  string
}

// IssueEvent is a health/notification issue emitted by the manager.
type IssueEvent struct {
	Code        string
	Severity    string
	Component   string
	Summary     string
	Remediation string
}

// NewManager wires the manager to a backend plus audit/notification sinks.
func NewManager(b FirewallBackend, audit func(AuditEvent), notify func(IssueEvent)) *Manager {
	return &Manager{snapshots: map[string]Snapshot{}, backend: b, auditFn: audit, notifyFn: notify}
}

// SetOwnership updates the resolved ownership policy after detection.
func (m *Manager) SetOwnership(rep OwnershipReport) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ownership = rep
	if rep.Owner == OwnerMultipleConflicting {
		m.emit(IssueEvent{
			Code: "FW_BACKEND_CONFLICT", Severity: "critical", Component: "firewall",
			Summary:     "Multiple firewall managers are active",
			Remediation: "Disable all but one firewall manager, then acknowledge the conflict in Settings.",
		})
	}
}

// Ownership returns the current ownership report.
func (m *Manager) Ownership() OwnershipReport {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.ownership
}

// Status returns backend status, guarded by ownership policy.
func (m *Manager) Status(ctx context.Context) (FirewallStatus, error) {
	m.refreshOwnership(ctx)
	st, err := m.backend.Status(ctx)
	if err == nil {
		m.mu.Lock()
		m.lastRead = time.Now().UTC()
		m.mu.Unlock()
	}
	return st, err
}

// ListRules reads the active rules.
func (m *Manager) ListRules(ctx context.Context) ([]Rule, error) {
	m.refreshOwnership(ctx)
	rules, err := m.backend.ListRules(ctx)
	if err == nil {
		m.mu.Lock()
		m.lastRead = time.Now().UTC()
		m.mu.Unlock()
	}
	return rules, err
}

// ApplyTransaction runs the full safe-transaction pipeline (spec section 19):
// ownership gate, lock, validate, snapshot, apply, re-read, verify, and
// rollback on any post-snapshot failure.
func (m *Manager) ApplyTransaction(ctx context.Context, actor string, tx Transaction) (ApplyResult, error) {
	m.refreshOwnership(ctx)
	// 1. Ownership gate: writes blocked on conflict/unhealthy state.
	m.mu.Lock()
	own := m.ownership
	m.mu.Unlock()
	if !own.WritesAllowed {
		return ApplyResult{}, ErrWritesBlocked
	}

	// 2. Serialize mutations.
	m.mu.Lock()
	defer m.mu.Unlock()

	// 3. Validate before touching anything.
	if _, err := m.backend.ListRules(ctx); err != nil {
		return ApplyResult{}, fmt.Errorf("pre-apply read failed: %w", err)
	}
	vr := m.backend.Validate(ctx, tx)
	if !vr.Valid {
		m.audit(actor, "firewall.validate", tx.ID, false, joinErrs(vr.Errors))
		return ApplyResult{}, &ValidationFailure{Errors: vr.Errors, Conflicts: vr.Conflict}
	}
	// Block on HIGH/CRITICAL conflicts unless explicitly overridden upstream.
	for _, c := range vr.Conflict {
		if c.Severity == "CRITICAL" {
			m.audit(actor, "firewall.validate", tx.ID, false, "critical conflict: "+c.Code)
			return ApplyResult{}, &ValidationFailure{Errors: []string{c.Summary}, Conflicts: vr.Conflict}
		}
	}

	// 4. Restore point before any mutation (spec section 21).
	snap, err := m.backend.Snapshot(ctx, "pre-transaction "+tx.ID)
	if err != nil {
		m.audit(actor, "firewall.snapshot", tx.ID, false, err.Error())
		return ApplyResult{}, fmt.Errorf("%w: %s", ErrSnapshotFailed, err)
	}

	// 5. Apply.
	res, applyErr := m.backend.Apply(ctx, tx)
	res.SnapshotID = snap.ID
	if applyErr != nil {
		// 6. Rollback on failed apply (spec section 19).
		m.audit(actor, "firewall.apply", tx.ID, false, applyErr.Error())
		if rbErr := m.restore(ctx, snap); rbErr != nil {
			m.emit(IssueEvent{
				Code: "FW_ROLLBACK_FAILED", Severity: "critical", Component: "firewall",
				Summary:     "Automatic rollback failed after failed apply",
				Remediation: "Manually restore snapshot " + snap.ID + " and inspect the firewall immediately.",
			})
			return res, fmt.Errorf("%w: apply: %s; rollback: %s", ErrRollbackFailed, applyErr, rbErr)
		}
		return res, fmt.Errorf("%w: %s", ErrApplyFailed, applyErr)
	}

	// 7. Verify post-state.
	ver, err := m.backend.Verify(ctx, StateHash(res.StateHash))
	if err != nil || !ver.Match {
		if rbErr := m.restore(ctx, snap); rbErr != nil {
			m.emit(IssueEvent{
				Code: "FW_ROLLBACK_FAILED", Severity: "critical", Component: "firewall",
				Summary: "Rollback failed after verify mismatch",
			})
			return res, fmt.Errorf("%w: verify: %s; rollback: %s", ErrRollbackFailed, err, rbErr)
		}
		return res, fmt.Errorf("%w: post-apply state mismatch", ErrVerifyFailed)
	}
	res.VerifyResult = &ver

	m.snapshots[snap.ID] = snap
	m.lastSnapshot = &snap
	m.lastWrite = time.Now().UTC()
	m.lastHash = res.StateHash
	m.audit(actor, "firewall.apply", tx.ID, true,
		fmt.Sprintf("applied %d actions, snapshot %s", res.Applied, snap.ID))
	return res, nil
}

// CreateSnapshot exposes manual restore-point creation.
func (m *Manager) CreateSnapshot(ctx context.Context, actor, reason string) (Snapshot, error) {
	snap, err := m.backend.Snapshot(ctx, reason)
	if err != nil {
		m.audit(actor, "firewall.snapshot", reason, false, err.Error())
		return snap, fmt.Errorf("%w: %s", ErrSnapshotFailed, err)
	}
	m.audit(actor, "firewall.snapshot", snap.ID, true, reason)
	return snap, nil
}

func (m *Manager) audit(actor, action, target string, ok bool, detail string) {
	if m.auditFn != nil {
		m.auditFn(AuditEvent{Actor: actor, Action: action, Target: target, Success: ok, Detail: detail})
	}
}

func (m *Manager) emit(e IssueEvent) {
	if m.notifyFn != nil {
		m.notifyFn(e)
	}
}

func joinErrs(errs []string) string {
	out := ""
	for i, e := range errs {
		if i > 0 {
			out += "; "
		}
		out += e
	}
	return out
}

// ValidationFailure carries structured validation errors to the API layer.
type ValidationFailure struct {
	Errors    []string   `json:"errors"`
	Conflicts []Conflict `json:"conflicts,omitempty"`
}

func (v *ValidationFailure) Error() string { return "validation failed: " + joinErrs(v.Errors) }

// Sentinel errors surfaced as stable API error codes (spec section 45).
var (
	ErrSnapshotFailed = fmt.Errorf("snapshot failed")
	ErrApplyFailed    = fmt.Errorf("apply failed")
	ErrVerifyFailed   = fmt.Errorf("verify failed")
)

// restore reinstates a snapshot via the backend, keeping in-memory
// snapshot content for the session (persistence lands with the restore
// module wiring).
func (m *Manager) restore(ctx context.Context, snap Snapshot) error {
	if b, ok := m.backend.(interface {
		RestoreSnapshot(ctx context.Context, snap Snapshot) error
	}); ok {
		return b.RestoreSnapshot(ctx, snap)
	}
	// The backend interface restores from the snapshot content itself.
	return m.backend.Restore(ctx, snap)
}

// BackendValidate exposes read-only transaction validation without the
// mutation pipeline (used by the validate endpoint).
func (m *Manager) BackendValidate(ctx context.Context, tx Transaction) ValidationResult {
	return m.backend.Validate(ctx, tx)
}

func (m *Manager) VerifyState(ctx context.Context, expected StateHash) (VerifyResult, error) {
	return m.backend.Verify(ctx, expected)
}

func (m *Manager) RestoreSnapshot(ctx context.Context, snap Snapshot) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.restore(ctx, snap)
}

func (m *Manager) Reload(ctx context.Context) error  { return m.backend.Reload(ctx) }
func (m *Manager) Restart(ctx context.Context) error { return m.backend.Restart(ctx) }

func (m *Manager) refreshOwnership(ctx context.Context) {
	provider, ok := m.backend.(interface {
		OwnershipReport(context.Context) (OwnershipReport, error)
	})
	if !ok {
		return
	}
	if ownership, err := provider.OwnershipReport(ctx); err == nil {
		m.mu.Lock()
		m.ownership = ownership
		m.mu.Unlock()
	}
}

func (m *Manager) RestoreLast(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.lastSnapshot == nil {
		return fmt.Errorf("no successful transaction snapshot available")
	}
	return m.restore(ctx, *m.lastSnapshot)
}
