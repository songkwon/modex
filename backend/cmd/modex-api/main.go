package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"modex/backend/internal/api"
	"modex/backend/internal/application"
	"modex/backend/internal/config"
	"modex/backend/internal/repository"
	"modex/backend/internal/vectorstore"
)

func analyticsSource() string {
	if api.PosthogConfigured() {
		return "built-in + PostHog (project_id=" + os.Getenv("POSTHOG_PROJECT_ID") + ", host=" + api.PosthogHost() + ")"
	}
	return "built-in"
}

func main() {
	if _, err := config.Load(); err != nil {
		log.Fatalf("load application config: %v", err)
	}
	addr := ":" + env("PORT", "8671")

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatalf("DATABASE_URL is required: modex-api persists all state in PostgreSQL")
	}
	repository := openRepository(databaseURL)

	vectorCtx, vectorCancel := context.WithTimeout(context.Background(), 10*time.Second)
	vectors, err := vectorstore.Open(vectorCtx, databaseURL)
	vectorCancel()
	if err != nil {
		log.Fatalf("open PostgreSQL vector store: %v", err)
	}
	defer vectors.Close()
	log.Printf("embedding store: PostgreSQL/pgvector")

	appSvc, err := application.NewConfigured(repository, vectors, repository)
	if err != nil {
		log.Fatalf("initialize application: %v", err)
	}
	defer appSvc.Close()
	srv := api.NewWithApplication(appSvc)
	log.Printf("analytics source: %s", analyticsSource())

	handler := srv.Handler()
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: envDuration("HTTP_READ_HEADER_TIMEOUT", 5*time.Second),
		ReadTimeout:       envDuration("HTTP_READ_TIMEOUT", 30*time.Second),
		WriteTimeout:      envDuration("HTTP_WRITE_TIMEOUT", 60*time.Second),
		IdleTimeout:       envDuration("HTTP_IDLE_TIMEOUT", 2*time.Minute),
		MaxHeaderBytes:    envInt("HTTP_MAX_HEADER_BYTES", 1<<20),
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("modex-api listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

func openRepository(databaseURL string) *repository.PostgresRepository {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := repository.OpenPostgres(ctx, databaseURL)
	if err != nil {
		log.Fatalf("open PostgreSQL business store: %v", err)
	}
	log.Printf("business store: PostgreSQL (request-level reads and writes)")
	return repo
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if parsed, err := time.ParseDuration(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}

func envInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.Atoi(value); err == nil && parsed > 0 {
			return parsed
		}
	}
	return fallback
}
