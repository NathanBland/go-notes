package oidcclient

import (
	"context"
	"testing"
)

func TestNewUsesDefaultScopes(t *testing.T) {
	client, err := New(context.Background(), Config{ClientID: "client", RedirectURL: "http://localhost/callback"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(client.oauth2Config.Scopes) != 3 {
		t.Fatalf("expected default scopes, got %#v", client.oauth2Config.Scopes)
	}
}

func TestAuthCodeURLReturnsProviderError(t *testing.T) {
	client, err := New(context.Background(), Config{
		IssuerURL:   "http://127.0.0.1:1",
		ClientID:    "client-id",
		RedirectURL: "http://localhost:8080/callback",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if _, err := client.AuthCodeURL(context.Background(), "state", "nonce", "verifier"); err == nil {
		t.Fatal("expected provider discovery error")
	}
}

func TestStringHelpers(t *testing.T) {
	if stringPtr("   ") != nil {
		t.Fatal("expected blank string to map to nil")
	}
	if got := stringPtr(" hello "); got == nil || *got != "hello" {
		t.Fatalf("unexpected string pointer result: %v", got)
	}
	if got := firstNonEmpty("", " second ", "third"); got != "second" {
		t.Fatalf("unexpected first non-empty result: %q", got)
	}
}
