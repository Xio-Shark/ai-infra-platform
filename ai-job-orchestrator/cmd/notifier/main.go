package main

import (
	"log"
	"os"
	"strconv"
	"time"
)

func main() {
	interval := 30 * time.Second
	if v := os.Getenv("NOTIFIER_INTERVAL_SEC"); v != "" {
		if sec, err := strconv.Atoi(v); err == nil && sec > 0 {
			interval = time.Duration(sec) * time.Second
		}
	}
	log.Printf("notifier stub: heartbeat every %s (wire webhooks / email later)", interval)
	for range time.Tick(interval) {
		log.Printf("notifier tick pid=%d", os.Getpid())
	}
}
