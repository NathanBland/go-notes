package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type KV interface {
	Get(ctx context.Context, key string) (string, bool)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Delete(ctx context.Context, keys ...string) error
}

// CacheSessionStore keeps sessions and pending OIDC state in Valkey as JSON.
type CacheSessionStore struct {
	kv KV
}

func NewCacheSessionStore(kv KV) *CacheSessionStore {
	return &CacheSessionStore{kv: kv}
}

func (s *CacheSessionStore) StorePendingAuth(ctx context.Context, pending PendingAuth, ttl time.Duration) error {
	payload, err := json.Marshal(pending)
	if err != nil {
		return err
	}
	return s.kv.Set(ctx, pendingAuthKey(pending.State), string(payload), ttl)
}

func (s *CacheSessionStore) ConsumePendingAuth(ctx context.Context, state string) (PendingAuth, error) {
	payload, ok := s.kv.Get(ctx, pendingAuthKey(state))
	if !ok {
		return PendingAuth{}, ErrInvalidState
	}
	// State is single-use. Deleting it before the callback finishes prevents the
	// same browser redirect from being replayed a second time.
	if err := s.kv.Delete(ctx, pendingAuthKey(state)); err != nil {
		return PendingAuth{}, err
	}

	var pending PendingAuth
	if err := json.Unmarshal([]byte(payload), &pending); err != nil {
		return PendingAuth{}, err
	}
	if time.Now().UTC().After(pending.ExpiresAt) {
		return PendingAuth{}, ErrInvalidState
	}
	return pending, nil
}

func (s *CacheSessionStore) CreateSession(ctx context.Context, userID uuid.UUID, ttl time.Duration) (Session, error) {
	id, err := randomToken(32)
	if err != nil {
		return Session{}, err
	}
	createdAt := time.Now().UTC()
	session := Session{
		ID:        id,
		UserID:    userID,
		CreatedAt: createdAt,
		ExpiresAt: createdAt.Add(ttl),
	}
	payload, err := json.Marshal(session)
	if err != nil {
		return Session{}, err
	}
	if err := s.kv.Set(ctx, sessionKey(id), string(payload), ttl); err != nil {
		return Session{}, err
	}
	return session, nil
}

func (s *CacheSessionStore) GetSession(ctx context.Context, id string) (Session, error) {
	payload, ok := s.kv.Get(ctx, sessionKey(id))
	if !ok {
		return Session{}, ErrUnauthorized
	}

	var session Session
	if err := json.Unmarshal([]byte(payload), &session); err != nil {
		return Session{}, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.kv.Delete(ctx, sessionKey(id))
		return Session{}, ErrUnauthorized
	}
	return session, nil
}

func (s *CacheSessionStore) DeleteSession(ctx context.Context, id string) error {
	return s.kv.Delete(ctx, sessionKey(id))
}

func pendingAuthKey(state string) string {
	return fmt.Sprintf("oidc:state:%s", state)
}

func sessionKey(id string) string {
	return fmt.Sprintf("session:%s", id)
}
