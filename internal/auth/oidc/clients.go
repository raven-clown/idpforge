package oidc

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/raven-clown/idpforge/internal/db"
)

// Client is an application registered against this IdP. Backed by the
// applications table (protocol='oidc'); config holds client_id/secret/
// redirect_uris so no separate oauth_clients table is needed.
type Client struct {
	ID            string   `json:"client_id"`
	SecretHash    string   `json:"client_secret_hash"`
	RedirectURIs  []string `json:"redirect_uris"`
	AllowedScopes []string `json:"allowed_scopes"`
	Name          string   `json:"-"`
}

func (c Client) AllowsRedirect(uri string) bool {
	for _, u := range c.RedirectURIs {
		if u == uri {
			return true
		}
	}
	return false
}

type ClientStore struct {
	db *db.DB
}

func NewClientStore(database *db.DB) *ClientStore {
	return &ClientStore{db: database}
}

func (s *ClientStore) Get(ctx context.Context, clientID string) (*Client, error) {
	var name string
	var raw string
	q := fmt.Sprintf(`SELECT name, config FROM applications WHERE protocol = 'oidc' AND enabled = %s`, s.db.Placeholder(1))

	rows, err := s.db.QueryContext(ctx, q, true)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&name, &raw); err != nil {
			return nil, err
		}
		var c Client
		if err := json.Unmarshal([]byte(raw), &c); err != nil {
			continue
		}
		if c.ID == clientID {
			c.Name = name
			return &c, nil
		}
	}
	return nil, sql.ErrNoRows
}
