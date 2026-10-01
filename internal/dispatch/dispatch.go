// Package dispatch handles routing filtered payments to notification channels.
package dispatch

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/AstroForgeLabs/stellar-sentinel/internal/horizon"
)

// WebhookPayload is the JSON body sent to the configured WEBHOOK_URL.
type WebhookPayload struct {
	EventID         string    `json:"event_id"`
	TrackedAddress  string    `json:"tracked_address"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Amount          string    `json:"amount"`
	AssetCode       string    `json:"asset_code"`
	AssetIssuer     string    `json:"asset_issuer,omitempty"`
	TransactionHash string    `json:"transaction_hash"`
	DeliveredAt     time.Time `json:"delivered_at"`
}

// SendWebhook signs and POSTs a payment event to the configured webhook URL.
// The HMAC-SHA256 signature is included in the X-Sentinel-Signature header.
func SendWebhook(ctx context.Context, webhookURL, secret, trackedAddr string, p horizon.Payment) error {
	payload := WebhookPayload{
		EventID:         p.ID,
		TrackedAddress:  trackedAddr,
		From:            p.From,
		To:              p.To,
		Amount:          p.Amount,
		AssetCode:       p.EffectiveAssetCode(),
		AssetIssuer:     p.AssetIssuer,
		TransactionHash: p.TransactionHash,
		DeliveredAt:     time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Sentinel-Event-ID", p.ID)

	if secret != "" {
		sig := computeHMAC(body, secret)
		req.Header.Set("X-Sentinel-Signature", "sha256="+sig)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("POST webhook: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// SendDiscord dispatches a Discord notification via an incoming webhook URL.
func SendDiscord(ctx context.Context, webhookURL, trackedAddr string, p horizon.Payment) error {
	message := fmt.Sprintf(
		"💫 **Stellar Payment Detected**\n"+
			"• Tracked: `%s`\n"+
			"• From: `%s`\n"+
			"• Amount: **%s %s**\n"+
			"• Tx: `%s`",
		trackedAddr, p.From, p.Amount, p.EffectiveAssetCode(), p.TransactionHash,
	)

	body, err := json.Marshal(map[string]string{"content": message})
	if err != nil {
		return fmt.Errorf("marshal discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("discord POST: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// SendTelegram dispatches a Telegram message using the Bot API.
func SendTelegram(ctx context.Context, botToken, chatID, trackedAddr string, p horizon.Payment) error {
	text := fmt.Sprintf(
		"💫 Stellar Payment Detected\n"+
			"Tracked: %s\n"+
			"From: %s\n"+
			"Amount: %s %s\n"+
			"Tx: %s",
		trackedAddr, p.From, p.Amount, p.EffectiveAssetCode(), p.TransactionHash,
	)

	apiURL := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	body, err := json.Marshal(map[string]string{
		"chat_id": chatID,
		"text":    text,
	})
	if err != nil {
		return fmt.Errorf("marshal telegram payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build telegram request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("telegram POST: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 300 {
		return fmt.Errorf("telegram returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// Dispatch fans out a payment event to all configured notification channels.
func Dispatch(ctx context.Context, cfg DispatchConfig, trackedAddr string, p horizon.Payment) {
	if cfg.WebhookURL != "" {
		if err := SendWebhook(ctx, cfg.WebhookURL, cfg.WebhookSecret, trackedAddr, p); err != nil {
			slog.Error("webhook delivery failed", "error", err, "event_id", p.ID)
		} else {
			slog.Info("webhook delivered", "event_id", p.ID)
		}
	}

	if cfg.DiscordWebhookURL != "" {
		if err := SendDiscord(ctx, cfg.DiscordWebhookURL, trackedAddr, p); err != nil {
			slog.Error("discord delivery failed", "error", err, "event_id", p.ID)
		} else {
			slog.Info("discord delivered", "event_id", p.ID)
		}
	}

	if cfg.TelegramBotToken != "" && cfg.TelegramChatID != "" {
		if err := SendTelegram(ctx, cfg.TelegramBotToken, cfg.TelegramChatID, trackedAddr, p); err != nil {
			slog.Error("telegram delivery failed", "error", err, "event_id", p.ID)
		} else {
			slog.Info("telegram delivered", "event_id", p.ID)
		}
	}
}

// DispatchConfig holds the notification channel configuration.
type DispatchConfig struct {
	WebhookURL    string
	WebhookSecret string

	DiscordWebhookURL string

	TelegramBotToken string
	TelegramChatID   string
}

// computeHMAC returns the hex-encoded HMAC-SHA256 of body signed with secret.
func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
