package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	TargetURLs      []string
	IntervalSeconds int
	RequestTimeout  int
	HealthPort      string
	UserAgent       string
	MaxRetries      int
}

func Load() (*Config, error) {
	targets := strings.Split(strings.TrimSpace(os.Getenv("TARGET_URLS")), ",")
	if len(targets) == 1 && targets[0] == "" {
		return nil, errors.New("TARGET_URLS is required")
	}

	interval := getInt("INTERVAL_SECONDS", 60)
	timeout := getInt("REQUEST_TIMEOUT_MS", 5000)
	retries := getInt("MAX_RETRIES", 2)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		TargetURLs:      targets,
		IntervalSeconds: interval,
		RequestTimeout:  timeout,
		HealthPort:      port,
		UserAgent:       getString("USER_AGENT", "keepalive-pinger/1.0"),
		MaxRetries:      retries,
	}, nil
}

func getInt(key string, def int) int {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func getString(key, def string) string {
	v := os.Getenv(key)
	if v == "" {
		return def
	}
	return v
}