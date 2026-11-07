package routes

import (
	"net/http"
	"time"

	"api/controllers"
	"api/middleware"
)

// Router registers every API route and returns the handler the server runs.
func Router(deps *controllers.Deps) (http.Handler, error) {
	users := controllers.NewUserController(deps)
	accounts := controllers.NewAccountController(deps)
	transfers := controllers.NewTransferController(deps)
	authentication, err := controllers.NewAuthController(deps)
	if err != nil {
		return nil, err
	}

	guesses := middleware.RateLimit(deps.Cache, "auth", 20, 15*time.Minute, middleware.ByIP)
	siteWide := middleware.RateLimit(deps.Cache, "auth-all", 300, time.Minute, middleware.Everyone)
	userGuesses := middleware.RateLimit(deps.Cache, "user-auth", 20, 15*time.Minute, middleware.ByUser)
	money := middleware.RateLimit(deps.Cache, "money", 60, time.Minute, middleware.ByUser)
	refresh := middleware.RateLimit(deps.Cache, "refresh", 120, time.Minute, middleware.ByIP)

	limited := func(handler http.HandlerFunc, limits ...func(http.Handler) http.Handler) http.Handler {
		var wrapped http.Handler = handler
		for _, limit := range limits {
			wrapped = limit(wrapped)
		}
		return wrapped
	}

	// login required
	protected := http.NewServeMux()
	protected.HandleFunc("GET /me", users.Me)
	protected.HandleFunc("PATCH /me", users.UpdateMe)
	protected.Handle("POST /me/password", limited(users.ChangePassword, userGuesses))

	protected.HandleFunc("GET /sessions", authentication.ListSessions)
	protected.HandleFunc("DELETE /sessions/{id}", authentication.RevokeSession)
	protected.HandleFunc("GET /activity", authentication.Activity)
	protected.HandleFunc("POST /2fa/setup", authentication.SetupTwoFactor)
	protected.Handle("POST /2fa/enable", limited(authentication.EnableTwoFactor, userGuesses))
	protected.Handle("POST /2fa/disable", limited(authentication.DisableTwoFactor, userGuesses))

	protected.HandleFunc("GET /accounts", accounts.List)
	protected.HandleFunc("POST /accounts", accounts.Create)
	protected.HandleFunc("GET /accounts/{id}", accounts.Get)
	protected.HandleFunc("GET /accounts/{id}/entries", accounts.Entries)
	protected.HandleFunc("GET /accounts/{id}/statements/{month}", accounts.Statement)
	protected.HandleFunc("GET /entries", accounts.RecentEntries)

	protected.Handle("POST /deposits", limited(transfers.Deposit, money))
	protected.Handle("POST /transfers", limited(transfers.Transfer, money))

	// public, with everything else falling through to protected
	api := http.NewServeMux()
	api.Handle("POST /users", limited(users.NewUser, guesses, siteWide))
	api.Handle("POST /auth/login", limited(authentication.Login, guesses, siteWide))
	api.Handle("POST /auth/refresh", limited(authentication.Refresh, refresh))
	api.HandleFunc("POST /auth/logout", authentication.Logout)
	api.Handle("/", middleware.RequireAuth(deps.Cfg.JWTSecret, deps.DB)(protected))

	root := http.NewServeMux()
	root.Handle("/api/v1/", http.StripPrefix("/api/v1", api))
	root.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		if err := deps.DB.Ping(r.Context()); err != nil {
			http.Error(w, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// Rejects cross-site writes, a second layer behind the SameSite cookies.
	csrf := http.NewCrossOriginProtection()
	return middleware.Logger(middleware.RealIP(deps.Cfg.TrustProxy)(middleware.SecureHeaders(csrf.Handler(root)))), nil
}
