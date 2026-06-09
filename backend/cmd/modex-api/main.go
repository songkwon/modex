package main

import (
	"log"
	"net/http"
	"os"

	"modex/backend/internal/api"
	"modex/backend/internal/store"
)

func main() {
	addr := ":" + env("PORT", "8080")
	srv := api.New(store.NewSeeded())
	log.Printf("modex-api listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
