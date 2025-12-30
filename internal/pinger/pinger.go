package pinger

import (
	"context"
	"io"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"keepalive-pinger/internal/config"
)

type Pinger struct {
	cfg     *config.Config
	client  *http.Client
	limiter *rate.Limiter

	mu        sync.Mutex
	inFlight  sync.WaitGroup
	shutdown  bool
}

func New(cfg *config.Config, client *http.Client) *Pinger {
	var limiter *rate.Limiter
	if cfg.RateLimitRPS > 0 {
		limiter = rate.NewLimiter(rate.Limit(cfg.RateLimitRPS), 1)
	}

	return &Pinger{
		cfg:     cfg,
		client: client,
		limiter: limiter,
	}
}

func NewHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
	}
}

func (p *Pinger) RunOnce(ctx context.Context) {
	sem := make(chan struct{}, p.cfg.Concurrency)

	for _, url := range p.cfg.Targets {
		sem <- struct{}{}
		p.inFlight.Add(1)

		go func(u string) {
			defer func() {
				<-sem
				p.inFlight.Done()
			}()

			p.pingWithRetry(ctx, u)
		}(url)
	}
}

func (p *Pinger) pingWithRetry(ctx context.Context, url string) {
	for attempt := 0; attempt <= p.cfg.MaxRetries; attempt++ {
		if p.limiter != nil {
			_ = p.limiter.Wait(ctx)
		}

		reqCtx, cancel := context.WithTimeout(ctx, p.cfg.RequestTimeout)
		req, _ := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
		req.Header.Set("User-Agent", p.cfg.UserAgent)

		start := time.Now()
		resp, err := p.client.Do(req)
		duration := time.Since(start)
		cancel()

		if err == nil {
			io.Copy(io.Discard, io.LimitReader(resp.Body, p.cfg.MaxBodyBytes))
			resp.Body.Close()

			log.Printf("url=%s status=%d duration_ms=%d attempt=%d",
				url, resp.StatusCode, duration.Milliseconds(), attempt)

			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				return
			}

			if !p.cfg.RetryStatuses[resp.StatusCode] {
				return
			}
		}

		if attempt < p.cfg.MaxRetries {
			backoff := p.cfg.BackoffBase * (1 << attempt)
			if backoff > p.cfg.BackoffMax {
				backoff = p.cfg.BackoffMax
			}
			jitter := time.Duration(rand.Int63n(int64(backoff)))
			time.Sleep(jitter)
		}
	}
}

func (p *Pinger) StartHealthServer(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:    ":" + p.cfg.HealthPort,
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		srv.Shutdown(context.Background())
	}()

	return srv.ListenAndServe()
}

func (p *Pinger) Shutdown() {
	p.mu.Lock()
	p.shutdown = true
	p.mu.Unlock()

	p.inFlight.Wait()
}