package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	oidcclient "github.com/nathanbland/go-notes/internal/platform/oidc"
)

// Config holds the environment-driven settings needed to run the API locally or
// in containers.
type Config struct {
	AppEnv              string
	HTTPAddr            string
	BaseURL             string
	DatabaseURL         string
	ValkeyAddr          string
	ValkeyPassword      string
	SessionCookieName   string
	SessionTTL          time.Duration
	OIDCStateTTL        time.Duration
	NoteCacheTTL        time.Duration
	ListCacheTTL        time.Duration
	ThrottleRequestsPS  float64
	ThrottleBurst       int
	SessionCookieSecure bool
	OIDC                oidcclient.Config
}

// MCPConfig keeps the smaller env surface needed by the local stdio MCP server.
type MCPConfig struct {
	DatabaseURL    string
	ValkeyAddr     string
	ValkeyPassword string
	NoteCacheTTL   time.Duration
	ListCacheTTL   time.Duration
	OwnerUserID    string
}

// LoadConfig reads environment variables, applies safe local-development
// defaults, and verifies the minimum required settings.
func LoadConfig() (Config, error) {
	config := Config{
		AppEnv:              envOrDefault("APP_ENV", "development"),
		HTTPAddr:            envOrDefault("HTTP_ADDR", ":8080"),
		BaseURL:             envOrDefault("BASE_URL", "http://localhost:8080"),
		DatabaseURL:         os.Getenv("DATABASE_URL"),
		ValkeyAddr:          envOrDefault("VALKEY_ADDR", "127.0.0.1:6379"),
		ValkeyPassword:      os.Getenv("VALKEY_PASSWORD"),
		SessionCookieName:   envOrDefault("SESSION_COOKIE_NAME", "go_notes_session"),
		SessionTTL:          parseDurationEnv("SESSION_TTL", 168*time.Hour),
		OIDCStateTTL:        parseDurationEnv("OIDC_STATE_TTL", 5*time.Minute),
		NoteCacheTTL:        parseDurationEnv("NOTE_CACHE_TTL", 5*time.Minute),
		ListCacheTTL:        parseDurationEnv("LIST_CACHE_TTL", 45*time.Second),
		ThrottleRequestsPS:  parseFloatEnv("THROTTLE_REQUESTS_PER_SECOND", 2),
		ThrottleBurst:       parseIntEnv("THROTTLE_BURST", 5),
		SessionCookieSecure: parseBoolEnv("SESSION_COOKIE_SECURE", false),
		OIDC: oidcclient.Config{
			IssuerURL:    os.Getenv("OIDC_ISSUER_URL"),
			ClientID:     os.Getenv("OIDC_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
			RedirectURL:  os.Getenv("OIDC_REDIRECT_URL"),
			Scopes:       splitCSV(envOrDefault("OIDC_SCOPES", "openid,profile,email")),
		},
	}

	missing := make([]string, 0)
	for _, pair := range []struct {
		name  string
		value string
	}{
		{"DATABASE_URL", config.DatabaseURL},
		{"OIDC_ISSUER_URL", config.OIDC.IssuerURL},
		{"OIDC_CLIENT_ID", config.OIDC.ClientID},
		{"OIDC_REDIRECT_URL", config.OIDC.RedirectURL},
	} {
		if strings.TrimSpace(pair.value) == "" {
			missing = append(missing, pair.name)
		}
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return config, nil
}

// LoadMCPConfig reads the env needed by the local MCP server.
// The owner UUID is required in this first slice because MCP auth is still a
// roadmap item and local development needs an explicit note owner scope.
func LoadMCPConfig() (MCPConfig, error) {
	config := MCPConfig{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		ValkeyAddr:     envOrDefault("VALKEY_ADDR", "127.0.0.1:6379"),
		ValkeyPassword: os.Getenv("VALKEY_PASSWORD"),
		NoteCacheTTL:   parseDurationEnv("NOTE_CACHE_TTL", 5*time.Minute),
		ListCacheTTL:   parseDurationEnv("LIST_CACHE_TTL", 45*time.Second),
		OwnerUserID:    strings.TrimSpace(os.Getenv("MCP_OWNER_USER_ID")),
	}

	missing := make([]string, 0)
	for _, pair := range []struct {
		name  string
		value string
	}{
		{"DATABASE_URL", config.DatabaseURL},
		{"MCP_OWNER_USER_ID", config.OwnerUserID},
	} {
		if strings.TrimSpace(pair.value) == "" {
			missing = append(missing, pair.name)
		}
	}
	if len(missing) > 0 {
		return MCPConfig{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return config, nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func parseDurationEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseBoolEnv(name string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseIntEnv(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseFloatEnv(name string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
