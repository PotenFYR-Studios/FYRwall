package database

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFleetEnrollmentStateAndCommands(t *testing.T) {
	db := testDB(t)
	repo := NewFleetRepo(db)
	token, _, err := repo.CreateEnrollmentToken("agent-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Enroll(token, AgentRecord{AgentID: "agent-1", Hostname: "host-1", DisplayName: "Host 1", Groups: []string{"blue"}}, "secret"); err != nil {
		t.Fatal(err)
	}
	if repo.Authenticate("agent-1", "wrong") {
		t.Fatal("wrong credential authenticated")
	}
	if !repo.Authenticate("agent-1", "secret") {
		t.Fatal("valid credential rejected")
	}
	if err := repo.Enroll(token, AgentRecord{AgentID: "agent-1", Hostname: "host-1"}, "other"); err == nil {
		t.Fatal("single-use token reused")
	}

	state := AgentState{Sequence: 1, StateHash: "abc", PolicyRevision: 2, HealthState: "HEALTHY", FirewallBackend: "iptables", FirewallOwner: "IPTABLES_NFT", Status: json.RawMessage(`{"enabled":true}`), Rules: json.RawMessage(`{"rules":[]}`)}
	if _, err := repo.UpdateState("agent-1", state); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.UpdateState("agent-1", AgentState{Sequence: 0}); err == nil {
		t.Fatal("stale sequence accepted")
	}
	drift, err := repo.UpdateState("agent-1", AgentState{Sequence: 2, StateHash: "changed", PolicyRevision: 2})
	if err != nil || !drift {
		t.Fatalf("expected drift, got %v, %v", drift, err)
	}
	agent, err := repo.GetAgent("agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if agent.StateHash != "changed" || agent.PolicyRevision != 2 {
		t.Fatalf("unexpected state: %+v", agent)
	}

	cmd, err := repo.QueueCommand("agent-1", "GetFirewallStatus", nil)
	if err != nil {
		t.Fatal(err)
	}
	next, err := repo.NextCommand("agent-1")
	if err != nil {
		t.Fatal(err)
	}
	if next == nil || next.CommandID != cmd.CommandID {
		t.Fatalf("unexpected command: %+v", next)
	}
	if err := repo.CompleteCommand("agent-1", cmd.CommandID, json.RawMessage(`{"ok":true}`), ""); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetCommand(cmd.CommandID)
	if err != nil {
		t.Fatal(err)
	}
	if got.State != "COMPLETED" {
		t.Fatalf("unexpected command state %s", got.State)
	}

	token2, _, err := repo.CreateEnrollmentToken("agent-2", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.Enroll(token2, AgentRecord{AgentID: "agent-2", Hostname: "host-2", DisplayName: "Host 2", Groups: []string{"blue"}}, "secret-2"); err != nil {
		t.Fatal(err)
	}
	rollout, err := repo.CreateRollout("blue", "Reload", nil, 1, 0)
	if err != nil {
		t.Fatal(err)
	}
	first, err := repo.NextCommand("agent-1")
	if err != nil || first == nil {
		t.Fatalf("first rollout command: %+v %v", first, err)
	}
	if err = repo.CompleteCommand("agent-1", first.CommandID, nil, ""); err != nil {
		t.Fatal(err)
	}
	second, err := repo.NextCommand("agent-2")
	if err != nil || second == nil {
		t.Fatalf("second rollout command: %+v %v", second, err)
	}
	if err = repo.CompleteCommand("agent-2", second.CommandID, nil, "failed"); err != nil {
		t.Fatal(err)
	}
	rollout, err = repo.GetRollout(rollout.RolloutID)
	if err != nil {
		t.Fatal(err)
	}
	if rollout.State != "FAILED" || rollout.CompletedTargets != 1 || rollout.FailedTargets != 1 {
		t.Fatalf("unexpected rollout: %+v", rollout)
	}
}
