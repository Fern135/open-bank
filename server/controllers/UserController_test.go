package controllers

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewUser(t *testing.T) {
	tests := []struct {
		name string
		body string
		want int
	}{
		{"valid", `{"name":"Ana","email":"ana@example.com"}`, http.StatusCreated},
		{"missing email", `{"name":"Ana"}`, http.StatusBadRequest},
		{"client sets id", `{"id":99,"name":"Ana","email":"ana@example.com"}`, http.StatusBadRequest},
		{"not json", `Hello World`, http.StatusBadRequest},
		{"too large", `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(tt.body))
			w := httptest.NewRecorder()

			NewUserController().NewUser(w, r)

			if w.Code != tt.want {
				t.Errorf("got status %d, want %d", w.Code, tt.want)
			}
		})
	}
}
