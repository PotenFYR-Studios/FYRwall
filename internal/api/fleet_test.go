package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/config"
	"github.com/PotenFYR-Studios/FYRwall/internal/database"
	"github.com/PotenFYR-Studios/FYRwall/internal/diagnostics"
	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
	"github.com/PotenFYR-Studios/FYRwall/internal/health"
)

func TestAgentEnrollmentPollAndResult(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default()
	cfg.Database.SQLitePath = filepath.Join(dir, "fleet.db")
	db, err := database.OpenSQLite(cfg.Database.SQLitePath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = db.Migrate(database.Schema()); err != nil {
		t.Fatal(err)
	}
	mgr := firewall.NewManager(stubBackend{}, nil, nil)
	mgr.SetOwnership(firewall.OwnershipReport{Owner: firewall.OwnerUFW, WritesAllowed: true})
	server := New(cfg, testLogger(), db, mgr, health.NewTracker(), diagnostics.NewRegistry())
	ts := httptest.NewServer(server.Router())
	defer ts.Close()
	repo := database.NewFleetRepo(db)
	token, _, err := repo.CreateEnrollmentToken("agent-e2e", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	enrollBody, _ := json.Marshal(map[string]any{"token": token, "agent": map[string]any{"agent_id": "agent-e2e", "hostname": "target", "display_name": "Target"}})
	resp, err := http.Post(ts.URL+"/api/v1/agent/enroll", "application/json", bytes.NewReader(enrollBody))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("enroll HTTP %d", resp.StatusCode)
	}
	var identity struct {
		Credential string `json:"credential"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&identity); err != nil {
		t.Fatal(err)
	}
	queued, err := repo.QueueCommand("agent-e2e", "GetFirewallStatus", nil)
	if err != nil {
		t.Fatal(err)
	}
	state, _ := json.Marshal(database.AgentState{Sequence: 1, StateHash: "state-1", PolicyRevision: 0, HealthState: "HEALTHY"})
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, ts.URL+"/api/v1/agent/poll", bytes.NewReader(state))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-FYRwall-Agent-ID", "agent-e2e")
	req.Header.Set("Authorization", "Bearer "+identity.Credential)
	poll, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer poll.Body.Close()
	if poll.StatusCode != http.StatusOK {
		t.Fatalf("poll HTTP %d", poll.StatusCode)
	}
	var command database.AgentCommand
	if err = json.NewDecoder(poll.Body).Decode(&command); err != nil {
		t.Fatal(err)
	}
	if command.CommandID != queued.CommandID {
		t.Fatalf("wrong command %s", command.CommandID)
	}
	result, _ := json.Marshal(map[string]any{"command_id": command.CommandID, "result": map[string]any{"enabled": true}})
	resultReq, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/agent/result", bytes.NewReader(result))
	resultReq.Header.Set("Content-Type", "application/json")
	resultReq.Header.Set("X-FYRwall-Agent-ID", "agent-e2e")
	resultReq.Header.Set("Authorization", "Bearer "+identity.Credential)
	resultResp, err := http.DefaultClient.Do(resultReq)
	if err != nil {
		t.Fatal(err)
	}
	resultResp.Body.Close()
	if resultResp.StatusCode != http.StatusOK {
		t.Fatalf("result HTTP %d", resultResp.StatusCode)
	}
	done, err := repo.GetCommand(command.CommandID)
	if err != nil || done.State != "COMPLETED" {
		t.Fatalf("command not completed: %+v %v", done, err)
	}
}
