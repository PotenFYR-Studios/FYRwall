package database

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/PotenFYR-Studios/FYRwall/internal/auth"
	"github.com/google/uuid"
)

// UserRepo manages FYRwall application users (spec section 14).
type UserRepo struct{ db *DB }

func NewUserRepo(db *DB) *UserRepo { return &UserRepo{db: db} }

// User is an FYRwall application user, never a Linux account.
type User struct {
	ID             string
	Username       string
	PasswordHash   string
	Role           auth.Role
	Enabled        bool
	MustChangePass bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
	LastLoginAt    *time.Time
}

// Create inserts a user; password must already be Argon2id-hashed.
func (r *UserRepo) Create(username, passwordHash string, role auth.Role, mustChange bool) (*User, error) {
	if !auth.ValidRole(role) {
		return nil, fmt.Errorf("invalid role %q", role)
	}
	now := time.Now().UTC()
	id := uuid.NewString()
	_, err := r.db.SQL().Exec(`INSERT INTO users
		(id, username, password_hash, role, enabled, must_change_password, created_at, updated_at)
		VALUES (?,?,?,?,1,?,?,?)`,
		id, username, passwordHash, string(role), boolInt(mustChange),
		now.Format(time.RFC3339), now.Format(time.RFC3339))
	if err != nil {
		return nil, err
	}
	return r.Get(username)
}

// Get fetches one user by username.
func (r *UserRepo) Get(username string) (*User, error) {
	row := r.db.SQL().QueryRow(`SELECT id, username, password_hash, role, enabled,
		must_change_password, created_at, updated_at, last_login_at
		FROM users WHERE username = ?`, username)
	return scanUser(row)
}

// GetByID fetches by user ID.
func (r *UserRepo) GetByID(id string) (*User, error) {
	row := r.db.SQL().QueryRow(`SELECT id, username, password_hash, role, enabled,
		must_change_password, created_at, updated_at, last_login_at
		FROM users WHERE id = ?`, id)
	return scanUser(row)
}

// List returns all users ordered by name.
func (r *UserRepo) List() ([]*User, error) {
	rows, err := r.db.SQL().Query(`SELECT id, username, password_hash, role, enabled,
		must_change_password, created_at, updated_at, last_login_at
		FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CountAdmins returns the number of enabled admin+ users (auth check,
// spec section 8.8).
func (r *UserRepo) CountAdmins() (int, error) {
	var n int
	err := r.db.SQL().QueryRow(`SELECT COUNT(*) FROM users
		WHERE enabled = 1 AND role IN ('super_admin','admin')`).Scan(&n)
	return n, err
}

// SetPassword updates the hash and clears must-change.
func (r *UserRepo) SetPassword(id, hash string, mustChange bool) error {
	_, err := r.db.SQL().Exec(`UPDATE users SET password_hash = ?, must_change_password = ?,
		updated_at = ? WHERE id = ?`, hash, boolInt(mustChange), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// SetEnabled toggles a user.
func (r *UserRepo) SetEnabled(id string, enabled bool) error {
	_, err := r.db.SQL().Exec(`UPDATE users SET enabled = ?, updated_at = ? WHERE id = ?`,
		boolInt(enabled), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// SetRole changes the role.
func (r *UserRepo) SetRole(id string, role auth.Role) error {
	if !auth.ValidRole(role) {
		return fmt.Errorf("invalid role %q", role)
	}
	_, err := r.db.SQL().Exec(`UPDATE users SET role = ?, updated_at = ? WHERE id = ?`,
		string(role), time.Now().UTC().Format(time.RFC3339), id)
	return err
}

// Delete removes a user.
func (r *UserRepo) Delete(id string) error {
	_, err := r.db.SQL().Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// TouchLogin records a successful login.
func (r *UserRepo) TouchLogin(id string) error {
	_, err := r.db.SQL().Exec(`UPDATE users SET last_login_at = ? WHERE id = ?`,
		time.Now().UTC().Format(time.RFC3339), id)
	return err
}

type rowScanner interface{ Scan(dest ...any) error }

func scanUser(row rowScanner) (*User, error) {
	var u User
	var role string
	var enabled, mustChange int
	var created, updated string
	var lastLogin sql.NullString
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &role, &enabled,
		&mustChange, &created, &updated, &lastLogin); err != nil {
		return nil, err
	}
	u.Role = auth.Role(role)
	u.Enabled = enabled == 1
	u.MustChangePass = mustChange == 1
	u.CreatedAt = parseTime(created)
	u.UpdatedAt = parseTime(updated)
	if lastLogin.Valid && lastLogin.String != "" {
		t := parseTime(lastLogin.String)
		u.LastLoginAt = &t
	}
	return &u, nil
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func parseTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// HashToken hashes an enrollment/API token for storage. Raw tokens are
// never persisted (spec sections 66, 81).
func HashToken(tok string) string {
	h := sha256.Sum256([]byte(tok))
	return hex.EncodeToString(h[:])
}
