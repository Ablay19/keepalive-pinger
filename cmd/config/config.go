package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Targets        []string
	Interval       time.Duration
	RequestTimeout time.Duration
	UserAgent      string
	Concurrency    int
	MaxRetries     int
	BackoffBase    time.Duration
	BackoffMax     time.Duration
	MaxBodyBytes   int64
	LogJSON        bool
	RateLimitRPS   float64
	HealthPort     string

	RetryStatuses map[int]bool
}

func Load() (*Config, error) {
	targets := strings.Split(os.Getenv("TARGET_URLS"), ",")
	if len(targets) == 0 || targets[0] == "" {
		return nil, errors.New("TARGET_URLS is required")
	}

	intervalSec := getInt("INTERVAL_SECONDS", 60)
	if intervalSec < 10 && os.Getenv("ALLOW_ONE_SECOND_INTERVAL") != "true" {
		return nil, errors.New("interval < 10s requires ALLOW_ONE_SECOND_INTERVAL=true")
	}

	cfg := &Config{
		Targets:        targets,
		Interval:       time.Duration(intervalSec) * time.Second,
		RequestTimeout: time.Duration(getInt("REQUEST_TIMEOUT_MS", 5000)) * time.Millisecond,
		UserAgent:      getString("USER_AGENT", "keepalive-pinger/1.0"),
		Concurrency:    getInt("CONCURRENCY", 1),
		MaxRetries:     getInt("MAX_RETRIES", 2),
		BackoffBase:    time.Duration(getInt("BACKOFF_BASE_MS", 200)) * time.Millisecond,
		BackoffMax:     time.Duration(getInt("BACKOFF_MAX_MS", 2000)) * time.Millisecond,
		MaxBodyBytes:   int64(getInt("MAX_BODY_BYTES", 1<<20)),
		LogJSON:        getBool("LOG_JSON", false),
		RateLimitRPS:   getFloat("RATE_LIMIT_RPS", 0),
		HealthPort:     getString("HEALTH_PORT", "8080"),
		RetryStatuses: parseRetryStatuses(getString("ALLOW_RETRY_ON_STATUS", "429,5xx")),
	}

	return cfg, nil
}

func parseRetryStatuses(v string) map[int]bool {
	m := make(map[int]bool)
	parts := strings.Split(v, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "5xx" {
			for i := 500; i <= 599; i++ {
				m[i] = true
			}
		} else {
			if code, err := strconv.Atoi(p); err == nil {
				m[code] = true
			}
		}
	}
	return m
}

func getString(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return def
}

func getFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true"
	}
	return def
}