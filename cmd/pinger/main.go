package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"keepalive-pinger/internal/config"
	"keepalive-pinger/internal/pinger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	p := pinger.New(cfg)
	go p.Start(ctx)

	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	log.Println("health server listening on", cfg.HealthPort)
	log.Fatal(http.ListenAndServe(":"+cfg.HealthPort, nil))
}