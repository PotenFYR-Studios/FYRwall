package database

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/auth"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := OpenSQLite(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := db.Migrate(Schema()); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSQLiteOpenAndMigrate(t *testing.T) {
	db := testDB(t)
	if err := db.IntegCheck(); err != nil {
		t.Fatalf("integrity: %v", err)
	}
	var n int
	if err := db.SQL().QueryRow("SELECT COUNT(*) FROM schema_migrations").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != len(Schema()) {
		t.Fatalf("expected %d migrations recorded, got %d", len(Schema()), n)
	}
}

func TestUserCRUD(t *testing.T) {
	db := testDB(t)
	users := NewUserRepo(db)

	hash, _ := auth.HashPassword("a-very-long-password-123", auth.DefaultArgonParams())
	u, err := users.Create("admin", hash, auth.RoleSuperAdmin, false)
	if err != nil {
		t.Fatal(err)
	}
	if u.Username != "admin" || u.Role != auth.RoleSuperAdmin {
		t.Fatalf("unexpected user %+v", u)
	}

	got, err := users.Get("admin")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != u.ID {
		t.Error("ID mismatch on Get")
	}

	n, err := users.CountAdmins()
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected 1 admin, got %d", n)
	}

	// Disabled admin does not count.
	users.SetEnabled(u.ID, false)
	n, _ = users.CountAdmins()
	if n != 0 {
		t.Fatalf("disabled admin must not count, got %d", n)
	}

	// Invalid role rejected.
	if _, err := users.Create("x", hash, "backdoor", false); err == nil {
		t.Fatal("invalid role must be rejected")
	}

	users.Delete(u.ID)
	if _, err := users.Get("admin"); err == nil {
		t.Fatal("deleted user must not be found")
	}
}

func TestNotificationDedup(t *testing.T) {
	db := testDB(t)
	repo := NewNotificationRepo(db)
	fp := "FW_TEST|firewall|host"

	for i := 0; i < 3; i++ {
		if err := repo.Upsert(Notification{
			Code: "FW_TEST", Severity: "warning", Category: "firewall",
			Component: "firewall", Summary: "test issue", Fingerprint: fp,
		}); err != nil {
			t.Fatal(err)
		}
	}
	list, err := repo.ListUnresolved()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("expected exactly 1 notification after 3 upserts, got %d", len(list))
	}
	if list[0].Occurrences != 3 {
		t.Fatalf("expected occurrences=3, got %d", list[0].Occurrences)
	}

	if err := repo.Resolve(fp); err != nil {
		t.Fatal(err)
	}
	list, _ = repo.ListUnresolved()
	if len(list) != 0 {
		t.Fatalf("expected 0 unresolved after resolve, got %d", len(list))
	}
}

func TestAuditTrail(t *testing.T) {
	db := testDB(t)
	repo := NewAuditRepo(db)
	for i := 0; i < 5; i++ {
		if err := repo.Insert(AuditEntry{
			Actor: "admin", Action: "firewall.apply", Target: "tx",
			Success: true, Detail: "ok",
		}); err != nil {
			t.Fatal(err)
		}
	}
	entries, err := repo.List(3, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected limit 3, got %d", len(entries))
	}
}

func TestMigrationsIdempotent(t *testing.T) {
	db := testDB(t)
	// Applying again must be a no-op.
	if err := db.Migrate(Schema()); err != nil {
		t.Fatalf("re-migrate must not fail: %v", err)
	}
	_ = time.Now
}
