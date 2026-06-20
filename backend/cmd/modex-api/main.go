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
	"modex/backend/internal/store"
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
	st, repository := loadStore(databaseURL)

	vectorCtx, vectorCancel := context.WithTimeout(context.Background(), 10*time.Second)
	vectors, err := vectorstore.Open(vectorCtx, databaseURL)
	vectorCancel()
	if err != nil {
		log.Fatalf("open PostgreSQL vector store: %v", err)
	}
	defer vectors.Close()
	log.Printf("embedding store: PostgreSQL/pgvector")

	appSvc, err := application.NewConfigured(st, vectors, repository)
	if err != nil {
		log.Fatalf("initialize application: %v", err)
	}
	defer appSvc.Close()
	srv := api.NewWithApplication(appSvc)
	initCtx, initCancel := context.WithTimeout(context.Background(), 10*time.Second)
	if err := appSvc.Save(initCtx); err != nil {
		log.Printf("initial PostgreSQL relational save failed: %v", err)
	}
	initCancel()

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

	// Periodic + graceful-shutdown persistence when PostgreSQL is configured.
	// The autosave goroutine gets its own stop channel (closed by main after the
	// OS signal). Sharing the signal channel would race: a signal delivered to a
	// channel wakes only ONE receiver, so the autosave goroutine could consume it
	// and leave main's `<-stop` blocked forever, skipping the final save.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	autosaveStop := make(chan struct{})
	if appSvc.HasRepository() {
		go autosavePostgres(appSvc, autosaveStop)
	}

	go func() {
		log.Printf("modex-api listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-stop
	close(autosaveStop)
	if appSvc.HasRepository() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := appSvc.Save(ctx); err != nil {
			log.Printf("final PostgreSQL store save failed: %v", err)
		} else {
			log.Printf("saved relational store to PostgreSQL")
		}
		cancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

func loadStore(databaseURL string) (*store.Store, *repository.PostgresRepository) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	repo, err := repository.OpenPostgres(ctx, databaseURL)
	if err != nil {
		log.Fatalf("open PostgreSQL business store: %v", err)
	}
	legacyDataDir := firstNonEmpty(os.Getenv("LEGACY_DATA_DIR"), os.Getenv("DATA_DIR"))
	st, migrated, err := repo.LoadOrMigrate(ctx, legacyDataDir)
	if err != nil {
		repo.Close()
		log.Fatalf("load PostgreSQL business store: %v", err)
	}
	if migrated {
		log.Printf("migrated legacy store data into PostgreSQL relational tables")
	} else {
		log.Printf("loaded business store from PostgreSQL")
	}
	return st, repo
}

func autosavePostgres(appSvc *application.Service, stop <-chan struct{}) {
	interval := 60 * time.Second
	if v := os.Getenv("DATA_SAVE_INTERVAL_SECONDS"); v != "" {
		if d, err := time.ParseDuration(v + "s"); err == nil && d > 0 {
			interval = d
		}
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if err := appSvc.Save(ctx); err != nil {
				log.Printf("PostgreSQL store save failed: %v", err)
			}
			cancel()
		case <-stop:
			return
		}
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
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
