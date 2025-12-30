package pinger

import (
	"context"
	"log"
	"net/http"
	"time"

	"keepalive-pinger/internal/config"
)

type Pinger struct {
	cfg    *config.Config
	client *http.Client
}

func New(cfg *config.Config) *Pinger {
	return &Pinger{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeout) * time.Millisecond,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
	}
}

func (p *Pinger) Start(ctx context.Context) {
	log.Println("pinger started")

	ticker := time.NewTicker(time.Duration(p.cfg.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			p.pingAll(ctx)
		case <-ctx.Done():
			log.Println("pinger shutdown complete")
			return
		}
	}
}

func (p *Pinger) pingAll(ctx context.Context) {
	for _, url := range p.cfg.TargetURLs {
		p.pingWithRetry(ctx, url)
	}
}

func (p *Pinger) pingWithRetry(ctx context.Context, url string) {
	var lastErr error

	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		start := time.Now()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Printf("url=%s error=%v\n", url, err)
			return
		}

		req.Header.Set("User-Agent", p.cfg.UserAgent)

		resp, err := p.client.Do(req)
		if err == nil {
			resp.Body.Close()
			duration := time.Since(start).Milliseconds()

			log.Printf(
				"url=%s status=%d duration_ms=%d attempt=%d\n",
				url, resp.StatusCode, duration, attempt,
			)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}
			lastErr = err
		} else {
			lastErr = err
		}

		time.Sleep(time.Duration(200*(attempt+1)) * time.Millisecond)
	}

	log.Printf("url=%s failed after retries error=%v\n", url, lastErr)
}