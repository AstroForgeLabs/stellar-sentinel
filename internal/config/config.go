// Package config loads and validates Stellar Sentinel configuration from
// environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds the fully validated runtime configuration.
type Config struct {
	HorizonURL      string
	TrackedAddresses []string

	// Notification channels — at least one must be set.
	WebhookURL    string
	WebhookSecret string

	DiscordWebhookURL string

	TelegramBotToken string
	TelegramChatID   string

	// Optional filters
	FilterAssets []string // empty = accept all assets
	MinAmount    float64  // 0 = no minimum

	// HTTP server
	Port int

	// Reconnect tuning
	MaxBackoffSecs int
}

// Load reads configuration from environment variables and returns a validated Config.
func Load() (*Config, error) {
	cfg := &Config{}

	cfg.HorizonURL = env("HORIZON_URL", "https://horizon-testnet.stellar.org")

	addrs := env("TRACKED_ADDRESSES", "")
	if addrs == "" {
		return nil, fmt.Errorf("TRACKED_ADDRESSES is required: set a comma-separated list of Stellar public keys")
	}
	cfg.TrackedAddresses = splitCSV(addrs)
	for _, addr := range cfg.TrackedAddresses {
		if !isValidPublicKey(addr) {
			return nil, fmt.Errorf("invalid Stellar public key in TRACKED_ADDRESSES: %q", addr)
		}
	}

	cfg.WebhookURL = env("WEBHOOK_URL", "")
	cfg.WebhookSecret = env("WEBHOOK_SECRET", "")
	cfg.DiscordWebhookURL = env("DISCORD_WEBHOOK_URL", "")
	cfg.TelegramBotToken = env("TELEGRAM_BOT_TOKEN", "")
	cfg.TelegramChatID = env("TELEGRAM_CHAT_ID", "")

	if cfg.WebhookURL == "" && cfg.DiscordWebhookURL == "" && cfg.TelegramBotToken == "" {
		return nil, fmt.Errorf("at least one notification channel must be configured: WEBHOOK_URL, DISCORD_WEBHOOK_URL, or TELEGRAM_BOT_TOKEN")
	}

	filterStr := env("FILTER_ASSETS", "")
	if filterStr != "" {
		cfg.FilterAssets = splitCSV(strings.ToUpper(filterStr))
	}

	minAmountStr := env("MIN_AMOUNT", "0")
	minAmount, err := strconv.ParseFloat(minAmountStr, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid MIN_AMOUNT %q: must be a number", minAmountStr)
	}
	cfg.MinAmount = minAmount

	portStr := env("PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid PORT %q: must be 1-65535", portStr)
	}
	cfg.Port = port

	backoffStr := env("MAX_BACKOFF_SECS", "60")
	backoff, err := strconv.Atoi(backoffStr)
	if err != nil || backoff < 1 {
		return nil, fmt.Errorf("invalid MAX_BACKOFF_SECS %q: must be a positive integer", backoffStr)
	}
	cfg.MaxBackoffSecs = backoff

	return cfg, nil
}

func env(key, defaultVal string) string {
	if v, ok := os.LookupEnv(key); ok {
		return strings.TrimSpace(v)
	}
	return defaultVal
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// isValidPublicKey does a quick structural check — 56 chars starting with G.
func isValidPublicKey(key string) bool {
	return len(key) == 56 && strings.HasPrefix(key, "G")
}
