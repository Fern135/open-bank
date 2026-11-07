package config

import "os"

// defaultPort matches the API_URL the web app expects in development.
const defaultPort = "3001"

// Config holds the settings the server reads at startup.
type Config struct {
	Port string // listen address in the form ":3001"
}

// New builds the Config from environment variables.
func New() *Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	return &Config{
		// net/http needs the leading colon, PORT is just the number
		Port: ":" + port,
	}
}
