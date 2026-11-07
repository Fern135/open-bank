package routes

import (
	"api/controllers"
	// "api/middleware" // use this for functions that require authentication.
	"net/http"
)

// Router registers every API route and returns the handler the server runs.
func Router() *http.ServeMux {
	router := http.NewServeMux()

	// user routes functions
	userController := controllers.NewUserController()

	// public, people have no session yet when they sign up
	router.HandleFunc("POST /users", userController.NewUser)

	return router
}
