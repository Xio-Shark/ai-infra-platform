package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type notifierConfig struct {
	Addr          string `json:"addr"`
	Mode          string `json:"mode"`
	UpstreamAPI   string `json:"upstream_api"`
	DeliveryStub  string `json:"delivery_stub"`
	TraceConsumer string `json:"trace_consumer"`
}

func main() {
	cfg := notifierConfig{
		Addr:          envOrDefault("NOTIFIER_ADDR", ":8081"),
		Mode:          envOrDefault("NOTIFIER_MODE", "noop"),
		UpstreamAPI:   envOrDefault("API_SERVER_URL", "http://127.0.0.1:8080"),
		DeliveryStub:  "log-only",
		TraceConsumer: "GET /jobs/{id}/trace",
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "mode": cfg.Mode})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ready", "upstream_api": cfg.UpstreamAPI})
	})
	mux.HandleFunc("/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, cfg)
	})
	mux.HandleFunc("/contract", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"consumes": []string{
				"job execution lifecycle events emitted through logs/traces",
				"API server trace timeline via " + cfg.TraceConsumer,
			},
			"current_behavior": []string{
				"does not send real email, webhook, or queue notification",
				"documents notifier boundary as a standalone HTTP process",
			},
		})
	})
	log.Printf("notifier listening on %s", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, mux); err != nil {
		log.Fatalf("notifier stopped: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
