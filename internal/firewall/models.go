// Package firewall defines the backend abstraction, normalized rule model,
// ownership detection, and the FirewallManager that orchestrates
// transactions, validation, conflicts, snapshots and rollback
// (spec sections 6, 7, 17 to 21, 37).
package firewall

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AddressFamily of a rule.
type Family string

const (
	FamilyIPv4 Family = "ipv4"
	FamilyIPv6 Family = "ipv6"
	FamilyBoth Family = "both"
)

// Direction of traffic a rule applies to.
type Direction string

const (
	DirIn      Direction = "in"
	DirOut     Direction = "out"
	DirForward Direction = "forward"
)

// Action a rule takes.
type Action string

const (
	ActionAllow  Action = "allow"
	ActionDeny   Action = "deny"
	ActionReject Action = "reject"
	ActionDrop   Action = "drop"
	ActionLog    Action = "log"
)

// Protocol for a rule.
type Protocol string

const (
	ProtoAny    Protocol = "any"
	ProtoTCP    Protocol = "tcp"
	ProtoUDP    Protocol = "udp"
	ProtoICMP   Protocol = "icmp"
	ProtoCustom Protocol = "custom"
)

// Rule is the normalized, backend-agnostic firewall rule (spec section 17).
type Rule struct {
	ID              string    `json:"id"`
	BackendID       string    `json:"backend_id,omitempty"`
	Family          Family    `json:"family"`
	Direction       Direction `json:"direction"`
	Action          Action    `json:"action"`
	Protocol        Protocol  `json:"protocol"`
	Source          string    `json:"source,omitempty"`
	SourcePort      string    `json:"source_port,omitempty"`
	Destination     string    `json:"destination,omitempty"`
	DestinationPort string    `json:"destination_port,omitempty"`
	InterfaceIn     string    `json:"interface_in,omitempty"`
	InterfaceOut    string    `json:"interface_out,omitempty"`
	State           string    `json:"state,omitempty"` // e.g. ESTABLISHED,RELATED
	Comment         string    `json:"comment,omitempty"`
	Enabled         bool      `json:"enabled"`
	Priority        int       `json:"priority"`
	Backend         string    `json:"backend"`
	RawReference    string    `json:"raw_reference,omitempty"`
	CreatedBy       string    `json:"created_by,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// NewRuleID returns a fresh stable UUID for a rule.
func NewRuleID() string { return uuid.NewString() }

// DefaultPolicy is the chain default (allow/deny) for a direction.
type DefaultPolicy struct {
	Direction Direction `json:"direction"`
	Action    Action    `json:"action"`
}

// FirewallStatus is what status endpoints and the dashboard consume.
type FirewallStatus struct {
	Backend         string          `json:"backend"`
	Enabled         bool            `json:"enabled"`
	DefaultPolicies []DefaultPolicy `json:"default_policies"`
	RuleCount       int             `json:"rule_count"`
	IPv4Ready       bool            `json:"ipv4_ready"`
	IPv6Ready       bool            `json:"ipv6_ready"`
	RawVersion      string          `json:"raw_version,omitempty"`
	CheckedAt       time.Time       `json:"checked_at"`
}

// DetectionResult reports whether a backend is usable on this host.
type DetectionResult struct {
	Name       string `json:"name"`
	Found      bool   `json:"found"`
	BinaryPath string `json:"binary_path,omitempty"`
	Version    string `json:"version,omitempty"`
	Active     bool   `json:"active"`
	Details    string `json:"details,omitempty"`
}

// ValidationResult carries static-analysis outcome for a rule or transaction.
type ValidationResult struct {
	Valid    bool       `json:"valid"`
	Errors   []string   `json:"errors,omitempty"`
	Conflict []Conflict `json:"conflicts,omitempty"`
}

// Conflict is a detected rule-interaction problem (spec section 18).
type Conflict struct {
	Code        string   `json:"code"`     // DUPLICATE_RULE, RULE_SHADOWED, ...
	Severity    string   `json:"severity"` // INFO|WARNING|HIGH|CRITICAL
	Summary     string   `json:"summary"`
	Details     string   `json:"details,omitempty"`
	RuleIDs     []string `json:"rule_ids,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

// Snapshot captures the pre-change firewall state for rollback (spec section 21).
type Snapshot struct {
	ID            string    `json:"id"`
	Reason        string    `json:"reason"`
	Backend       string    `json:"backend"`
	BackendVer    string    `json:"backend_version,omitempty"`
	IptablesSave  string    `json:"iptables_save,omitempty"`
	IP6TablesSave string    `json:"ip6tables_save,omitempty"`
	UFWStatus     string    `json:"ufw_status,omitempty"`
	StateHash     string    `json:"state_hash"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedBy     string    `json:"created_by,omitempty"`
	Notes         string    `json:"notes,omitempty"`
}

// Transaction is a batched firewall mutation (spec section 19).
type Transaction struct {
	ID        string      `json:"id"`
	Backend   string      `json:"backend"`
	Actions   []TxnAction `json:"actions"`
	CreatedBy string      `json:"created_by"`
	CreatedAt time.Time   `json:"created_at"`
}

// TxnAction is one step inside a transaction.
type TxnAction struct {
	Op     string `json:"op"` // add|update|delete|enable|disable|reorder|set_policy
	Rule   *Rule  `json:"rule,omitempty"`
	RuleID string `json:"rule_id,omitempty"`
}

// ApplyResult reports what a successful apply changed.
type ApplyResult struct {
	TxnID        string        `json:"txn_id"`
	Applied      int           `json:"applied"`
	SnapshotID   string        `json:"snapshot_id"`
	StateHash    string        `json:"state_hash"`
	DurationMs   int64         `json:"duration_ms"`
	Warnings     []string      `json:"warnings,omitempty"`
	VerifyResult *VerifyResult `json:"verify_result,omitempty"`
}

// VerifyResult compares observed post-apply state to the expected hash.
type VerifyResult struct {
	Match    bool   `json:"match"`
	Observed string `json:"observed_hash"`
	Expected string `json:"expected_hash"`
	Details  string `json:"details,omitempty"`
}

// StateHashable is implemented by backends so the drift detector can
// compare stable hashes of observed vs stored state (spec section 37).
type StateHashable interface {
	StateHash(ctx context.Context) (string, error)
}

// FirewallBackend is the abstraction all adapters implement (spec section 6).
type FirewallBackend interface {
	Name() string
	Detect(ctx context.Context) DetectionResult
	Status(ctx context.Context) (FirewallStatus, error)
	ListRules(ctx context.Context) ([]Rule, error)
	Validate(ctx context.Context, tx Transaction) ValidationResult
	Snapshot(ctx context.Context, reason string) (Snapshot, error)
	Apply(ctx context.Context, tx Transaction) (ApplyResult, error)
	Verify(ctx context.Context, expected StateHash) (VerifyResult, error)
	Restore(ctx context.Context, snap Snapshot) error
	Reload(ctx context.Context) error
	Restart(ctx context.Context) error
}

// StateHash is a stable hash of observed state used for verify and drift.
type StateHash string

// ErrBackendUnavailable is returned when a backend binary/service is missing.
var ErrBackendUnavailable = fmt.Errorf("firewall backend unavailable")

// ErrWritesBlocked is returned when ownership conflicts or unhealthy state
// forbid mutation (spec section 7).
var ErrWritesBlocked = fmt.Errorf("firewall writes blocked: resolve conflicts first")

// ErrCommandFailed is returned when a backend command exits non-zero.
var ErrCommandFailed = fmt.Errorf("firewall command failed")

// ErrRollbackFailed is returned when restoring a snapshot fails; this is a
// critical safety event and MUST surface as such (spec sections 19, 45).
var ErrRollbackFailed = fmt.Errorf("firewall rollback failed")
