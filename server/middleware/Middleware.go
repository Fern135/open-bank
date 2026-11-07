package middleware

import "net/http"

// RequireAuth only lets requests with a valid session reach next.
//
// shortcut: rejects every request because sessions don't exist yet, add the
// scs session check here before wrapping any route with this.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// no valid session: respond with 401 and return
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
}
