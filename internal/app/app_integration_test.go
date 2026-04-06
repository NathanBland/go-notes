package app

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestNewBuildsRealApplicationWithDockerServices(t *testing.T) {
	if os.Getenv("RUN_INTEGRATION") != "1" {
		t.Skip("integration coverage only")
	}

	t.Setenv("DATABASE_URL", getenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable"))
	t.Setenv("VALKEY_ADDR", getenv("VALKEY_ADDR", "127.0.0.1:6379"))
	t.Setenv("OIDC_ISSUER_URL", "https://issuer.example")
	t.Setenv("OIDC_CLIENT_ID", "client-id")
	t.Setenv("OIDC_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	app, err := New(ctx)
	if err != nil {
		t.Fatalf("expected application to build, got %v", err)
	}
	if app.Server == nil || app.DBPool == nil || app.CacheClient == nil {
		t.Fatalf("expected app dependencies to be initialized, got %+v", app)
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected application shutdown to succeed, got %v", err)
	}
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
