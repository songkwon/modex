package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"modex/backend/internal/api"
	"modex/backend/internal/store"
)

func analyticsSource() string {
	if api.PosthogConfigured() {
		return "PostHog (project_id=" + os.Getenv("POSTHOG_PROJECT_ID") + ", host=" + api.PosthogHost() + ")"
	}
	return "internal store (PostHog not configured)"
}

func main() {
	addr := ":" + env("PORT", "8671")

	st, snapshotPath := loadStore()
	srv := api.New(st)

	log.Printf("analytics source: %s", analyticsSource())

	httpServer := &http.Server{Addr: addr, Handler: srv.Handler()}

	// Periodic + graceful-shutdown persistence when DATA_DIR is configured.
	// The autosave goroutine gets its own stop channel (closed by main after the
	// OS signal). Sharing the signal channel would race: a signal delivered to a
	// channel wakes only ONE receiver, so the autosave goroutine could consume it
	// and leave main's `<-stop` blocked forever, skipping the final save.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	autosaveStop := make(chan struct{})
	if snapshotPath != "" {
		go autosave(st, snapshotPath, autosaveStop)
	}

	go func() {
		log.Printf("modex-api listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-stop
	close(autosaveStop)
	if snapshotPath != "" {
		if err := st.Save(snapshotPath); err != nil {
			log.Printf("final snapshot save failed: %v", err)
		} else {
			log.Printf("saved store snapshot to %s", snapshotPath)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}

// loadStore returns the store and the snapshot path (empty when persistence is
// disabled). When DATA_DIR is set it loads an existing snapshot or falls back to
// seeded data.
func loadStore() (*store.Store, string) {
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		log.Printf("DATA_DIR not set; starting with empty store (no demo data)")
		return store.New(), ""
	}
	path := filepath.Join(dataDir, "modex-store.json")
	if st, err := store.Load(path); err == nil {
		log.Printf("loaded store snapshot from %s", path)
		return st, path
	} else {
		log.Printf("no usable snapshot at %s (%v); starting from clean empty store", path, err)
		return store.New(), path
	}
}

func autosave(st *store.Store, path string, stop <-chan struct{}) {
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
			if err := st.Save(path); err != nil {
				log.Printf("snapshot save failed: %v", err)
			}
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
