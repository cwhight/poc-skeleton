// Package config holds the runtime config: the env-driven service config and
// the idea config (the per-POC artifact, mirrored from idea.config.ts).
package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

// Idea is the Go mirror of idea.config.ts — the per-POC artifact. Only the
// fields the API needs at runtime are represented here.
type Idea struct {
	Name          string
	Slug          string
	SystemPrompt  string
	MaxInputBytes int64
	// SignalKind is "email" or "thumbs" — what /signal accepts for this POC.
	SignalKind string
}

// Config is the resolved runtime config from environment variables.
type Config struct {
	Port            string
	LLMProvider     string
	LLMAPIKey       string
	LLMModel        string
	DatabaseURL     string
	MaxInputBytes   int64
	RateLimitPerIP  int
	RateLimitWindow time.Duration
	Slug            string

	Idea Idea
}

// Load reads config from the environment. idea is the compiled-in idea config
// (the skeleton ships scopelock as the default — see DefaultIdea).
func Load(idea Idea) (Config, error) {
	c := Config{
		Port:        getenv("PORT", "8080"),
		LLMProvider: os.Getenv("LLM_PROVIDER"),
		LLMAPIKey:   os.Getenv("LLM_API_KEY"),
		LLMModel:    os.Getenv("LLM_MODEL"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Slug:        getenv("POC_SLUG", idea.Slug),
		Idea:        idea,
	}

	c.MaxInputBytes = atoiDefault(os.Getenv("MAX_INPUT_BYTES"), idea.MaxInputBytes)
	if c.MaxInputBytes <= 0 {
		c.MaxInputBytes = idea.MaxInputBytes
	}
	// Keep the idea's cap authoritative if env is unset.
	idea.MaxInputBytes = c.MaxInputBytes
	c.Idea = idea

	c.RateLimitPerIP = int(atoiDefault(os.Getenv("RATE_LIMIT_PER_IP"), 20))

	win := os.Getenv("RATE_LIMIT_WINDOW")
	if win == "" {
		c.RateLimitWindow = time.Hour
	} else if d, err := time.ParseDuration(win); err == nil {
		c.RateLimitWindow = d
	} else {
		c.RateLimitWindow = time.Hour
	}

	if c.LLMProvider == "" {
		return c, errors.New("LLM_PROVIDER is required")
	}
	if c.DatabaseURL == "" {
		return c, errors.New("DATABASE_URL is required")
	}
	return c, nil
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func atoiDefault(s string, def int64) int64 {
	if s == "" {
		return def
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return def
	}
	return n
}
