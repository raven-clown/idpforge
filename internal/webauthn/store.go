// Package webauthn wires go-webauthn into IdpForge's user store so
// fingerprint/face/security-key enrollment ("biometric login") goes through
// the standard FIDO2 ceremony: the browser and OS do the actual biometric
// capture and matching, the server only ever sees a public-key credential.
package webauthn

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/raven-clown/idpforge/internal/db"
)

type CredentialStore struct {
	db *db.DB
}

func NewCredentialStore(database *db.DB) *CredentialStore {
	return &CredentialStore{db: database}
}

// webAuthnUser adapts a users.User row to the gowebauthn.User interface.
type webAuthnUser struct {
	id          string
	username    string
	credentials []gowebauthn.Credential
}

func (u *webAuthnUser) WebAuthnID() []byte                           { return []byte(u.id) }
func (u *webAuthnUser) WebAuthnName() string                         { return u.username }
func (u *webAuthnUser) WebAuthnDisplayName() string                  { return u.username }
func (u *webAuthnUser) WebAuthnCredentials() []gowebauthn.Credential { return u.credentials }
func (u *webAuthnUser) WebAuthnIcon() string                         { return "" }

func (s *CredentialStore) Load(ctx context.Context, userID string) (*webAuthnUser, error) {
	var username string
	var raw sql.NullString
	q := fmt.Sprintf(`SELECT username, webauthn_credentials FROM users WHERE id = %s`, s.db.Placeholder(1))
	if err := s.db.QueryRowContext(ctx, q, userID).Scan(&username, &raw); err != nil {
		return nil, err
	}

	var creds []gowebauthn.Credential
	if raw.Valid && raw.String != "" {
		if err := json.Unmarshal([]byte(raw.String), &creds); err != nil {
			return nil, fmt.Errorf("decode stored credentials: %w", err)
		}
	}
	return &webAuthnUser{id: userID, username: username, credentials: creds}, nil
}

func (s *CredentialStore) AddCredential(ctx context.Context, userID string, cred gowebauthn.Credential) error {
	u, err := s.Load(ctx, userID)
	if err != nil {
		return err
	}
	u.credentials = append(u.credentials, cred)
	return s.save(ctx, userID, u.credentials)
}

func (s *CredentialStore) RemoveCredential(ctx context.Context, userID string, credentialID []byte) error {
	u, err := s.Load(ctx, userID)
	if err != nil {
		return err
	}
	kept := u.credentials[:0]
	for _, c := range u.credentials {
		if string(c.ID) != string(credentialID) {
			kept = append(kept, c)
		}
	}
	return s.save(ctx, userID, kept)
}

func (s *CredentialStore) save(ctx context.Context, userID string, creds []gowebauthn.Credential) error {
	encoded, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	q := fmt.Sprintf(`UPDATE users SET webauthn_credentials = %s, mfa_enabled = %s WHERE id = %s`,
		s.db.Placeholder(1), s.db.Placeholder(2), s.db.Placeholder(3))
	_, err = s.db.ExecContext(ctx, q, string(encoded), len(creds) > 0, userID)
	return err
}
