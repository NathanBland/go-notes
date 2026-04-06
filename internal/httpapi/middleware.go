package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func chain(handler http.Handler, middleware ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler
}

func requestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			buf := make([]byte, 8)
			_, _ = rand.Read(buf)
			requestID := hex.EncodeToString(buf)
			next.ServeHTTP(w, r.WithContext(withRequestID(r.Context(), requestID)))
		})
	}
}

func loggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			logger.Info("request complete",
				"request_id", requestIDFromContext(r.Context()),
				"method", r.Method,
				"path", r.URL.Path,
				"status", recorder.status,
				"duration", time.Since(started).String(),
			)
		})
	}
}

func recoverMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered", "request_id", requestIDFromContext(r.Context()), "panic", recovered)
					writeError(w, http.StatusInternalServerError, "internal_error", "internal server error", nil)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

type clientLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

func throttleMiddleware(requestsPerSecond float64, burst int) func(http.Handler) http.Handler {
	if requestsPerSecond <= 0 || burst <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	limiters := map[string]*clientLimiter{}
	var mu sync.Mutex
	lastCleanup := time.Now().UTC()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := throttleKey(r)
			now := time.Now().UTC()

			mu.Lock()
			if now.Sub(lastCleanup) >= time.Minute {
				for candidate, limiter := range limiters {
					if now.Sub(limiter.lastSeen) > 10*time.Minute {
						delete(limiters, candidate)
					}
				}
				lastCleanup = now
			}

			limiter, ok := limiters[key]
			if !ok {
				limiter = &clientLimiter{
					limiter:  rate.NewLimiter(rate.Limit(requestsPerSecond), burst),
					lastSeen: now,
				}
				limiters[key] = limiter
			} else {
				limiter.lastSeen = now
			}
			allowed := limiter.limiter.Allow()
			mu.Unlock()

			if !allowed {
				writeError(w, http.StatusTooManyRequests, "rate_limited", "too many requests", nil)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func throttleKey(r *http.Request) string {
	// We prefer the first X-Forwarded-For value when present so the middleware
	// still works behind a reverse proxy. For direct local development, fall
	// back to the socket peer from RemoteAddr.
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		first := strings.TrimSpace(strings.Split(forwarded, ",")[0])
		if first != "" {
			return first
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	if strings.TrimSpace(r.RemoteAddr) != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
