package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"keepalive-pinger/internal/config"
	"keepalive-pinger/internal/pinger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Fatalf("config error: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpClient := pinger.NewHTTPClient()

	p := pinger.New(cfg, httpClient)

	go func() {
		if err := p.StartHealthServer(ctx); err != nil && err != http.ErrServerClosed {
			log.Fatalf("health server error: %v", err)
		}
	}()

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	log.Println("pinger started")

	for {
		select {
		case <-ticker.C:
			p.RunOnce(ctx)
		case <-ctx.Done():
			log.Println("shutdown requested")
			p.Shutdown()
			return
		}
	}
}
