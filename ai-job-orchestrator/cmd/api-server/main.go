package main

import (
	"log"
	"net/http"
	"os"

	"ai-job-orchestrator/internal/api"
	"ai-job-orchestrator/internal/store"
)

func main() {
	_ = os.MkdirAll("data", 0o755)
	dsn := os.Getenv("JOB_DB_DSN")
	db, err := store.Open(dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	addr := ":8080"
	if v := os.Getenv("API_LISTEN_ADDR"); v != "" {
		addr = v
	}
	log.Printf("api-server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, api.NewRouter(db)))
}
