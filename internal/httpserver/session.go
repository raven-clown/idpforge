package httpserver

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/raven-clown/idpforge/internal/cache"
)

const sessionCookie = "idpforge_session"
const sessionTTL = 8 * time.Hour

type sessionData struct {
	UserID string `json:"user_id"`
}

type sessionStore struct {
	cache cache.Cache
}

func newSessionStore(c cache.Cache) *sessionStore {
	return &sessionStore{cache: c}
}

func (s *sessionStore) create(ctx context.Context, userID string) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(sessionData{UserID: userID})
	if err != nil {
		return "", err
	}
	if err := s.cache.Set(ctx, "session:"+id, string(encoded), sessionTTL); err != nil {
		return "", err
	}
	return id, nil
}

func (s *sessionStore) get(ctx context.Context, id string) (string, bool, error) {
	raw, ok, err := s.cache.Get(ctx, "session:"+id)
	if err != nil || !ok {
		return "", false, err
	}
	var data sessionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return "", false, err
	}
	return data.UserID, true, nil
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

// requireSession is Fiber middleware that resolves the session cookie into
// a user ID stored on c.Locals("user_id"), or returns 401.
func requireSession(store *sessionStore) fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Cookies(sessionCookie)
		if id == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "not authenticated")
		}
		userID, ok, err := store.get(c.Context(), id)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "session lookup failed")
		}
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "session expired")
		}
		c.Locals("user_id", userID)
		return c.Next()
	}
}
