package apiclient

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

	"github.com/google/uuid"

	"github.com/raven-clown/idpforge/internal/db"
)

var ErrNotFound = errors.New("not found")

type Repository struct {
	db *db.DB
}

func NewRepository(database *db.DB) *Repository {
	return &Repository{db: database}
}

func (r *Repository) Create(ctx context.Context, name string, allowedFields, scopes, allowedIPs []string, rateLimitMax, rateLimitWindowSeconds int) (*Client, string, error) {
	apiKey, err := randomKey()
	if err != nil {
		return nil, "", err
	}
	if len(allowedFields) == 0 {
		allowedFields = []string{"id", "username"}
	}
	fieldsJSON, err := json.Marshal(allowedFields)
	if err != nil {
		return nil, "", err
	}
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return nil, "", err
	}
	ipsJSON, err := json.Marshal(allowedIPs)
	if err != nil {
		return nil, "", err
	}

	id := uuid.NewString()
	q := fmt.Sprintf(`INSERT INTO api_clients (id, name, api_key_hash, allowed_fields, scopes, allowed_ips, rate_limit_max, rate_limit_window_seconds)
VALUES (%s, %s, %s, %s, %s, %s, %s, %s)`,
		r.db.Placeholder(1), r.db.Placeholder(2), r.db.Placeholder(3), r.db.Placeholder(4), r.db.Placeholder(5), r.db.Placeholder(6), r.db.Placeholder(7), r.db.Placeholder(8))
	if _, err := r.db.ExecContext(ctx, q, id, name, hashKey(apiKey), string(fieldsJSON), string(scopesJSON), string(ipsJSON), rateLimitMax, rateLimitWindowSeconds); err != nil {
		return nil, "", err
	}

	return &Client{
		ID: id, Name: name, AllowedFields: allowedFields, Scopes: scopes, AllowedIPs: allowedIPs,
		RateLimitMax: rateLimitMax, RateLimitWindowSeconds: rateLimitWindowSeconds, Enabled: true,
	}, apiKey, nil
}

func (r *Repository) List(ctx context.Context) ([]Client, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, name, allowed_fields, scopes, allowed_ips, rate_limit_max, rate_limit_window_seconds, enabled, created_at FROM api_clients ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Client
	for rows.Next() {
		c, err := scanClient(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func (r *Repository) Authenticate(ctx context.Context, apiKey string) (*Client, error) {
	q := fmt.Sprintf(`SELECT id, name, allowed_fields, scopes, allowed_ips, rate_limit_max, rate_limit_window_seconds, enabled, created_at
FROM api_clients WHERE api_key_hash = %s AND enabled = %s`, r.db.Placeholder(1), r.db.Placeholder(2))
	row := r.db.QueryRowContext(ctx, q, hashKey(apiKey), true)
	c, err := scanClient(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return c, err
}

func (r *Repository) Delete(ctx context.Context, id string) error {
	q := fmt.Sprintf(`DELETE FROM api_clients WHERE id = %s`, r.db.Placeholder(1))
	_, err := r.db.ExecContext(ctx, q, id)
	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanClient(row rowScanner) (*Client, error) {
	var c Client
	var fieldsJSON string
	var scopesJSON, ipsJSON sql.NullString
	if err := row.Scan(&c.ID, &c.Name, &fieldsJSON, &scopesJSON, &ipsJSON, &c.RateLimitMax, &c.RateLimitWindowSeconds, &c.Enabled, &c.CreatedAt); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(fieldsJSON), &c.AllowedFields); err != nil {
		return nil, fmt.Errorf("decode allowed_fields: %w", err)
	}
	if scopesJSON.Valid && scopesJSON.String != "" {
		if err := json.Unmarshal([]byte(scopesJSON.String), &c.Scopes); err != nil {
			return nil, fmt.Errorf("decode scopes: %w", err)
		}
	}
	if ipsJSON.Valid && ipsJSON.String != "" {
		if err := json.Unmarshal([]byte(ipsJSON.String), &c.AllowedIPs); err != nil {
			return nil, fmt.Errorf("decode allowed_ips: %w", err)
		}
	}
	return &c, nil
}

func randomKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "apik_" + base64.RawURLEncoding.EncodeToString(b), nil
}

func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// FilterFields projects a user record (as a JSON-decoded map) down to only
// the fields this client is allowed to see.
func FilterFields(full map[string]interface{}, allowed []string) map[string]interface{} {
	out := make(map[string]interface{}, len(allowed))
	for _, field := range allowed {
		if v, ok := full[field]; ok {
			out[field] = v
		}
	}
	return out
}
