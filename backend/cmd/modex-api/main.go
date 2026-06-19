package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"modex/backend/internal/api"
	"modex/backend/internal/application"
	"modex/backend/internal/repository"
	"modex/backend/internal/store"
	"modex/backend/internal/vectorstore"
)

func analyticsSource() string {
	if api.PosthogConfigured() {
		return "PostHog (project_id=" + os.Getenv("POSTHOG_PROJECT_ID") + ", host=" + api.PosthogHost() + ")"
	}
	return "disabled (PostHog not configured)"
}

func main() {
	addr := ":" + env("PORT", "8671")

	databaseURL := os.Getenv("DATABASE_URL")
	st, repository := loadStore(databaseURL)
	var vectors *vectorstore.Postgres
	if databaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		var err error
		vectors, err = vectorstore.Open(ctx, databaseURL)
		cancel()
		if err != nil {
			log.Fatalf("open PostgreSQL vector store: %v", err)
		}
		defer vectors.Close()
		log.Printf("embedding store: PostgreSQL/pgvector")
	}
	appSvc := application.New(st, vectors, repository)
	defer appSvc.Close()
	srv := api.NewWithApplication(appSvc)
	if appSvc.HasRepository() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		if err := appSvc.Save(ctx); err != nil {
			log.Printf("initial PostgreSQL relational save failed: %v", err)
		}
		cancel()
	}

	log.Printf("analytics source: %s", analyticsSource())

	handler := srv.Handler()
	if appSvc.HasRepository() {
		handler = persistMutations(handler, appSvc)
	}
	httpServer := &http.Server{Addr: addr, Handler: handler}

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
	if databaseURL == "" {
		if os.Getenv("DATA_DIR") != "" {
			log.Printf("DATA_DIR is ignored without DATABASE_URL; starting with volatile in-memory store")
		} else {
			log.Printf("DATABASE_URL not set; starting with volatile in-memory store")
		}
		return store.New(), nil
	}
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

func persistMutations(next http.Handler, appSvc *application.Service) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			if err := appSvc.Save(ctx); err != nil {
				log.Printf("persist relational mutation %s %s: %v", r.Method, r.URL.Path, err)
			}
		}
	})
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
