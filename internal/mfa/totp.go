// Package mfa implements TOTP enrollment and verification, required for
// admin-and-above accounts alongside the WebAuthn path in internal/webauthn.
package mfa

import (
	"context"
	"fmt"

	"github.com/pquerna/otp/totp"

	"github.com/raven-clown/idpforge/internal/db"
)

type Service struct {
	db     *db.DB
	issuer string
}

func NewService(database *db.DB, issuer string) *Service {
	return &Service{db: database, issuer: issuer}
}

func (s *Service) Enroll(ctx context.Context, userID, username string) (secret string, otpauthURL string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      s.issuer,
		AccountName: username,
	})
	if err != nil {
		return "", "", err
	}

	q := fmt.Sprintf(`UPDATE users SET mfa_secret = %s WHERE id = %s`, s.db.Placeholder(1), s.db.Placeholder(2))
	if _, err := s.db.ExecContext(ctx, q, key.Secret(), userID); err != nil {
		return "", "", err
	}
	return key.Secret(), key.URL(), nil
}

// Confirm verifies the first code after enrollment and flips mfa_enabled on.
func (s *Service) Confirm(ctx context.Context, userID, code string) error {
	secret, err := s.secretFor(ctx, userID)
	if err != nil {
		return err
	}
	if !totp.Validate(code, secret) {
		return fmt.Errorf("invalid code")
	}
	q := fmt.Sprintf(`UPDATE users SET mfa_enabled = %s WHERE id = %s`, s.db.Placeholder(1), s.db.Placeholder(2))
	_, err = s.db.ExecContext(ctx, q, true, userID)
	return err
}

func (s *Service) Verify(ctx context.Context, userID, code string) (bool, error) {
	secret, err := s.secretFor(ctx, userID)
	if err != nil {
		return false, err
	}
	return totp.Validate(code, secret), nil
}

func (s *Service) secretFor(ctx context.Context, userID string) (string, error) {
	var secret string
	q := fmt.Sprintf(`SELECT mfa_secret FROM users WHERE id = %s`, s.db.Placeholder(1))
	if err := s.db.QueryRowContext(ctx, q, userID).Scan(&secret); err != nil {
		return "", err
	}
	if secret == "" {
		return "", fmt.Errorf("no MFA secret enrolled")
	}
	return secret, nil
}
