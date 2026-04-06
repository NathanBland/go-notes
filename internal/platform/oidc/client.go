package oidcclient

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"github.com/nathanbland/go-notes/internal/auth"
)

var ErrNonceMismatch = errors.New("oidc: nonce mismatch")

// Config keeps the OIDC client settings in one place for app wiring.
type Config struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	RedirectURL  string
	Scopes       []string
}

// Client wraps provider discovery, PKCE URL building, and token verification.
type Client struct {
	config       Config
	oauth2Config oauth2.Config
	verifier     *oidc.IDTokenVerifier
	mu           sync.Mutex
}

// New stores the config and defers provider discovery until auth is needed.
// That keeps local development smooth: the API can boot, hot-reload, and serve
// health checks even if the OIDC provider is not running yet.
func New(_ context.Context, config Config) (*Client, error) {
	scopes := config.Scopes
	if len(scopes) == 0 {
		scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	return &Client{
		config: config,
		oauth2Config: oauth2.Config{
			ClientID:     config.ClientID,
			ClientSecret: config.ClientSecret,
			RedirectURL:  config.RedirectURL,
			Scopes:       scopes,
		},
	}, nil
}

// AuthCodeURL builds the provider login URL for an authorization-code flow
// with PKCE and a nonce.
func (c *Client) AuthCodeURL(ctx context.Context, state, nonce, verifier string) (string, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return "", err
	}
	return c.oauth2Config.AuthCodeURL(
		state,
		oauth2.AccessTypeOffline,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.S256ChallengeOption(verifier),
	), nil
}

// Exchange trades the callback code for tokens, verifies the ID token, and
// maps provider claims into the internal auth identity model.
func (c *Client) Exchange(ctx context.Context, code, verifier, expectedNonce string) (auth.Identity, error) {
	if err := c.ensureInitialized(ctx); err != nil {
		return auth.Identity{}, err
	}
	token, err := c.oauth2Config.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return auth.Identity{}, err
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return auth.Identity{}, errors.New("oidc: missing id_token")
	}

	// The ID token is where the OIDC identity claims live. We verify the token,
	// then explicitly check the nonce and access token hash so the callback is
	// tied to the login attempt we started earlier.
	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return auth.Identity{}, err
	}
	if expectedNonce != "" && idToken.Nonce != expectedNonce {
		return auth.Identity{}, ErrNonceMismatch
	}
	if idToken.AccessTokenHash != "" {
		if err := idToken.VerifyAccessToken(token.AccessToken); err != nil {
			return auth.Identity{}, err
		}
	}

	var claims struct {
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return auth.Identity{}, err
	}

	displayName := firstNonEmpty(claims.Name, claims.PreferredUsername)
	return auth.Identity{
		Issuer:        idToken.Issuer,
		Subject:       idToken.Subject,
		Email:         stringPtr(claims.Email),
		EmailVerified: claims.EmailVerified,
		DisplayName:   stringPtr(displayName),
		PictureURL:    stringPtr(claims.Picture),
	}, nil
}

func (c *Client) ensureInitialized(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.verifier != nil {
		return nil
	}

	provider, err := oidc.NewProvider(ctx, c.config.IssuerURL)
	if err != nil {
		return err
	}
	c.oauth2Config.Endpoint = provider.Endpoint()
	c.verifier = provider.Verifier(&oidc.Config{ClientID: c.config.ClientID})
	return nil
}

func stringPtr(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
