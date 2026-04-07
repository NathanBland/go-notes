package httpapi

import (
	"log/slog"
	"net/http"

	"github.com/nathanbland/go-notes/internal/auth"
	"github.com/nathanbland/go-notes/internal/notes"
)

// Dependencies collects the concrete services and settings the HTTP layer needs.
type Dependencies struct {
	Logger              *slog.Logger
	AuthService         *auth.Service
	NotesService        *notes.Service
	SessionCookieName   string
	SessionCookieSecure bool
	ThrottleRequestsPS  float64
	ThrottleBurst       int
}

// API keeps handler dependencies together after the public constructor has
// validated and copied them.
type API struct {
	logger              *slog.Logger
	authService         *auth.Service
	notesService        *notes.Service
	sessionCookieName   string
	sessionCookieSecure bool
}

// NewHandler builds the versioned API routes and wraps them in common
// middleware for recovery, request IDs, and structured logging.
func NewHandler(deps Dependencies) http.Handler {
	api := &API{
		logger:              deps.Logger,
		authService:         deps.AuthService,
		notesService:        deps.NotesService,
		sessionCookieName:   deps.SessionCookieName,
		sessionCookieSecure: deps.SessionCookieSecure,
	}
	throttle := throttleMiddleware(deps.ThrottleRequestsPS, deps.ThrottleBurst)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", api.handleHome)
	mux.Handle("GET /shared/{slug}", throttle(http.HandlerFunc(api.handlePublicSharedNote)))
	mux.HandleFunc("GET /api/v1/healthz", api.handleHealth)
	mux.Handle("GET /api/v1/auth/login", throttle(http.HandlerFunc(api.handleLogin)))
	mux.Handle("GET /api/v1/auth/callback", throttle(http.HandlerFunc(api.handleCallback)))
	mux.Handle("GET /api/v1/auth/me", api.requireUser(http.HandlerFunc(api.handleMe)))
	mux.Handle("POST /api/v1/auth/logout", api.requireUser(http.HandlerFunc(api.handleLogout)))
	mux.Handle("POST /app/logout", api.requireUser(http.HandlerFunc(api.handleUILogout)))
	mux.Handle("POST /app/saved-queries", api.requireUser(http.HandlerFunc(api.handleUICreateSavedQuery)))
	mux.Handle("POST /app/saved-queries/{id}/delete", api.requireUser(http.HandlerFunc(api.handleUIDeleteSavedQuery)))
	mux.Handle("POST /app/tags/rename", api.requireUser(http.HandlerFunc(api.handleUIRenameTag)))
	mux.Handle("POST /app/notes", api.requireUser(http.HandlerFunc(api.handleUICreateNote)))
	mux.Handle("GET /app/notes/{id}", api.requireUser(http.HandlerFunc(api.handleUIShowNote)))
	mux.Handle("GET /app/notes/{id}/edit", api.requireUser(http.HandlerFunc(api.handleUIEditNote)))
	mux.Handle("POST /app/notes/{id}", api.requireUser(http.HandlerFunc(api.handleUIUpdateNote)))
	mux.Handle("POST /api/v1/notes", api.requireUser(http.HandlerFunc(api.handleCreateNote)))
	mux.Handle("GET /api/v1/notes", api.requireUser(http.HandlerFunc(api.handleListNotes)))
	mux.Handle("GET /api/v1/notes/{id}", api.requireUser(http.HandlerFunc(api.handleGetNote)))
	mux.Handle("PATCH /api/v1/notes/{id}", api.requireUser(http.HandlerFunc(api.handlePatchNote)))
	mux.Handle("DELETE /api/v1/notes/{id}", api.requireUser(http.HandlerFunc(api.handleDeleteNote)))
	mux.Handle("GET /api/v1/saved-queries", api.requireUser(http.HandlerFunc(api.handleListSavedQueries)))
	mux.Handle("POST /api/v1/saved-queries", api.requireUser(http.HandlerFunc(api.handleCreateSavedQuery)))
	mux.Handle("DELETE /api/v1/saved-queries/{id}", api.requireUser(http.HandlerFunc(api.handleDeleteSavedQuery)))
	mux.Handle("POST /api/v1/tags/rename", api.requireUser(http.HandlerFunc(api.handleRenameTag)))
	mux.Handle("GET /api/v1/notes/shared/{slug}", throttle(http.HandlerFunc(api.handleGetSharedNote)))
	mux.HandleFunc("/", api.handleNotFound)

	return chain(mux,
		recoverMiddleware(deps.Logger),
		requestIDMiddleware(),
		loggingMiddleware(deps.Logger),
	)
}
