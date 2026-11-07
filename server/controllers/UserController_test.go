package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// These cases are all rejected before the database is touched. The success
// path is covered against real services in routes/Routes_test.go.
func TestNewUserValidation(t *testing.T) {
	const password = "correct horse battery"

	tests := []struct {
		name string
		body string
		want int
	}{
		{"missing email", `{"name":"Ana","password":"` + password + `"}`, http.StatusBadRequest},
		{"bad email", `{"name":"Ana","email":"ana","password":"` + password + `"}`, http.StatusBadRequest},
		{"email with display name", `{"name":"Ana","email":"Ana <ana@example.com>","password":"` + password + `"}`, http.StatusBadRequest},
		{"missing name", `{"email":"ana@example.com","password":"` + password + `"}`, http.StatusBadRequest},
		{"short password", `{"name":"Ana","email":"ana@example.com","password":"short"}`, http.StatusBadRequest},
		{"client sets id", `{"id":"99","name":"Ana","email":"ana@example.com","password":"` + password + `"}`, http.StatusBadRequest},
		{"not json", `Hello World`, http.StatusBadRequest},
		{"too large", `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			NewUserController(&Deps{}).NewUser(w, r)

			if w.Code != tt.want {
				t.Errorf("got status %d, want %d", w.Code, tt.want)
			}
		})
	}
}
