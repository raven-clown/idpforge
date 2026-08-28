package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/raven-clown/idpforge/internal/cache"
)

const sessionCookie = "idpforge_session"
const sessionTTL = 8 * time.Hour

type sessionData struct {
	UserID             string `json:"user_id"`
	MustChangePassword bool   `json:"must_change_password,omitempty"`
}

type sessionStore struct {
	cache cache.Cache
}

func newSessionStore(c cache.Cache) *sessionStore {
	return &sessionStore{cache: c}
}

func (s *sessionStore) create(ctx context.Context, userID string, mustChangePassword bool) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(sessionData{UserID: userID, MustChangePassword: mustChangePassword})
	if err != nil {
		return "", err
	}
	if err := s.cache.Set(ctx, "session:"+id, string(encoded), sessionTTL); err != nil {
		return "", err
	}
	return id, nil
}

func (s *sessionStore) get(ctx context.Context, id string) (string, bool, error) {
	data, ok, err := s.getFull(ctx, id)
	if err != nil || !ok {
		return "", false, err
	}
	return data.UserID, true, nil
}

func (s *sessionStore) getFull(ctx context.Context, id string) (sessionData, bool, error) {
	raw, ok, err := s.cache.Get(ctx, "session:"+id)
	if err != nil || !ok {
		return sessionData{}, false, err
	}
	var data sessionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return sessionData{}, false, err
	}
	return data, true, nil
}

// clearMustChangePassword flips the flag off for an existing session, after
// a successful password change, without forcing a fresh login.
func (s *sessionStore) clearMustChangePassword(ctx context.Context, id, userID string) error {
	encoded, err := json.Marshal(sessionData{UserID: userID, MustChangePassword: false})
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, "session:"+id, string(encoded), sessionTTL)
}

func (s *sessionStore) destroy(ctx context.Context, id string) error {
	return s.cache.Delete(ctx, "session:"+id)
}

func randomID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
