package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"api/cache"
	"api/config"
	"api/controllers"
	"api/database"
	"api/routes"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.New()
	if err != nil {
		log.Fatal(err)
	}

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatal(err)
	}

	redis, err := cache.Connect(ctx, cfg.RedisURL)
	if err != nil {
		log.Fatal(err)
	}
	defer redis.Close()

	router, err := routes.Router(&controllers.Deps{DB: db, Cache: redis, Cfg: cfg})
	if err != nil {
		log.Fatal(err)
	}

	// the zero values mean no timeouts, which lets slow clients hold connections open
	server := &http.Server{
		Addr:              cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}

	go func() {
		<-ctx.Done()
		// let requests in progress finish, so a posting isn't cut off mid-way
		shutdown, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			slog.Error("shutdown failed", "error", err)
		}
	}()

	// running the actual server, this only returns if it fails or is stopped
	slog.Info("listening", "port", cfg.Port)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
