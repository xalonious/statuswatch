package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ServiceProvider string

const (
	ProviderStatusIO  ServiceProvider = "statusio"
	ProviderAtlassian ServiceProvider = "atlassian"
)

type Service struct {
	Name       string          `json:"name"`
	URL        string          `json:"url"`
	Provider   ServiceProvider `json:"provider"`
	Components []string        `json:"components,omitempty"`
}

type Config struct {
	DiscordWebhookURL   string    `json:"webhook_url"`
	PollIntervalSeconds int       `json:"poll_interval_seconds"`
	MinImpact           string    `json:"min_impact,omitempty"`
	Services            []Service `json:"services"`
}

func loadConfig() Config {
	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: could not determine executable path:", err)
		os.Exit(1)
	}
	configPath := filepath.Join(filepath.Dir(exe), "config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: could not read config.json:", err)
		fmt.Fprintln(os.Stderr, "  Make sure config.json exists next to the binary.")
		os.Exit(1)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: could not parse config.json:", err)
		os.Exit(1)
	}

	if cfg.DiscordWebhookURL == "" {
		fmt.Fprintln(os.Stderr, "ERROR: webhook_url is not set in config.json")
		os.Exit(1)
	}

	if len(cfg.Services) == 0 {
		fmt.Fprintln(os.Stderr, "ERROR: no services defined in config.json")
		os.Exit(1)
	}

	if cfg.PollIntervalSeconds <= 0 {
		cfg.PollIntervalSeconds = 120
	}

	return cfg
}