package app

import (
	"strings"
	"testing"
	"time"
)

func TestLoadConfigSuccess(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable")
	t.Setenv("OIDC_ISSUER_URL", "http://localhost:8081/realms/go-notes")
	t.Setenv("OIDC_CLIENT_ID", "go-notes")
	t.Setenv("OIDC_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback")
	t.Setenv("SESSION_TTL", "2h")
	t.Setenv("SESSION_COOKIE_SECURE", "true")
	t.Setenv("OIDC_SCOPES", "openid, profile ,email")
	t.Setenv("THROTTLE_REQUESTS_PER_SECOND", "3.5")
	t.Setenv("THROTTLE_BURST", "7")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SessionTTL != 2*time.Hour {
		t.Fatalf("expected parsed session ttl, got %s", cfg.SessionTTL)
	}
	if !cfg.SessionCookieSecure {
		t.Fatal("expected secure cookie to be parsed")
	}
	if len(cfg.OIDC.Scopes) != 3 {
		t.Fatalf("expected scopes to be split, got %#v", cfg.OIDC.Scopes)
	}
	if cfg.ThrottleRequestsPS != 3.5 {
		t.Fatalf("expected parsed throttle rps, got %v", cfg.ThrottleRequestsPS)
	}
	if cfg.ThrottleBurst != 7 {
		t.Fatalf("expected parsed throttle burst, got %d", cfg.ThrottleBurst)
	}
}

func TestLoadConfigMissingRequiredValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OIDC_ISSUER_URL", "")
	t.Setenv("OIDC_CLIENT_ID", "")
	t.Setenv("OIDC_REDIRECT_URL", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected missing env error")
	}
	for _, name := range []string{"DATABASE_URL", "OIDC_ISSUER_URL", "OIDC_CLIENT_ID", "OIDC_REDIRECT_URL"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("expected error to mention %s, got %v", name, err)
		}
	}
}

func TestLoadMCPConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable")
	t.Setenv("MCP_OWNER_USER_ID", "11111111-1111-1111-1111-111111111111")
	t.Setenv("NOTE_CACHE_TTL", "90s")

	cfg, err := LoadMCPConfig()
	if err != nil {
		t.Fatalf("unexpected mcp config error: %v", err)
	}
	if cfg.OwnerUserID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected owner user id: %q", cfg.OwnerUserID)
	}
	if cfg.NoteCacheTTL != 90*time.Second {
		t.Fatalf("unexpected note cache ttl: %s", cfg.NoteCacheTTL)
	}
}

func TestLoadMCPConfigMissingRequiredValues(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("MCP_OWNER_USER_ID", "")

	_, err := LoadMCPConfig()
	if err == nil {
		t.Fatal("expected missing env error")
	}
	for _, name := range []string{"DATABASE_URL", "MCP_OWNER_USER_ID"} {
		if !strings.Contains(err.Error(), name) {
			t.Fatalf("expected error to mention %s, got %v", name, err)
		}
	}
}

func TestHelperEnvParsers(t *testing.T) {
	t.Setenv("APP_ENV", " production ")
	if got := envOrDefault("APP_ENV", "development"); got != "production" {
		t.Fatalf("expected trimmed env value, got %q", got)
	}

	t.Setenv("BAD_DURATION", "nope")
	if got := parseDurationEnv("BAD_DURATION", time.Minute); got != time.Minute {
		t.Fatalf("expected fallback duration, got %s", got)
	}

	t.Setenv("GOOD_BOOL", "true")
	if !parseBoolEnv("GOOD_BOOL", false) {
		t.Fatal("expected parsed bool")
	}
	t.Setenv("BAD_BOOL", "nope")
	if got := parseBoolEnv("BAD_BOOL", true); !got {
		t.Fatal("expected bool fallback")
	}

	t.Setenv("GOOD_INT", "9")
	if got := parseIntEnv("GOOD_INT", 1); got != 9 {
		t.Fatalf("expected parsed int, got %d", got)
	}

	t.Setenv("BAD_INT", "nope")
	if got := parseIntEnv("BAD_INT", 4); got != 4 {
		t.Fatalf("expected int fallback, got %d", got)
	}

	t.Setenv("GOOD_FLOAT", "1.75")
	if got := parseFloatEnv("GOOD_FLOAT", 1); got != 1.75 {
		t.Fatalf("expected parsed float, got %v", got)
	}

	t.Setenv("BAD_FLOAT", "nope")
	if got := parseFloatEnv("BAD_FLOAT", 2.5); got != 2.5 {
		t.Fatalf("expected float fallback, got %v", got)
	}

	got := splitCSV(" one, two , ,three ")
	if len(got) != 3 || got[1] != "two" {
		t.Fatalf("unexpected csv split result: %#v", got)
	}
}
