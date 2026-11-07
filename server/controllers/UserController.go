package controllers

import (
	"encoding/json"
	"net/http"
)

// maxBodyBytes caps request bodies so a huge upload can't exhaust memory.
const maxBodyBytes = 1 << 20 // 1 MB

// UserController groups the handlers for the /users routes.
type UserController struct{}

// User is what the API sends back. Never add a password field here without
// the `json:"-"` tag, or it will appear in responses.
type User struct {
	ID       int    `json:"id"`
	FullName string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"-"` // never expose password
	Address  string `json:"address"`
	Phone    string `json:"phone"`
}

// createUserRequest is what clients may send. It's separate from User so a
// client can't choose its own ID.
type createUserRequest struct {
	UserName string `json:"name"`
	Email    string `json:"email"`
}

// NewUserController returns a ready to use UserController.
func NewUserController() *UserController {
	return &UserController{}
}

// NewUser handles POST /users: it validates the JSON body and returns the
// new user with a 201.
func (uc *UserController) NewUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)

	var req createUserRequest
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields() // reject fields we don't expect, like "id"
	if err := dec.Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserName == "" || req.Email == "" {
		http.Error(w, "name and email are required", http.StatusBadRequest)
		return
	}

	// shortcut: nothing is saved and the ID is fake, replace this with a
	// database insert that returns the real ID.
	user := User{ID: 1, FullName: req.UserName, Email: req.Email}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user)
}
