package database

import (
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FleetRepo struct{ db *DB }

func NewFleetRepo(db *DB) *FleetRepo { return &FleetRepo{db: db} }

type AgentRecord struct {
	AgentID           string          `json:"agent_id"`
	DisplayName       string          `json:"display_name"`
	Hostname          string          `json:"hostname"`
	PrimaryIP         string          `json:"primary_ip,omitempty"`
	Architecture      string          `json:"architecture"`
	Distribution      string          `json:"distribution"`
	KernelVersion     string          `json:"kernel_version"`
	AgentVersion      string          `json:"agent_version"`
	FirewallBackend   string          `json:"firewall_backend"`
	FirewallOwner     string          `json:"firewall_owner"`
	ConnectionState   string          `json:"connection_state"`
	HealthState       string          `json:"health_state"`
	LastSeenAt        *time.Time      `json:"last_seen_at,omitempty"`
	CertificateExpiry *time.Time      `json:"certificate_expiry,omitempty"`
	Labels            []string        `json:"labels,omitempty"`
	Groups            []string        `json:"groups,omitempty"`
	Site              string          `json:"site,omitempty"`
	Sequence          int64           `json:"sequence"`
	StateHash         string          `json:"state_hash,omitempty"`
	PolicyRevision    int64           `json:"policy_revision"`
	Status            json.RawMessage `json:"status,omitempty"`
	Rules             json.RawMessage `json:"rules,omitempty"`
	Capabilities      json.RawMessage `json:"capabilities,omitempty"`
}

type AgentState struct {
	Sequence        int64           `json:"sequence"`
	StateHash       string          `json:"state_hash"`
	PolicyRevision  int64           `json:"policy_revision"`
	HealthState     string          `json:"health_state"`
	FirewallBackend string          `json:"firewall_backend"`
	FirewallOwner   string          `json:"firewall_owner"`
	Status          json.RawMessage `json:"status,omitempty"`
	Rules           json.RawMessage `json:"rules,omitempty"`
	Capabilities    json.RawMessage `json:"capabilities,omitempty"`
}

type AgentCommand struct {
	CommandID string          `json:"command_id"`
	AgentID   string          `json:"agent_id"`
	Op        string          `json:"op"`
	Params    json.RawMessage `json:"params,omitempty"`
	State     string          `json:"state"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     string          `json:"error,omitempty"`
}

type FleetRollout struct {
	RolloutID        string `json:"rollout_id"`
	TargetGroup      string `json:"target_group"`
	Op               string `json:"op"`
	BatchSize        int    `json:"batch_size"`
	MaxFailures      int    `json:"max_failures"`
	State            string `json:"state"`
	TotalTargets     int    `json:"total_targets"`
	CompletedTargets int    `json:"completed_targets"`
	FailedTargets    int    `json:"failed_targets"`
}

func tokenHash(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func (r *FleetRepo) CreateEnrollmentToken(agentID string, ttl time.Duration) (string, time.Time, error) {
	raw := uuid.NewString() + uuid.NewString()
	expires := time.Now().UTC().Add(ttl)
	_, err := r.db.SQL().Exec(`INSERT INTO enrollment_tokens(token_hash, agent_id, created_at, expires_at) VALUES(?,?,?,?)`, tokenHash(raw), agentID, time.Now().UTC().Format(time.RFC3339Nano), expires.Format(time.RFC3339Nano))
	return raw, expires, err
}

func (r *FleetRepo) Enroll(rawToken string, a AgentRecord, credential string) error {
	tx, err := r.db.SQL().Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var wanted sql.NullString
	var expires string
	var used sql.NullString
	err = tx.QueryRow(`SELECT agent_id, expires_at, used_at FROM enrollment_tokens WHERE token_hash=?`, tokenHash(rawToken)).Scan(&wanted, &expires, &used)
	if err != nil || used.Valid || parseTime(expires).Before(time.Now().UTC()) {
		return fmt.Errorf("invalid or expired enrollment token")
	}
	if wanted.Valid && wanted.String != "" && wanted.String != a.AgentID {
		return fmt.Errorf("enrollment token belongs to another agent")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	labels, _ := json.Marshal(a.Labels)
	groups, _ := json.Marshal(a.Groups)
	_, err = tx.Exec(`INSERT INTO agents(agent_id,display_name,hostname,architecture,distribution,kernel_version,agent_version,connection_state,health_state,last_seen_at,labels,groups,site,created_at,updated_at,credential_hash)
		VALUES(?,?,?,?,?,?,?,'ONLINE','UNKNOWN',?,?,?,?,?,?,?)
		ON CONFLICT(agent_id) DO UPDATE SET display_name=excluded.display_name,hostname=excluded.hostname,architecture=excluded.architecture,distribution=excluded.distribution,kernel_version=excluded.kernel_version,agent_version=excluded.agent_version,connection_state='ONLINE',last_seen_at=excluded.last_seen_at,labels=excluded.labels,groups=excluded.groups,site=excluded.site,updated_at=excluded.updated_at,credential_hash=excluded.credential_hash,sequence=0,state_hash=NULL,status_json=NULL,rules_json=NULL,capabilities_json=NULL`,
		a.AgentID, a.DisplayName, a.Hostname, a.Architecture, a.Distribution, a.KernelVersion, a.AgentVersion, now, string(labels), string(groups), a.Site, now, now, tokenHash(credential))
	if err != nil {
		return err
	}
	if _, err = tx.Exec(`UPDATE enrollment_tokens SET used_at=?, used_by=? WHERE token_hash=?`, now, a.AgentID, tokenHash(rawToken)); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *FleetRepo) Authenticate(agentID, credential string) bool {
	var stored string
	if r.db.SQL().QueryRow(`SELECT credential_hash FROM agents WHERE agent_id=?`, agentID).Scan(&stored) != nil {
		return false
	}
	a, b := []byte(stored), []byte(tokenHash(credential))
	return len(a) == len(b) && subtle.ConstantTimeCompare(a, b) == 1
}

func (r *FleetRepo) SetAgentCertificate(agentID, fingerprint string, expiry time.Time) error {
	_, err := r.db.SQL().Exec(`UPDATE agents SET certificate_fingerprint=?,certificate_expiry=? WHERE agent_id=?`, fingerprint, expiry.UTC().Format(time.RFC3339Nano), agentID)
	return err
}
func (r *FleetRepo) RotateAgentCertificate(agentID, fingerprint string, expiry time.Time) error {
	var current sql.NullString
	if err := r.db.SQL().QueryRow(`SELECT certificate_fingerprint FROM agents WHERE agent_id=?`, agentID).Scan(&current); err != nil {
		return err
	}
	combined := fingerprint
	if current.Valid && current.String != "" {
		parts := strings.Split(current.String, ",")
		combined = parts[len(parts)-1] + "," + fingerprint
	}
	return r.SetAgentCertificate(agentID, combined, expiry)
}
func (r *FleetRepo) CertificateMatches(agentID, fingerprint string) bool {
	var stored sql.NullString
	var state string
	if r.db.SQL().QueryRow(`SELECT certificate_fingerprint,connection_state FROM agents WHERE agent_id=?`, agentID).Scan(&stored, &state) != nil {
		return false
	}
	if state == "REVOKED" || !stored.Valid {
		return false
	}
	parts := strings.Split(stored.String, ",")
	for i, candidate := range parts {
		if candidate == fingerprint {
			// First successful request with rotated certificate retires overlap.
			if len(parts) > 1 && i == len(parts)-1 {
				_, _ = r.db.SQL().Exec(`UPDATE agents SET certificate_fingerprint=? WHERE agent_id=?`, fingerprint, agentID)
			}
			return true
		}
	}
	return false
}
func (r *FleetRepo) RevokeAgent(agentID string) error {
	res, err := r.db.SQL().Exec(`UPDATE agents SET connection_state='REVOKED',credential_hash=NULL,certificate_fingerprint=NULL WHERE agent_id=?`, agentID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *FleetRepo) UpdateState(agentID string, s AgentState) (bool, error) {
	var previousHash string
	var previousRevision int64
	_ = r.db.SQL().QueryRow(`SELECT COALESCE(state_hash,''), policy_revision FROM agents WHERE agent_id=?`, agentID).Scan(&previousHash, &previousRevision)
	now := time.Now().UTC().Format(time.RFC3339Nano)
	res, err := r.db.SQL().Exec(`UPDATE agents SET connection_state='ONLINE',health_state=?,last_seen_at=?,firewall_backend=?,firewall_owner=?,sequence=?,state_hash=?,policy_revision=?,status_json=?,rules_json=?,capabilities_json=?,updated_at=? WHERE agent_id=? AND sequence<=?`, s.HealthState, now, s.FirewallBackend, s.FirewallOwner, s.Sequence, s.StateHash, s.PolicyRevision, nullableJSON(s.Status), nullableJSON(s.Rules), nullableJSON(s.Capabilities), now, agentID, s.Sequence)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return false, fmt.Errorf("stale agent sequence")
	}
	drift := previousHash != "" && previousHash != s.StateHash && previousRevision == s.PolicyRevision
	return drift, nil
}

func nullableJSON(v json.RawMessage) any {
	if len(v) == 0 {
		return nil
	}
	return string(v)
}

func (r *FleetRepo) ListAgents() ([]AgentRecord, error) {
	rows, err := r.db.SQL().Query(`SELECT agent_id,COALESCE(display_name,''),COALESCE(hostname,''),COALESCE(primary_ip,''),COALESCE(architecture,''),COALESCE(distribution,''),COALESCE(kernel_version,''),COALESCE(agent_version,''),COALESCE(firewall_backend,''),COALESCE(firewall_owner,''),connection_state,health_state,last_seen_at,certificate_expiry,COALESCE(labels,'[]'),COALESCE(groups,'[]'),COALESCE(site,''),sequence,COALESCE(state_hash,''),policy_revision,status_json,rules_json,capabilities_json FROM agents ORDER BY display_name,hostname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AgentRecord
	for rows.Next() {
		var a AgentRecord
		var seen, certExpiry, status, rules, caps sql.NullString
		var labels, groups string
		if err := rows.Scan(&a.AgentID, &a.DisplayName, &a.Hostname, &a.PrimaryIP, &a.Architecture, &a.Distribution, &a.KernelVersion, &a.AgentVersion, &a.FirewallBackend, &a.FirewallOwner, &a.ConnectionState, &a.HealthState, &seen, &certExpiry, &labels, &groups, &a.Site, &a.Sequence, &a.StateHash, &a.PolicyRevision, &status, &rules, &caps); err != nil {
			return nil, err
		}
		if certExpiry.Valid {
			t := parseTime(certExpiry.String)
			a.CertificateExpiry = &t
		}
		if seen.Valid {
			t := parseTime(seen.String)
			a.LastSeenAt = &t
			if time.Since(t) > 45*time.Second {
				a.ConnectionState = "OFFLINE"
			}
		}
		_ = json.Unmarshal([]byte(labels), &a.Labels)
		_ = json.Unmarshal([]byte(groups), &a.Groups)
		if status.Valid {
			a.Status = json.RawMessage(status.String)
		}
		if rules.Valid {
			a.Rules = json.RawMessage(rules.String)
		}
		if caps.Valid {
			a.Capabilities = json.RawMessage(caps.String)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *FleetRepo) GetAgent(agentID string) (AgentRecord, error) {
	agents, err := r.ListAgents()
	if err != nil {
		return AgentRecord{}, err
	}
	for _, agent := range agents {
		if agent.AgentID == agentID {
			return agent, nil
		}
	}
	return AgentRecord{}, sql.ErrNoRows
}

func (r *FleetRepo) QueueCommand(agentID, op string, params json.RawMessage) (AgentCommand, error) {
	c := AgentCommand{CommandID: uuid.NewString(), AgentID: agentID, Op: op, Params: params, State: "QUEUED"}
	_, err := r.db.SQL().Exec(`INSERT INTO agent_commands(command_id,agent_id,op,params_json,state,created_at) VALUES(?,?,?,?,?,?)`, c.CommandID, c.AgentID, c.Op, nullableJSON(c.Params), c.State, time.Now().UTC().Format(time.RFC3339Nano))
	return c, err
}

func (r *FleetRepo) NextCommand(agentID string) (*AgentCommand, error) {
	tx, err := r.db.SQL().Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	_, _ = tx.Exec(`UPDATE agent_commands SET state='QUEUED', delivered_at=NULL WHERE agent_id=? AND state='DELIVERED' AND delivered_at<?`, agentID, time.Now().UTC().Add(-2*time.Minute).Format(time.RFC3339Nano))
	var c AgentCommand
	var p sql.NullString
	err = tx.QueryRow(`SELECT command_id,op,params_json FROM agent_commands WHERE agent_id=? AND state='QUEUED' ORDER BY created_at LIMIT 1`, agentID).Scan(&c.CommandID, &c.Op, &p)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.AgentID = agentID
	c.State = "DELIVERED"
	if p.Valid {
		c.Params = json.RawMessage(p.String)
	}
	_, err = tx.Exec(`UPDATE agent_commands SET state='DELIVERED',delivered_at=? WHERE command_id=? AND state='QUEUED'`, time.Now().UTC().Format(time.RFC3339Nano), c.CommandID)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *FleetRepo) CompleteCommand(agentID, id string, result json.RawMessage, msg string) error {
	_, err := r.db.SQL().Exec(`UPDATE agent_commands SET state=?,result_json=?,error=?,completed_at=? WHERE command_id=? AND agent_id=?`, map[bool]string{true: "FAILED", false: "COMPLETED"}[msg != ""], nullableJSON(result), msg, time.Now().UTC().Format(time.RFC3339Nano), id, agentID)
	if err == nil {
		_ = r.advanceRollout(id, msg != "")
	}
	return err
}

func (r *FleetRepo) GetCommand(id string) (AgentCommand, error) {
	var c AgentCommand
	var p, res, er sql.NullString
	err := r.db.SQL().QueryRow(`SELECT command_id,agent_id,op,params_json,state,result_json,error FROM agent_commands WHERE command_id=?`, id).Scan(&c.CommandID, &c.AgentID, &c.Op, &p, &c.State, &res, &er)
	if p.Valid {
		c.Params = json.RawMessage(p.String)
	}
	if res.Valid {
		c.Result = json.RawMessage(res.String)
	}
	c.Error = er.String
	return c, err
}

func (r *FleetRepo) CreateRollout(group, op string, params json.RawMessage, batchSize, maxFailures int) (FleetRollout, error) {
	agents, err := r.ListAgents()
	if err != nil {
		return FleetRollout{}, err
	}
	var targets []string
	for _, a := range agents {
		for _, g := range a.Groups {
			if g == group {
				targets = append(targets, a.AgentID)
				break
			}
		}
	}
	if len(targets) == 0 {
		return FleetRollout{}, fmt.Errorf("no agents in group %q", group)
	}
	if batchSize < 1 {
		batchSize = 1
	}
	if batchSize > len(targets) {
		batchSize = len(targets)
	}
	if maxFailures < 0 {
		maxFailures = 0
	}
	rollout := FleetRollout{RolloutID: uuid.NewString(), TargetGroup: group, Op: op, BatchSize: batchSize, MaxFailures: maxFailures, State: "RUNNING", TotalTargets: len(targets)}
	tx, err := r.db.SQL().Begin()
	if err != nil {
		return rollout, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO fleet_rollouts(rollout_id,target_group,op,params_json,batch_size,max_failures,state,total_targets,created_at) VALUES(?,?,?,?,?,?,?,?,?)`, rollout.RolloutID, group, op, nullableJSON(params), batchSize, maxFailures, rollout.State, len(targets), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return rollout, err
	}
	for i, id := range targets {
		if _, err = tx.Exec(`INSERT INTO fleet_rollout_targets(rollout_id,agent_id,ordinal) VALUES(?,?,?)`, rollout.RolloutID, id, i); err != nil {
			return rollout, err
		}
	}
	if err = tx.Commit(); err != nil {
		return rollout, err
	}
	err = r.queueRolloutBatch(rollout.RolloutID)
	return rollout, err
}

func (r *FleetRepo) queueRolloutBatch(rolloutID string) error {
	var op string
	var params sql.NullString
	var batch int
	if err := r.db.SQL().QueryRow(`SELECT op,params_json,batch_size FROM fleet_rollouts WHERE rollout_id=? AND state='RUNNING'`, rolloutID).Scan(&op, &params, &batch); err != nil {
		return err
	}
	rows, err := r.db.SQL().Query(`SELECT agent_id FROM fleet_rollout_targets WHERE rollout_id=? AND state='PENDING' ORDER BY ordinal LIMIT ?`, rolloutID, batch)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	rows.Close()
	for _, id := range ids {
		cmd, err := r.QueueCommand(id, op, json.RawMessage(params.String))
		if err != nil {
			return err
		}
		if _, err = r.db.SQL().Exec(`UPDATE fleet_rollout_targets SET command_id=?,state='ACTIVE' WHERE rollout_id=? AND agent_id=?`, cmd.CommandID, rolloutID, id); err != nil {
			return err
		}
	}
	return nil
}

func (r *FleetRepo) advanceRollout(commandID string, failed bool) error {
	var rolloutID string
	err := r.db.SQL().QueryRow(`SELECT rollout_id FROM fleet_rollout_targets WHERE command_id=?`, commandID).Scan(&rolloutID)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	state := "COMPLETED"
	if failed {
		state = "FAILED"
	}
	if _, err = r.db.SQL().Exec(`UPDATE fleet_rollout_targets SET state=? WHERE command_id=?`, state, commandID); err != nil {
		return err
	}
	var active, pending, complete, fail, maxFail int
	if err = r.db.SQL().QueryRow(`SELECT SUM(CASE WHEN state='ACTIVE' THEN 1 ELSE 0 END),SUM(CASE WHEN state='PENDING' THEN 1 ELSE 0 END),SUM(CASE WHEN state='COMPLETED' THEN 1 ELSE 0 END),SUM(CASE WHEN state='FAILED' THEN 1 ELSE 0 END) FROM fleet_rollout_targets WHERE rollout_id=?`, rolloutID).Scan(&active, &pending, &complete, &fail); err != nil {
		return err
	}
	if err = r.db.SQL().QueryRow(`SELECT max_failures FROM fleet_rollouts WHERE rollout_id=?`, rolloutID).Scan(&maxFail); err != nil {
		return err
	}
	if fail > maxFail {
		_, err = r.db.SQL().Exec(`UPDATE fleet_rollouts SET state='FAILED',completed_targets=?,failed_targets=?,completed_at=? WHERE rollout_id=?`, complete, fail, time.Now().UTC().Format(time.RFC3339Nano), rolloutID)
		return err
	}
	if active == 0 && pending == 0 {
		_, err = r.db.SQL().Exec(`UPDATE fleet_rollouts SET state='COMPLETED',completed_targets=?,failed_targets=?,completed_at=? WHERE rollout_id=?`, complete, fail, time.Now().UTC().Format(time.RFC3339Nano), rolloutID)
		return err
	}
	_, _ = r.db.SQL().Exec(`UPDATE fleet_rollouts SET completed_targets=?,failed_targets=? WHERE rollout_id=?`, complete, fail, rolloutID)
	if active == 0 {
		return r.queueRolloutBatch(rolloutID)
	}
	return nil
}

func (r *FleetRepo) GetRollout(id string) (FleetRollout, error) {
	var v FleetRollout
	err := r.db.SQL().QueryRow(`SELECT rollout_id,target_group,op,batch_size,max_failures,state,total_targets,completed_targets,failed_targets FROM fleet_rollouts WHERE rollout_id=?`, id).Scan(&v.RolloutID, &v.TargetGroup, &v.Op, &v.BatchSize, &v.MaxFailures, &v.State, &v.TotalTargets, &v.CompletedTargets, &v.FailedTargets)
	return v, err
}
