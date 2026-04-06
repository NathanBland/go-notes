package oidcclient

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	jose "github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

func TestAuthCodeURLAndExchangeWithLocalIssuer(t *testing.T) {
	issuer := newTestOIDCIssuer(t)
	client, err := New(context.Background(), Config{
		IssuerURL:    issuer.server.URL,
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:8080/callback",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	authURL, err := client.AuthCodeURL(context.Background(), "state-1", "nonce-1", "verifier-1")
	if err != nil {
		t.Fatalf("unexpected auth URL error: %v", err)
	}
	parsed, err := url.Parse(authURL)
	if err != nil {
		t.Fatalf("unexpected auth URL parse error: %v", err)
	}
	if parsed.Query().Get("state") != "state-1" {
		t.Fatalf("expected state to round-trip, got %q", parsed.Query().Get("state"))
	}
	if parsed.Query().Get("nonce") != "nonce-1" {
		t.Fatalf("expected nonce to round-trip, got %q", parsed.Query().Get("nonce"))
	}

	identity, err := client.Exchange(context.Background(), "good-code", "verifier-1", "nonce-1")
	if err != nil {
		t.Fatalf("unexpected exchange error: %v", err)
	}
	if identity.Issuer != issuer.server.URL || identity.Subject != "subject-1" {
		t.Fatalf("unexpected identity: %+v", identity)
	}
	if identity.DisplayName == nil || *identity.DisplayName != "Example User" {
		t.Fatalf("expected display name from claims, got %+v", identity)
	}
}

func TestExchangeHandlesMissingIDTokenAndNonceMismatch(t *testing.T) {
	issuer := newTestOIDCIssuer(t)
	client, err := New(context.Background(), Config{
		IssuerURL:    issuer.server.URL,
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:8080/callback",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}

	if _, err := client.Exchange(context.Background(), "missing-id-token", "verifier-1", "nonce-1"); err == nil {
		t.Fatal("expected missing id_token error")
	}

	if _, err := client.Exchange(context.Background(), "good-code", "verifier-1", "wrong-nonce"); err != ErrNonceMismatch {
		t.Fatalf("expected nonce mismatch, got %v", err)
	}
}

type testOIDCIssuer struct {
	server     *httptest.Server
	privateKey *rsa.PrivateKey
}

func newTestOIDCIssuer(t *testing.T) *testOIDCIssuer {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate rsa key: %v", err)
	}

	issuer := &testOIDCIssuer{privateKey: privateKey}
	mux := http.NewServeMux()
	mux.HandleFunc("/.well-known/openid-configuration", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"issuer":                 issuer.server.URL,
			"authorization_endpoint": issuer.server.URL + "/authorize",
			"token_endpoint":         issuer.server.URL + "/token",
			"jwks_uri":               issuer.server.URL + "/keys",
			"id_token_signing_alg_values_supported": []string{
				"RS256",
			},
		})
	})
	mux.HandleFunc("/keys", func(w http.ResponseWriter, r *http.Request) {
		keySet := jose.JSONWebKeySet{
			Keys: []jose.JSONWebKey{{
				Key:       &privateKey.PublicKey,
				KeyID:     "test-key-1",
				Algorithm: string(jose.RS256),
				Use:       "sig",
			}},
		}
		writeJSON(w, keySet)
	})
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://localhost/unused", http.StatusFound)
	})
	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		code := r.Form.Get("code")
		switch code {
		case "missing-id-token":
			writeJSON(w, map[string]any{
				"access_token": "access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
			})
		default:
			token, err := issuer.signToken("nonce-1")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]any{
				"access_token": "access-token",
				"token_type":   "Bearer",
				"expires_in":   3600,
				"id_token":     token,
			})
		}
	})

	issuer.server = httptest.NewServer(mux)
	t.Cleanup(issuer.server.Close)
	return issuer
}

func (i *testOIDCIssuer) signToken(nonce string) (string, error) {
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: i.privateKey}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test-key-1"))
	if err != nil {
		return "", err
	}

	claims := jwt.Claims{
		Issuer:   i.server.URL,
		Subject:  "subject-1",
		Audience: jwt.Audience{"client-id"},
		Expiry:   jwt.NewNumericDate(time.Now().Add(time.Hour)),
		IssuedAt: jwt.NewNumericDate(time.Now()),
	}
	privateClaims := struct {
		Nonce             string `json:"nonce"`
		Email             string `json:"email"`
		EmailVerified     bool   `json:"email_verified"`
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
		Picture           string `json:"picture"`
	}{
		Nonce:             nonce,
		Email:             "user@example.com",
		EmailVerified:     true,
		Name:              "Example User",
		PreferredUsername: "example",
		Picture:           "https://example.com/avatar.png",
	}
	return jwt.Signed(signer).Claims(claims).Claims(privateClaims).Serialize()
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func TestEnsureInitializedCachesProvider(t *testing.T) {
	issuer := newTestOIDCIssuer(t)
	client, err := New(context.Background(), Config{
		IssuerURL:    issuer.server.URL,
		ClientID:     "client-id",
		ClientSecret: "secret",
		RedirectURL:  "http://localhost:8080/callback",
	})
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if err := client.ensureInitialized(context.Background()); err != nil {
		t.Fatalf("unexpected initialize error: %v", err)
	}
	firstVerifier := client.verifier
	if firstVerifier == nil {
		t.Fatal("expected verifier to be initialized")
	}
	if err := client.ensureInitialized(context.Background()); err != nil {
		t.Fatalf("unexpected second initialize error: %v", err)
	}
	if client.verifier != firstVerifier {
		t.Fatal("expected verifier to be reused after first initialization")
	}
	if !strings.Contains(client.oauth2Config.Endpoint.TokenURL, "/token") {
		t.Fatalf("expected token endpoint from discovery, got %+v", client.oauth2Config.Endpoint)
	}
}
