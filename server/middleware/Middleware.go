package middleware

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"api/auth"
	"api/cache"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AccessCookie is the name of the cookie holding the access JWT.
const AccessCookie = "access_token"

type contextKey int

const (
	identityKey contextKey = iota
	clientIPKey
)

type identity struct {
	userID    string
	sessionID string
}

// UserID returns the signed-in user. It's only set behind RequireAuth.
func UserID(ctx context.Context) string {
	id, _ := ctx.Value(identityKey).(identity)
	return id.userID
}

// SessionID returns the current session. It's only set behind RequireAuth.
func SessionID(ctx context.Context) string {
	id, _ := ctx.Value(identityKey).(identity)
	return id.sessionID
}

// ClientIP returns the address set by RealIP.
func ClientIP(ctx context.Context) string {
	ip, _ := ctx.Value(clientIPKey).(string)
	return ip
}

// RequireAuth only lets requests with a valid JWT and a live session reach
// next. The session check is what makes signing out take effect immediately
// instead of when the JWT expires.
func RequireAuth(secret []byte, db *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(AccessCookie)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
				return
			}
			claims, err := auth.ParseAccessToken(secret, cookie.Value)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
				return
			}

			var live bool
			err = db.QueryRow(r.Context(), `
				SELECT EXISTS (
					SELECT 1 FROM sessions
					WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL AND expires_at > now()
				)`, claims.SessionID, claims.Subject).Scan(&live)
			if err != nil {
				slog.Error("session check failed", "error", err)
				writeError(w, http.StatusServiceUnavailable, "unavailable", "Try again in a moment.")
				return
			}
			if !live {
				writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in to continue.")
				return
			}

			ctx := context.WithValue(r.Context(), identityKey, identity{claims.Subject, claims.SessionID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ByIP keys a rate limit on the client address. The address can be forged on
// public routes, so it's best effort and never the only protection.
func ByIP(r *http.Request) string { return ClientIP(r.Context()) }

// ByUser keys a rate limit on the signed-in user, which can't be forged. It
// only works behind RequireAuth.
func ByUser(r *http.Request) string { return UserID(r.Context()) }

// Everyone keys a rate limit on the whole site. Password hashing is costly by
// design, so this caps what forged addresses can make the server do.
func Everyone(*http.Request) string { return "all" }

// RateLimit allows limit requests per key in each window. If Redis is down it
// refuses the request, because skipping the limit would leave sign-in open to
// password guessing.
func RateLimit(c *cache.Cache, name string, limit int64, window time.Duration, key func(*http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			allowed, err := c.Allow(r.Context(), "rate:"+name+":"+key(r), limit, window)
			if err != nil {
				slog.Error("rate limit check failed", "error", err)
				writeError(w, http.StatusServiceUnavailable, "unavailable", "Try again in a moment.")
				return
			}
			if !allowed {
				writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many attempts. Wait a few minutes and try again.")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RealIP records the client address. X-Forwarded-For can be forged by anyone,
// so it's only read when a proxy we control sets it.
func RealIP(trustProxy bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			if forwarded := r.Header.Get("X-Forwarded-For"); trustProxy && forwarded != "" {
				first, _, _ := strings.Cut(forwarded, ",")
				ip = strings.TrimSpace(first)
			}
			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), clientIPKey, ip)))
		})
	}
}

// SecureHeaders stops browsers and proxies from storing account data.
func SecureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(status int) {
	s.status = status
	s.ResponseWriter.WriteHeader(status)
}

// Logger writes one line per request. It logs the route, never bodies or
// cookies, so passwords and tokens can't end up in logs.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r)
		slog.Info("request", "method", r.Method, "path", r.URL.Path, "status", recorder.status, "duration", time.Since(start))
	})
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"code": code, "error": message})
}
