package webauthn

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	gowebauthn "github.com/go-webauthn/webauthn/webauthn"

	"github.com/raven-clown/idpforge/internal/cache"
)

const sessionTTL = 5 * time.Minute

type Service struct {
	wa    *gowebauthn.WebAuthn
	store *CredentialStore
	cache cache.Cache
}

func NewService(rpID, rpName, rpOrigin string, store *CredentialStore, c cache.Cache) (*Service, error) {
	wa, err := gowebauthn.New(&gowebauthn.Config{
		RPID:          rpID,
		RPDisplayName: rpName,
		RPOrigins:     []string{rpOrigin},
	})
	if err != nil {
		return nil, err
	}
	return &Service{wa: wa, store: store, cache: c}, nil
}

func (s *Service) BeginRegistration(ctx context.Context, userID string) (*protocol.CredentialCreation, error) {
	user, err := s.store.Load(ctx, userID)
	if err != nil {
		return nil, err
	}
	options, sessionData, err := s.wa.BeginRegistration(user)
	if err != nil {
		return nil, err
	}
	if err := s.putSession(ctx, "reg:"+userID, sessionData); err != nil {
		return nil, err
	}
	return options, nil
}

func (s *Service) FinishRegistration(ctx context.Context, userID string, response []byte) error {
	user, err := s.store.Load(ctx, userID)
	if err != nil {
		return err
	}
	sessionData, err := s.getSession(ctx, "reg:"+userID)
	if err != nil {
		return err
	}
	parsed, err := protocol.ParseCredentialCreationResponseBody(bytes.NewReader(response))
	if err != nil {
		return fmt.Errorf("parse registration response: %w", err)
	}
	cred, err := s.wa.CreateCredential(user, *sessionData, parsed)
	if err != nil {
		return fmt.Errorf("finish registration: %w", err)
	}
	return s.store.AddCredential(ctx, userID, *cred)
}

func (s *Service) BeginLogin(ctx context.Context, userID string) (*protocol.CredentialAssertion, error) {
	user, err := s.store.Load(ctx, userID)
	if err != nil {
		return nil, err
	}
	options, sessionData, err := s.wa.BeginLogin(user)
	if err != nil {
		return nil, err
	}
	if err := s.putSession(ctx, "login:"+userID, sessionData); err != nil {
		return nil, err
	}
	return options, nil
}

func (s *Service) FinishLogin(ctx context.Context, userID string, response []byte) error {
	user, err := s.store.Load(ctx, userID)
	if err != nil {
		return err
	}
	sessionData, err := s.getSession(ctx, "login:"+userID)
	if err != nil {
		return err
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(response))
	if err != nil {
		return fmt.Errorf("parse login response: %w", err)
	}
	_, err = s.wa.ValidateLogin(user, *sessionData, parsed)
	if err != nil {
		return fmt.Errorf("finish login: %w", err)
	}
	return nil
}

func (s *Service) putSession(ctx context.Context, key string, data *gowebauthn.SessionData) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, "webauthn:session:"+key, string(encoded), sessionTTL)
}

func (s *Service) getSession(ctx context.Context, key string) (*gowebauthn.SessionData, error) {
	raw, ok, err := s.cache.Get(ctx, "webauthn:session:"+key)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no pending webauthn ceremony for %s", key)
	}
	var data gowebauthn.SessionData
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		return nil, err
	}
	return &data, nil
}
