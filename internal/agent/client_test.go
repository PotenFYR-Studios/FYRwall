package agent

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/PotenFYR-Studios/FYRwall/internal/firewall"
)

type testBackend struct{ restored bool }

func (b *testBackend) Name() string { return "test" }
func (b *testBackend) Detect(context.Context) firewall.DetectionResult {
	return firewall.DetectionResult{Name: "test", Found: true, Active: true}
}
func (b *testBackend) Status(context.Context) (firewall.FirewallStatus, error) {
	return firewall.FirewallStatus{Backend: "test", Enabled: true}, nil
}
func (b *testBackend) ListRules(context.Context) ([]firewall.Rule, error) {
	return []firewall.Rule{{ID: "r1"}}, nil
}
func (b *testBackend) Validate(context.Context, firewall.Transaction) firewall.ValidationResult {
	return firewall.ValidationResult{Valid: true}
}
func (b *testBackend) Snapshot(context.Context, string) (firewall.Snapshot, error) {
	return firewall.Snapshot{ID: "s1", StateHash: "hash"}, nil
}
func (b *testBackend) Apply(context.Context, firewall.Transaction) (firewall.ApplyResult, error) {
	return firewall.ApplyResult{TxnID: "t1", StateHash: "hash"}, nil
}
func (b *testBackend) Verify(_ context.Context, e firewall.StateHash) (firewall.VerifyResult, error) {
	return firewall.VerifyResult{Match: string(e) == "hash", Observed: "hash", Expected: string(e)}, nil
}
func (b *testBackend) Restore(context.Context, firewall.Snapshot) error {
	b.restored = true
	return nil
}
func (b *testBackend) Reload(context.Context) error  { return nil }
func (b *testBackend) Restart(context.Context) error { return nil }

func TestUnixClientRoutesTypedOperations(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	backend := &testBackend{}
	mgr := firewall.NewManager(backend, nil, nil)
	mgr.SetOwnership(firewall.OwnershipReport{Owner: firewall.OwnerIptablesNFT, WritesAllowed: true})
	srv := NewServer(mgr, zerolog.Nop())
	path := filepath.Join(t.TempDir(), "agent.sock")
	if err := srv.Listen(path); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx) }()
	client := NewClient(path)

	cap, err := client.Capabilities(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if cap.Ownership.Owner != firewall.OwnerIptablesNFT {
		t.Fatalf("unexpected owner %s", cap.Ownership.Owner)
	}
	status, err := client.Status(ctx)
	if err != nil || !status.Enabled {
		t.Fatalf("status: %+v %v", status, err)
	}
	rules, err := client.ListRules(ctx)
	if err != nil || len(rules) != 1 {
		t.Fatalf("rules: %+v %v", rules, err)
	}
	verify, err := client.Verify(ctx, "hash")
	if err != nil || !verify.Match {
		t.Fatalf("verify: %+v %v", verify, err)
	}
	if err := client.Restore(ctx, firewall.Snapshot{ID: "s1"}); err != nil {
		t.Fatal(err)
	}
	if !backend.restored {
		t.Fatal("restore not routed")
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("server did not stop")
	}
}
