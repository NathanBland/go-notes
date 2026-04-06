package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/nathanbland/go-notes/internal/auth"
)

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	redirectURL, err := a.authService.BeginLogin(r.Context(), safeRedirectTarget(r.URL.Query().Get("redirect_to")))
	if err != nil {
		a.logger.Error("failed to begin login",
			"request_id", requestIDFromContext(r.Context()),
			"error", err,
		)
		writeError(w, http.StatusInternalServerError, "login_failed", "failed to begin login", nil)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (a *API) handleCallback(w http.ResponseWriter, r *http.Request) {
	if oidcError := r.URL.Query().Get("error"); oidcError != "" {
		writeError(w, http.StatusUnauthorized, "oidc_error", "identity provider returned an authentication error", map[string]string{"error": oidcError})
		return
	}
	user, session, pending, err := a.authService.FinishLogin(r.Context(), r.URL.Query().Get("state"), r.URL.Query().Get("code"))
	if err != nil {
		a.logger.Error("failed to finish login",
			"request_id", requestIDFromContext(r.Context()),
			"error", err,
		)
		status := http.StatusUnauthorized
		code := "authentication_failed"
		if errors.Is(err, auth.ErrInvalidState) || errors.Is(err, auth.ErrInvalidCode) {
			status = http.StatusBadRequest
			code = "invalid_callback"
		}
		writeError(w, status, code, "failed to finish login", nil)
		return
	}
	http.SetCookie(w, a.sessionCookie(session.ID, session.ExpiresAt))
	if target := safeRedirectTarget(pending.RedirectTo); target != "" {
		http.Redirect(w, r, target, http.StatusSeeOther)
		return
	}
	writeData(w, http.StatusOK, user)
}

func (a *API) handleMe(w http.ResponseWriter, r *http.Request) {
	user, ok := userFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
		return
	}
	writeData(w, http.StatusOK, user)
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	sessionCookie, _ := r.Cookie(a.sessionCookieName)
	if sessionCookie != nil {
		_ = a.authService.Logout(r.Context(), sessionCookie.Value)
	}
	http.SetCookie(w, a.expiredSessionCookie())
	writeData(w, http.StatusOK, map[string]bool{"logged_out": true})
}

func (a *API) requireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionCookie, err := r.Cookie(a.sessionCookieName)
		if err != nil || sessionCookie.Value == "" {
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
			return
		}
		user, _, authErr := a.authService.Authenticate(r.Context(), sessionCookie.Value)
		if authErr != nil {
			http.SetCookie(w, a.expiredSessionCookie())
			writeError(w, http.StatusUnauthorized, "unauthorized", "authentication required", nil)
			return
		}
		next.ServeHTTP(w, r.WithContext(withUser(r.Context(), user)))
	})
}

func (a *API) sessionCookie(sessionID string, expiresAt time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     a.sessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   a.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func (a *API) expiredSessionCookie() *http.Cookie {
	return &http.Cookie{
		Name:     a.sessionCookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   a.sessionCookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
}

func safeRedirectTarget(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") || strings.HasPrefix(trimmed, "//") {
		return ""
	}
	return trimmed
}
