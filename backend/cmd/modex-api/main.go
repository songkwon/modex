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

func main() {
	addr := ":" + env("PORT", "8671")

	st, snapshotPath := loadStore()
	srv := api.New(st)

	httpServer := &http.Server{Addr: addr, Handler: srv.Handler()}

	// Periodic + graceful-shutdown persistence when DATA_DIR is configured.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	if snapshotPath != "" {
		go autosave(st, snapshotPath, stop)
	}

	go func() {
		log.Printf("modex-api listening on %s", addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-stop
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

func autosave(st *store.Store, path string, stop <-chan os.Signal) {
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
