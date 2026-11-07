package main

import (
	"api/config"
	"api/routes"
	"log"
	"net/http"
)

func main() {
	cfg := config.New()

	// running the actual server, this only returns if it fails
	log.Printf("listening on %s", cfg.Port)
	log.Fatal(http.ListenAndServe(cfg.Port, routes.Router()))
}
