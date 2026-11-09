package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Address, DatabaseURL, Token, OllamaURL, Model, ResearchHosts string
	AIEnabled                                                    bool
	QueueCapacity                                                int
	HTTPTimeout                                                  time.Duration
}

func Load() (Config, error) {
	c := Config{Address: env("HTTP_ADDR", "127.0.0.1:8080"), DatabaseURL: os.Getenv("DATABASE_URL"), Token: os.Getenv("APP_TOKEN"), OllamaURL: env("OLLAMA_BASE_URL", "http://localhost:11434"), Model: env("OLLAMA_MODEL", "qwen3:1.7b"), ResearchHosts: os.Getenv("RESEARCH_ALLOWED_HOSTS"), HTTPTimeout: 25 * time.Second, QueueCapacity: 20}
	if c.DatabaseURL == "" {
		return c, fmt.Errorf("DATABASE_URL is required")
	}
	if len(c.Token) < 24 {
		return c, fmt.Errorf("APP_TOKEN must contain at least 24 characters")
	}
	var e error
	c.AIEnabled, e = strconv.ParseBool(env("AI_ENABLED", "true"))
	if e != nil {
		return c, fmt.Errorf("invalid AI_ENABLED")
	}
	if env("AI_MAX_CONCURRENT_REQUESTS", "1") != "1" {
		return c, fmt.Errorf("AI_MAX_CONCURRENT_REQUESTS must be 1 on this local-first MVP")
	}
	u, e := url.Parse(c.OllamaURL)
	if e != nil || u.Scheme != "http" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || !(u.Hostname() == "localhost" || net.ParseIP(u.Hostname()) != nil && net.ParseIP(u.Hostname()).IsLoopback()) {
		return c, fmt.Errorf("OLLAMA_BASE_URL must be a local HTTP origin")
	}
	return c, nil
}
func env(k, d string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return d
}
