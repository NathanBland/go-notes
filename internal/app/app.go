package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/httpapi"
	"github.com/nathanbland/go-notes/internal/notes"
	cacheclient "github.com/nathanbland/go-notes/internal/platform/cache"
	"github.com/nathanbland/go-notes/internal/platform/db"
	oidcclient "github.com/nathanbland/go-notes/internal/platform/oidc"
)

var (
	newDBPool      = db.NewPool
	newCacheClient = cacheclient.New
	newOIDCClient  = oidcclient.New
	newHTTPHandler = httpapi.NewHandler
)

// Application owns the long-lived process dependencies so startup and shutdown
// happen in one place.
type Application struct {
	Config      Config
	Logger      *slog.Logger
	Server      *http.Server
	DBPool      *pgxpool.Pool
	CacheClient *cacheclient.Client
}

// New wires the application from environment configuration.
// It opens PostgreSQL and Valkey connections, constructs services, and builds
// the HTTP server without starting it yet.
func New(ctx context.Context) (*Application, error) {
	config, err := LoadConfig()
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	dbPool, err := newDBPool(ctx, config.DatabaseURL)
	if err != nil {
		return nil, err
	}

	cache, err := newCacheClient(config.ValkeyAddr, config.ValkeyPassword)
	if err != nil {
		if dbPool != nil {
			dbPool.Close()
		}
		return nil, err
	}

	oidcProvider, err := newOIDCClient(ctx, config.OIDC)
	if err != nil {
		if cache != nil {
			_ = cache.Close()
		}
		if dbPool != nil {
			dbPool.Close()
		}
		return nil, err
	}

	store := db.NewStore(dbPool)
	sessionStore := auth.NewCacheSessionStore(cache)
	authService := auth.NewService(store, oidcProvider, sessionStore, config.SessionTTL, config.OIDCStateTTL)
	noteService := notes.NewService(store, cache, config.NoteCacheTTL, config.ListCacheTTL)

	handler := newHTTPHandler(httpapi.Dependencies{
		Logger:              logger,
		AuthService:         authService,
		NotesService:        noteService,
		SessionCookieName:   config.SessionCookieName,
		SessionCookieSecure: config.SessionCookieSecure,
		ThrottleRequestsPS:  config.ThrottleRequestsPS,
		ThrottleBurst:       config.ThrottleBurst,
	})

	server := &http.Server{
		Addr:         config.HTTPAddr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  2 * time.Minute,
	}

	return &Application{
		Config:      config,
		Logger:      logger,
		Server:      server,
		DBPool:      dbPool,
		CacheClient: cache,
	}, nil
}

// Shutdown closes the HTTP server and infrastructure clients in dependency
// order so in-flight work gets a chance to finish cleanly.
func (a *Application) Shutdown(ctx context.Context) error {
	var result error
	if a.Server != nil {
		if err := a.Server.Shutdown(ctx); err != nil {
			result = errors.Join(result, err)
		}
	}
	if a.CacheClient != nil {
		if err := a.CacheClient.Close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	if a.DBPool != nil {
		a.DBPool.Close()
	}
	return result
}
