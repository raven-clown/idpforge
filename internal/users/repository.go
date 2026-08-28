// Package users implements CRUD against the users table and password hashing.
package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/raven-clown/idpforge/internal/db"
)

var ErrNotFound = errors.New("user not found")

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) Create(ctx context.Context, in CreateInput) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	id := uuid.NewString()
	source := in.Source
	if source == "" {
		source = "local"
	}

	q := fmt.Sprintf(`INSERT INTO users (id, username, email, password_hash, status, source)
VALUES (%s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3),
		r.db.Placeholder(4), r.db.Placeholder(5), r.db.Placeholder(6))

	if _, err := r.db.ExecContext(ctx, q, id, in.Username, in.Email, string(hash), string(StatusActive), source); err != nil {
		return nil, err
	}
	return r.Get(ctx, id)
}

func (r *Repository) Get(ctx context.Context, id string) (*User, error) {
	q := fmt.Sprintf(`SELECT id, username, email, status, mfa_enabled, source, external_id, avatar_url, created_at, updated_at, last_login_at
FROM users WHERE id = %s`, r.db.Placeholder(1))
	return r.scanOne(r.db.QueryRowContext(ctx, q, id))
}

func (r *Repository) GetByUsername(ctx context.Context, username string) (*User, error) {
	q := fmt.Sprintf(`SELECT id, username, email, status, mfa_enabled, source, external_id, avatar_url, created_at, updated_at, last_login_at
FROM users WHERE username = %s`, r.db.Placeholder(1))
	return r.scanOne(r.db.QueryRowContext(ctx, q, username))
}

func (r *Repository) SetAvatar(ctx context.Context, id, avatarURL string) error {
	q := fmt.Sprintf(`UPDATE users SET avatar_url = %s WHERE id = %s`, r.db.Placeholder(1), r.db.Placeholder(2))
	_, err := r.db.ExecContext(ctx, q, avatarURL, id)
	return err
}

// SetPassword updates the password hash and resets password_changed_at,
// clearing any pending expiry.
func (r *Repository) SetPassword(ctx context.Context, id, newPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	q := fmt.Sprintf(`UPDATE users SET password_hash = %s, password_changed_at = %s WHERE id = %s`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3))
	_, err = r.db.ExecContext(ctx, q, string(hash), time.Now().UTC(), id)
	return err
}

// PasswordAge returns how long it has been since this user's password was
// last set.
func (r *Repository) PasswordAge(ctx context.Context, id string) (time.Duration, error) {
	var changedAt time.Time
	q := fmt.Sprintf(`SELECT password_changed_at FROM users WHERE id = %s`, r.db.Placeholder(1))
	if err := r.db.QueryRowContext(ctx, q, id).Scan(&changedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, err
	}
	return time.Since(changedAt), nil
}

// PasswordHash returns the bcrypt hash for the given username; used only by
// the local-password login path (LDAP/OIDC/SAML-sourced users have none).
func (r *Repository) PasswordHash(ctx context.Context, username string) (string, error) {
	var hash sql.NullString
	q := fmt.Sprintf(`SELECT password_hash FROM users WHERE username = %s`, r.db.Placeholder(1))
	if err := r.db.QueryRowContext(ctx, q, username).Scan(&hash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", err
	}
	return hash.String, nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]*User, error) {
	q := fmt.Sprintf(`SELECT id, username, email, status, mfa_enabled, source, external_id, avatar_url, created_at, updated_at, last_login_at
FROM users ORDER BY created_at DESC LIMIT %s OFFSET %s`, r.db.Placeholder(1), r.db.Placeholder(2))

	rows, err := r.db.QueryContext(ctx, q, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*User
	for rows.Next() {
		u, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *Repository) Update(ctx context.Context, id string, in UpdateInput) (*User, error) {
	if in.Email != nil {
		q := fmt.Sprintf(`UPDATE users SET email = %s, updated_at = %s WHERE id = %s`,
			r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3))
		if _, err := r.db.ExecContext(ctx, q, *in.Email, time.Now().UTC(), id); err != nil {
			return nil, err
		}
	}
	if in.Status != nil {
		q := fmt.Sprintf(`UPDATE users SET status = %s WHERE id = %s`, r.db.Placeholder(1), r.db.Placeholder(2))
		if _, err := r.db.ExecContext(ctx, q, string(*in.Status), id); err != nil {
			return nil, err
		}
	}
	return r.Get(ctx, id)
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM users WHERE id = %s`, r.db.Placeholder(1))
	res, err := r.db.ExecContext(ctx, q, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) TouchLastLogin(ctx context.Context, id string) error {
	q := fmt.Sprintf(`UPDATE users SET last_login_at = %s WHERE id = %s`, r.db.Placeholder(1), r.db.Placeholder(2))
	_, err := r.db.ExecContext(ctx, q, time.Now().UTC(), id)
	return err
}

func (r *Repository) VerifyPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

func (r *Repository) scanOne(row *sql.Row) (*User, error) {
	var u User
	var externalID, avatarURL sql.NullString
	var lastLogin sql.NullTime
	err := row.Scan(&u.ID, &u.Username, &u.Email, &u.Status, &u.MFAEnabled, &u.Source, &externalID, &avatarURL, &u.CreatedAt, &u.UpdatedAt, &lastLogin)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.ExternalID = externalID.String
	u.AvatarURL = avatarURL.String
	if lastLogin.Valid {
		t := lastLogin.Time
		u.LastLoginAt = &t
	}
	return &u, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanRow(rows rowScanner) (*User, error) {
	var u User
	var externalID, avatarURL sql.NullString
	var lastLogin sql.NullTime
	err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.Status, &u.MFAEnabled, &u.Source, &externalID, &avatarURL, &u.CreatedAt, &u.UpdatedAt, &lastLogin)
	if err != nil {
		return nil, err
	}
	u.ExternalID = externalID.String
	u.AvatarURL = avatarURL.String
	if lastLogin.Valid {
		t := lastLogin.Time
		u.LastLoginAt = &t
	}
	return &u, nil
}
