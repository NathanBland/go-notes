package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"
)

// Service coordinates OIDC, users, and the Valkey-backed session store.
type Service struct {
	users      UserStore
	oidc       OIDCClient
	sessions   SessionStore
	sessionTTL time.Duration
	stateTTL   time.Duration
}

func NewService(users UserStore, oidc OIDCClient, sessions SessionStore, sessionTTL, stateTTL time.Duration) *Service {
	return &Service{
		users:      users,
		oidc:       oidc,
		sessions:   sessions,
		sessionTTL: sessionTTL,
		stateTTL:   stateTTL,
	}
}

// BeginLogin creates one-time OIDC callback state, stores it in Valkey, and
// returns the provider redirect URL.
func (s *Service) BeginLogin(ctx context.Context, redirectTo string) (string, error) {
	state, err := randomToken(32)
	if err != nil {
		return "", err
	}
	nonce, err := randomToken(32)
	if err != nil {
		return "", err
	}
	verifier, err := randomToken(48)
	if err != nil {
		return "", err
	}

	pending := PendingAuth{
		State:      state,
		Nonce:      nonce,
		Verifier:   verifier,
		RedirectTo: redirectTo,
		ExpiresAt:  time.Now().UTC().Add(s.stateTTL),
	}
	if err := s.sessions.StorePendingAuth(ctx, pending, s.stateTTL); err != nil {
		return "", err
	}
	redirectURL, err := s.oidc.AuthCodeURL(ctx, state, nonce, verifier)
	if err != nil {
		return "", err
	}
	return redirectURL, nil
}

// FinishLogin consumes the pending OIDC state, verifies the callback with the
// provider, upserts the user, and creates an opaque server-side session.
func (s *Service) FinishLogin(ctx context.Context, state, code string) (User, Session, PendingAuth, error) {
	if state == "" {
		return User{}, Session{}, PendingAuth{}, ErrInvalidState
	}
	if code == "" {
		return User{}, Session{}, PendingAuth{}, ErrInvalidCode
	}

	pending, err := s.sessions.ConsumePendingAuth(ctx, state)
	if err != nil {
		return User{}, Session{}, PendingAuth{}, err
	}

	identity, err := s.oidc.Exchange(ctx, code, pending.Verifier, pending.Nonce)
	if err != nil {
		return User{}, Session{}, PendingAuth{}, err
	}

	user, err := s.users.UpsertUserFromOIDC(ctx, identity)
	if err != nil {
		return User{}, Session{}, PendingAuth{}, err
	}

	session, err := s.sessions.CreateSession(ctx, user.ID, s.sessionTTL)
	if err != nil {
		return User{}, Session{}, PendingAuth{}, err
	}

	return user, session, pending, nil
}

// Authenticate resolves an opaque session ID into the current user.
func (s *Service) Authenticate(ctx context.Context, sessionID string) (User, Session, error) {
	if sessionID == "" {
		return User{}, Session{}, ErrUnauthorized
	}

	session, err := s.sessions.GetSession(ctx, sessionID)
	if err != nil {
		return User{}, Session{}, ErrUnauthorized
	}

	user, err := s.users.GetUserByID(ctx, session.UserID)
	if err != nil {
		return User{}, Session{}, ErrUnauthorized
	}

	return user, session, nil
}

// Logout deletes the server-side session. The HTTP layer handles expiring the
// browser cookie separately.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.sessions.DeleteSession(ctx, sessionID)
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
