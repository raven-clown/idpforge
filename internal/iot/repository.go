package iot

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/db"
)

var ErrNotFound = errors.New("not found")
var ErrCredentialMismatch = errors.New("credentials resolve to different users")

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) CreateDevice(ctx context.Context, name, deviceType, location string, allowedIPs []string) (*Device, string, error) {
	apiKey, err := randomKey()
	if err != nil {
		return nil, "", err
	}
	ipsJSON, err := json.Marshal(allowedIPs)
	if err != nil {
		return nil, "", err
	}
	id := uuid.NewString()
	q := fmt.Sprintf(`INSERT INTO iot_devices (id, name, device_type, location, api_key_hash, allowed_ips) VALUES (%s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4), r.db.Placeholder(5), r.db.Placeholder(6))
	if _, err := r.db.ExecContext(ctx, q, id, name, deviceType, location, hashKey(apiKey), string(ipsJSON)); err != nil {
		return nil, "", err
	}
	return &Device{ID: id, Name: name, DeviceType: deviceType, Location: location, AllowedIPs: allowedIPs, Enabled: true}, apiKey, nil
}

func (r *Repository) ListDevices(ctx context.Context) ([]Device, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, device_type, location, allowed_ips, enabled, created_at FROM iot_devices ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Device
	for rows.Next() {
		d, err := scanDevice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *d)
	}
	return out, rows.Err()
}

// AuthenticateDevice resolves an X-Device-Key header to its device, or
// ErrNotFound if the key doesn't match an enabled device.
func (r *Repository) AuthenticateDevice(ctx context.Context, apiKey string) (*Device, error) {
	q := fmt.Sprintf(`SELECT id, name, device_type, location, allowed_ips, enabled, created_at FROM iot_devices WHERE api_key_hash = %s AND enabled = %s`,
		r.db.Placeholder(1), r.db.Placeholder(2))
	d, err := scanDevice(r.db.QueryRowContext(ctx, q, hashKey(apiKey), true))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return d, err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanDevice(row rowScanner) (*Device, error) {
	var d Device
	var location, ipsJSON sql.NullString
	if err := row.Scan(&d.ID, &d.Name, &d.DeviceType, &location, &ipsJSON, &d.Enabled, &d.CreatedAt); err != nil {
		return nil, err
	}
	d.Location = location.String
	if ipsJSON.Valid && ipsJSON.String != "" {
		if err := json.Unmarshal([]byte(ipsJSON.String), &d.AllowedIPs); err != nil {
			return nil, fmt.Errorf("decode allowed_ips: %w", err)
		}
	}
	return &d, nil
}

func (r *Repository) AddCredential(ctx context.Context, userID, credType, ref, label string) (*Credential, error) {
	id := uuid.NewString()
	q := fmt.Sprintf(`INSERT INTO device_credentials (id, user_id, credential_type, credential_ref, label) VALUES (%s, %s, %s, %s, %s)`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4), r.db.Placeholder(5))
	if _, err := r.db.ExecContext(ctx, q, id, userID, credType, ref, label); err != nil {
		return nil, err
	}
	return &Credential{ID: id, UserID: userID, CredentialType: credType, CredentialRef: ref, Label: label}, nil
}

func (r *Repository) ListCredentials(ctx context.Context, userID string) ([]Credential, error) {
	q := fmt.Sprintf(`SELECT id, user_id, credential_type, credential_ref, label, created_at FROM device_credentials WHERE user_id = %s ORDER BY created_at`, r.db.Placeholder(1))
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Credential
	for rows.Next() {
		var c Credential
		var label sql.NullString
		if err := rows.Scan(&c.ID, &c.UserID, &c.CredentialType, &c.CredentialRef, &label, &c.CreatedAt); err != nil {
			return nil, err
		}
		c.Label = label.String
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) DeleteCredential(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM device_credentials WHERE id = %s`, r.db.Placeholder(1))
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

func (r *Repository) resolveCredential(ctx context.Context, credType, ref string) (string, error) {
	var userID string
	q := fmt.Sprintf(`SELECT user_id FROM device_credentials WHERE credential_type = %s AND credential_ref = %s`,
		r.db.Placeholder(1), r.db.Placeholder(2))
	err := r.db.QueryRowContext(ctx, q, credType, ref).Scan(&userID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}

// ResolveUser matches one or more scanned proofs to a single user. All
// proofs must agree on the same user_id; a mismatch (e.g. a face match and
// a card swipe that resolve to different people, exactly the failure mode
// that motivates requiring a second factor for lookalikes) is reported as
// ErrCredentialMismatch rather than silently picking one.
func (r *Repository) ResolveUser(ctx context.Context, proofs []CredentialProof) (string, error) {
	if len(proofs) == 0 {
		return "", fmt.Errorf("no credentials supplied")
	}
	var userID string
	for i, p := range proofs {
		resolved, err := r.resolveCredential(ctx, p.CredentialType, p.CredentialRef)
		if err != nil {
			return "", err
		}
		if i == 0 {
			userID = resolved
			continue
		}
		if resolved != userID {
			return "", ErrCredentialMismatch
		}
	}
	return userID, nil
}

func (r *Repository) RecordEvent(ctx context.Context, e Event) (int64, error) {
	metadata := e.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}
	q := fmt.Sprintf(`INSERT INTO device_events (device_id, user_id, credential_type, credential_ref, event_type, resource, metadata, status, "timestamp")
VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4),
		r.db.Placeholder(5), r.db.Placeholder(6), r.db.Placeholder(7), r.db.Placeholder(8), r.db.Placeholder(9))

	res, err := r.db.ExecContext(ctx, q,
		e.DeviceID, nullableString(e.UserID), nullableString(e.CredentialType), nullableString(e.CredentialRef),
		e.EventType, nullableString(e.Resource), string(metadata), e.Status, e.Timestamp)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, nil // driver may not support LastInsertId (fine, id is informational)
	}
	return id, nil
}

// HasEventToday reports whether a matching (user, event_type, resource)
// event already exists since the start of today (UTC). A downstream system
// (a canteen POS enforcing one discount per day, a door controller logging
// re-entries) uses this to implement its own policy.
func (r *Repository) HasEventToday(ctx context.Context, userID, eventType, resource string) (bool, error) {
	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	q := fmt.Sprintf(`SELECT COUNT(*) FROM device_events WHERE user_id = %s AND event_type = %s AND resource = %s AND "timestamp" >= %s`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4))
	var count int
	if err := r.db.QueryRowContext(ctx, q, userID, eventType, resource, startOfDay).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *Repository) QueryEvents(ctx context.Context, f EventFilter) ([]Event, error) {
	where := []string{"1=1"}
	var args []interface{}
	add := func(cond string, val interface{}) {
		args = append(args, val)
		where = append(where, fmt.Sprintf(cond, r.db.Placeholder(len(args))))
	}

	if f.UserID != "" {
		add("user_id = %s", f.UserID)
	}
	if f.DeviceID != "" {
		add("device_id = %s", f.DeviceID)
	}
	if f.EventType != "" {
		add("event_type = %s", f.EventType)
	}
	if f.Resource != "" {
		add("resource = %s", f.Resource)
	}
	if f.Since != nil {
		add(`"timestamp" >= %s`, *f.Since)
	}
	if f.Until != nil {
		add(`"timestamp" <= %s`, *f.Until)
	}

	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	args = append(args, limit)
	limitPh := r.db.Placeholder(len(args))
	args = append(args, f.Offset)
	offsetPh := r.db.Placeholder(len(args))

	q := fmt.Sprintf(`SELECT id, device_id, user_id, credential_type, credential_ref, event_type, resource, metadata, status, "timestamp"
FROM device_events WHERE %s ORDER BY "timestamp" DESC LIMIT %s OFFSET %s`, strings.Join(where, " AND "), limitPh, offsetPh)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var e Event
		var userID, credType, credRef, resource sql.NullString
		var metadata sql.NullString
		if err := rows.Scan(&e.ID, &e.DeviceID, &userID, &credType, &credRef, &e.EventType, &resource, &metadata, &e.Status, &e.Timestamp); err != nil {
			return nil, err
		}
		e.UserID = userID.String
		e.CredentialType = credType.String
		e.CredentialRef = credRef.String
		e.Resource = resource.String
		if metadata.Valid {
			e.Metadata = []byte(metadata.String)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

func randomKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "iotk_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}
