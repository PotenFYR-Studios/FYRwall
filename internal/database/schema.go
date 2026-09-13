package database

// Schema returns the ordered migration list for the full v1 schema:
// users, sessions, settings, audit_log, notifications, firewall rule
// metadata, restore points, templates (spec sections 3.3, 99).
func Schema() []Migration {
	return []Migration{
		{Version: 1, Name: "base", SQL: `
CREATE TABLE users (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL,
	enabled INTEGER NOT NULL DEFAULT 1,
	must_change_password INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	last_login_at TEXT
);
CREATE INDEX idx_users_username ON users(username);

CREATE TABLE settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	updated_by TEXT
);

CREATE TABLE audit_log (
	id TEXT PRIMARY KEY,
	ts TEXT NOT NULL,
	actor TEXT NOT NULL,
	source_ip TEXT,
	action TEXT NOT NULL,
	target TEXT,
	before_summary TEXT,
	after_summary TEXT,
	success INTEGER NOT NULL,
	request_id TEXT,
	detail TEXT
);
CREATE INDEX idx_audit_ts ON audit_log(ts);
CREATE INDEX idx_audit_actor ON audit_log(actor);

CREATE TABLE notifications (
	id TEXT PRIMARY KEY,
	code TEXT NOT NULL,
	severity TEXT NOT NULL,
	category TEXT NOT NULL,
	component TEXT NOT NULL,
	summary TEXT NOT NULL,
	details TEXT,
	remediation TEXT,
	fingerprint TEXT NOT NULL,
	first_seen_at TEXT NOT NULL,
	last_seen_at TEXT NOT NULL,
	occurrences INTEGER NOT NULL DEFAULT 1,
	resolved INTEGER NOT NULL DEFAULT 0,
	resolved_at TEXT,
	read INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX idx_notifications_fingerprint ON notifications(fingerprint);
CREATE INDEX idx_notifications_unresolved ON notifications(resolved, severity);

CREATE TABLE rules (
	id TEXT PRIMARY KEY,
	backend_id TEXT,
	family TEXT NOT NULL,
	direction TEXT NOT NULL,
	action TEXT NOT NULL,
	protocol TEXT NOT NULL,
	source TEXT,
	source_port TEXT,
	destination TEXT,
	destination_port TEXT,
	interface_in TEXT,
	interface_out TEXT,
	state TEXT,
	comment TEXT,
	enabled INTEGER NOT NULL DEFAULT 1,
	priority INTEGER NOT NULL DEFAULT 0,
	backend TEXT NOT NULL,
	raw_reference TEXT,
	created_by TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
CREATE INDEX idx_rules_priority ON rules(priority);

CREATE TABLE restore_points (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	reason TEXT,
	backend TEXT NOT NULL,
	backend_version TEXT,
	iptables_save TEXT,
	ip6tables_save TEXT,
	ufw_status TEXT,
	state_hash TEXT,
	app_version TEXT,
	host_id TEXT,
	created_by TEXT,
	notes TEXT,
	created_at TEXT NOT NULL
);
CREATE INDEX idx_restore_created ON restore_points(created_at);

CREATE TABLE rule_templates (
	id TEXT PRIMARY KEY,
	name TEXT NOT NULL UNIQUE,
	description TEXT,
	definition TEXT NOT NULL,
	builtin INTEGER NOT NULL DEFAULT 0,
	version INTEGER NOT NULL DEFAULT 1,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);
`},
		{Version: 2, Name: "fleet", SQL: `
-- Central fleet management tables (V2 override, spec section 99).
CREATE TABLE agents (
	agent_id TEXT PRIMARY KEY,
	host_id TEXT,
	display_name TEXT,
	hostname TEXT,
	primary_ip TEXT,
	architecture TEXT,
	distribution TEXT,
	kernel_version TEXT,
	agent_version TEXT,
	firewall_backend TEXT,
	firewall_owner TEXT,
	connection_state TEXT NOT NULL DEFAULT 'UNENROLLED',
	health_state TEXT NOT NULL DEFAULT 'UNKNOWN',
	certificate_fingerprint TEXT,
	certificate_expiry TEXT,
	last_seen_at TEXT,
	labels TEXT,
	groups TEXT,
	site TEXT,
	notes TEXT,
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL
);

CREATE TABLE agent_events (
	event_id TEXT PRIMARY KEY,
	agent_id TEXT NOT NULL,
	observed_at TEXT NOT NULL,
	sent_at TEXT,
	received_at TEXT NOT NULL,
	source TEXT,
	sequence INTEGER,
	raw_message TEXT,
	action TEXT,
	protocol TEXT,
	src_addr TEXT,
	dst_addr TEXT,
	src_port INTEGER,
	dst_port INTEGER,
	interface_in TEXT,
	interface_out TEXT,
	policy_id TEXT,
	rule_id TEXT,
	correlation_confidence TEXT
);
CREATE INDEX idx_agent_events_agent_time ON agent_events(agent_id, observed_at);

CREATE TABLE enrollment_tokens (
	token_hash TEXT PRIMARY KEY,
	agent_id TEXT,
	created_at TEXT NOT NULL,
	expires_at TEXT NOT NULL,
	used_at TEXT,
	used_by TEXT
);
`},
		{Version: 3, Name: "fleet_sync", SQL: `
ALTER TABLE agents ADD COLUMN credential_hash TEXT;
ALTER TABLE agents ADD COLUMN sequence INTEGER NOT NULL DEFAULT 0;
ALTER TABLE agents ADD COLUMN state_hash TEXT;
ALTER TABLE agents ADD COLUMN policy_revision INTEGER NOT NULL DEFAULT 0;
ALTER TABLE agents ADD COLUMN status_json TEXT;
ALTER TABLE agents ADD COLUMN rules_json TEXT;
ALTER TABLE agents ADD COLUMN capabilities_json TEXT;

CREATE TABLE agent_commands (
	command_id TEXT PRIMARY KEY,
	agent_id TEXT NOT NULL,
	op TEXT NOT NULL,
	params_json TEXT,
	state TEXT NOT NULL DEFAULT 'QUEUED',
	result_json TEXT,
	error TEXT,
	created_at TEXT NOT NULL,
	delivered_at TEXT,
	completed_at TEXT,
	FOREIGN KEY(agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE
);
CREATE INDEX idx_agent_commands_pending ON agent_commands(agent_id, state, created_at);
`},
		{Version: 4, Name: "fleet_rollouts", SQL: `
CREATE TABLE fleet_rollouts (
	rollout_id TEXT PRIMARY KEY,
	target_group TEXT NOT NULL,
	op TEXT NOT NULL,
	params_json TEXT,
	batch_size INTEGER NOT NULL,
	max_failures INTEGER NOT NULL,
	state TEXT NOT NULL DEFAULT 'RUNNING',
	total_targets INTEGER NOT NULL,
	completed_targets INTEGER NOT NULL DEFAULT 0,
	failed_targets INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL,
	completed_at TEXT
);
CREATE TABLE fleet_rollout_targets (
	rollout_id TEXT NOT NULL,
	agent_id TEXT NOT NULL,
	ordinal INTEGER NOT NULL,
	command_id TEXT,
	state TEXT NOT NULL DEFAULT 'PENDING',
	PRIMARY KEY(rollout_id, agent_id),
	FOREIGN KEY(rollout_id) REFERENCES fleet_rollouts(rollout_id) ON DELETE CASCADE,
	FOREIGN KEY(agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE
);
CREATE INDEX idx_rollout_targets_state ON fleet_rollout_targets(rollout_id, state, ordinal);
`},
	}
}
