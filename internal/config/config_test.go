package config_test

import (
	"os"
	"testing"

	"github.com/AstroForgeLabs/stellar-sentinel/internal/config"
)

func setEnv(t *testing.T, key, val string) {
	t.Helper()
	t.Setenv(key, val)
}

func baseEnv(t *testing.T) {
	t.Helper()
	setEnv(t, "TRACKED_ADDRESSES", "GABC1234567890123456789012345678901234567890123456789012")
	setEnv(t, "WEBHOOK_URL", "https://example.com/hook")
}

func TestLoad_ValidMinimalConfig(t *testing.T) {
	baseEnv(t)
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HorizonURL != "https://horizon-testnet.stellar.org" {
		t.Errorf("expected default testnet URL, got %q", cfg.HorizonURL)
	}
	if len(cfg.TrackedAddresses) != 1 {
		t.Errorf("expected 1 tracked address, got %d", len(cfg.TrackedAddresses))
	}
	if cfg.Port != 8080 {
		t.Errorf("expected default port 8080, got %d", cfg.Port)
	}
}

func TestLoad_CustomHorizonURL(t *testing.T) {
	baseEnv(t)
	setEnv(t, "HORIZON_URL", "https://horizon.stellar.org")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.HorizonURL != "https://horizon.stellar.org" {
		t.Errorf("expected mainnet URL, got %q", cfg.HorizonURL)
	}
}

func TestLoad_MissingTrackedAddresses(t *testing.T) {
	os.Unsetenv("TRACKED_ADDRESSES")
	setEnv(t, "WEBHOOK_URL", "https://example.com/hook")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for missing TRACKED_ADDRESSES")
	}
}

func TestLoad_InvalidPublicKey(t *testing.T) {
	setEnv(t, "TRACKED_ADDRESSES", "notakey")
	setEnv(t, "WEBHOOK_URL", "https://example.com/hook")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid public key")
	}
}

func TestLoad_NoNotificationChannel(t *testing.T) {
	setEnv(t, "TRACKED_ADDRESSES", "GABC1234567890123456789012345678901234567890123456789012")
	os.Unsetenv("WEBHOOK_URL")
	os.Unsetenv("DISCORD_WEBHOOK_URL")
	os.Unsetenv("TELEGRAM_BOT_TOKEN")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error when no notification channel set")
	}
}

func TestLoad_MultipleTrackedAddresses(t *testing.T) {
	setEnv(t, "TRACKED_ADDRESSES",
		"GABC1234567890123456789012345678901234567890123456789012,"+
			"GBCD1234567890123456789012345678901234567890123456789012")
	setEnv(t, "WEBHOOK_URL", "https://example.com/hook")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.TrackedAddresses) != 2 {
		t.Errorf("expected 2 tracked addresses, got %d", len(cfg.TrackedAddresses))
	}
}

func TestLoad_FilterAssets(t *testing.T) {
	baseEnv(t)
	setEnv(t, "FILTER_ASSETS", "usdc,xlm")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.FilterAssets) != 2 {
		t.Errorf("expected 2 filter assets, got %d", len(cfg.FilterAssets))
	}
}

func TestLoad_InvalidPort(t *testing.T) {
	baseEnv(t)
	setEnv(t, "PORT", "99999")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error for invalid port")
	}
}

func TestLoad_MinAmount(t *testing.T) {
	baseEnv(t)
	setEnv(t, "MIN_AMOUNT", "10.5")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.MinAmount != 10.5 {
		t.Errorf("expected min amount 10.5, got %f", cfg.MinAmount)
	}
}
