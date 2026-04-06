package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrUnauthorized = errors.New("auth: unauthorized")
	ErrInvalidState = errors.New("auth: invalid state")
	ErrInvalidCode  = errors.New("auth: invalid code")
)

// User is the authenticated API identity persisted in PostgreSQL.
type User struct {
	ID            uuid.UUID `json:"id"`
	OIDCIssuer    string    `json:"-"`
	OIDCSubject   string    `json:"-"`
	Email         *string   `json:"email"`
	EmailVerified bool      `json:"email_verified"`
	DisplayName   *string   `json:"display_name"`
	PictureURL    *string   `json:"picture_url"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Identity is the verified information extracted from an OIDC ID token.
type Identity struct {
	Issuer        string
	Subject       string
	Email         *string
	EmailVerified bool
	DisplayName   *string
	PictureURL    *string
}

// PendingAuth keeps enough PKCE state in Valkey to finish the callback safely.
type PendingAuth struct {
	State      string    `json:"state"`
	Nonce      string    `json:"nonce"`
	Verifier   string    `json:"verifier"`
	RedirectTo string    `json:"redirect_to,omitempty"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Session is intentionally opaque to clients; the cookie stores only the random ID.
type Session struct {
	ID        string    `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// UserStore persists authenticated users in PostgreSQL.
type UserStore interface {
	UpsertUserFromOIDC(ctx context.Context, identity Identity) (User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (User, error)
}

// OIDCClient handles provider redirects and verified token exchange.
type OIDCClient interface {
	AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error)
	Exchange(ctx context.Context, code, verifier, expectedNonce string) (Identity, error)
}

// SessionStore keeps short-lived login state plus authenticated sessions.
type SessionStore interface {
	StorePendingAuth(ctx context.Context, pending PendingAuth, ttl time.Duration) error
	ConsumePendingAuth(ctx context.Context, state string) (PendingAuth, error)
	CreateSession(ctx context.Context, userID uuid.UUID, ttl time.Duration) (Session, error)
	GetSession(ctx context.Context, id string) (Session, error)
	DeleteSession(ctx context.Context, id string) error
}
