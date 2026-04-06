package app

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nathanbland/go-notes/internal/httpapi"
	cacheclient "github.com/nathanbland/go-notes/internal/platform/cache"
	oidcclient "github.com/nathanbland/go-notes/internal/platform/oidc"
)

func TestNewUsesInjectedConstructors(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable")
	t.Setenv("OIDC_ISSUER_URL", "http://issuer.example")
	t.Setenv("OIDC_CLIENT_ID", "client-id")
	t.Setenv("OIDC_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback")

	origDB := newDBPool
	origCache := newCacheClient
	origOIDC := newOIDCClient
	origHandler := newHTTPHandler
	t.Cleanup(func() {
		newDBPool = origDB
		newCacheClient = origCache
		newOIDCClient = origOIDC
		newHTTPHandler = origHandler
	})

	newDBPool = func(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) { return nil, nil }
	newCacheClient = func(address, password string) (*cacheclient.Client, error) { return nil, nil }
	newOIDCClient = func(ctx context.Context, config oidcclient.Config) (*oidcclient.Client, error) { return nil, nil }
	newHTTPHandler = func(deps httpapi.Dependencies) http.Handler { return http.NotFoundHandler() }

	app, err := New(context.Background())
	if err != nil {
		t.Fatalf("unexpected constructor error: %v", err)
	}
	if app.Server == nil {
		t.Fatal("expected server to be initialized")
	}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("unexpected shutdown error: %v", err)
	}
}

func TestNewReturnsDependencyErrors(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/go_notes?sslmode=disable")
	t.Setenv("OIDC_ISSUER_URL", "http://issuer.example")
	t.Setenv("OIDC_CLIENT_ID", "client-id")
	t.Setenv("OIDC_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback")

	origDB := newDBPool
	origCache := newCacheClient
	origOIDC := newOIDCClient
	origHandler := newHTTPHandler
	t.Cleanup(func() {
		newDBPool = origDB
		newCacheClient = origCache
		newOIDCClient = origOIDC
		newHTTPHandler = origHandler
	})

	dbErr := errors.New("db failed")
	newDBPool = func(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) { return nil, dbErr }
	if _, err := New(context.Background()); !errors.Is(err, dbErr) {
		t.Fatalf("expected db error, got %v", err)
	}

	newDBPool = func(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) { return nil, nil }
	cacheErr := errors.New("cache failed")
	newCacheClient = func(address, password string) (*cacheclient.Client, error) { return nil, cacheErr }
	if _, err := New(context.Background()); !errors.Is(err, cacheErr) {
		t.Fatalf("expected cache error, got %v", err)
	}

	newCacheClient = func(address, password string) (*cacheclient.Client, error) { return &cacheclient.Client{}, nil }
	oidcErr := errors.New("oidc failed")
	newOIDCClient = func(ctx context.Context, config oidcclient.Config) (*oidcclient.Client, error) { return nil, oidcErr }
	newHTTPHandler = func(deps httpapi.Dependencies) http.Handler { return http.NotFoundHandler() }
	if _, err := New(context.Background()); !errors.Is(err, oidcErr) {
		t.Fatalf("expected oidc error, got %v", err)
	}
}

func TestNewReturnsConfigErrors(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("OIDC_ISSUER_URL", "http://issuer.example")
	t.Setenv("OIDC_CLIENT_ID", "client-id")
	t.Setenv("OIDC_REDIRECT_URL", "http://localhost:8080/api/v1/auth/callback")

	if _, err := New(context.Background()); err == nil {
		t.Fatal("expected missing required env error")
	}
}

func TestShutdownHandlesCanceledContext(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })

	releaseHandler := make(chan struct{})
	handlerStarted := make(chan struct{})
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(handlerStarted)
			<-releaseHandler
			_, _ = w.Write([]byte("ok"))
		}),
	}
	go func() {
		_ = server.Serve(listener)
	}()

	responseDone := make(chan struct{})
	go func() {
		defer close(responseDone)
		resp, reqErr := http.Get("http://" + listener.Addr().String())
		if reqErr == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}
	}()
	<-handlerStarted

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	app := &Application{Server: server}
	if err := app.Shutdown(ctx); !errors.Is(err, context.DeadlineExceeded) {
		close(releaseHandler)
		t.Fatalf("expected shutdown to surface deadline exceeded, got %v", err)
	}
	close(releaseHandler)
	<-responseDone
}

func TestShutdownWithNilDependencies(t *testing.T) {
	app := &Application{}
	if err := app.Shutdown(context.Background()); err != nil {
		t.Fatalf("expected nil dependency shutdown to succeed, got %v", err)
	}
}
