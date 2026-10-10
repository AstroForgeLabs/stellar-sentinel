// Stellar Sentinel — lightweight Stellar payment monitor and notification relay.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/AstroForgeLabs/stellar-sentinel/internal/config"
	"github.com/AstroForgeLabs/stellar-sentinel/internal/dispatch"
	"github.com/AstroForgeLabs/stellar-sentinel/internal/filter"
	"github.com/AstroForgeLabs/stellar-sentinel/internal/horizon"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	matcher := filter.New(cfg.FilterAssets, cfg.MinAmount)

	dispatchCfg := dispatch.DispatchConfig{
		WebhookURL:        cfg.WebhookURL,
		WebhookSecret:     cfg.WebhookSecret,
		DiscordWebhookURL: cfg.DiscordWebhookURL,
		TelegramBotToken:  cfg.TelegramBotToken,
		TelegramChatID:    cfg.TelegramChatID,
	}

	slog.Info("stellar sentinel starting",
		"horizon", cfg.HorizonURL,
		"tracking", cfg.TrackedAddresses,
		"filter_assets", cfg.FilterAssets,
		"min_amount", cfg.MinAmount,
	)

	// Start a stream goroutine per tracked address.
	var wg sync.WaitGroup
	for _, addr := range cfg.TrackedAddresses {
		wg.Add(1)
		go func(accountID string) {
			defer wg.Done()
			runStream(ctx, cfg, matcher, dispatchCfg, accountID)
		}(addr)
	}

	// Health check HTTP server.
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ok"}`)
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"ready"}`)
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	go func() {
		slog.Info("health server listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("health server error", "error", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)

	wg.Wait()
	slog.Info("sentinel stopped")
}

func runStream(ctx context.Context, cfg *config.Config, matcher *filter.Matcher, dispatchCfg dispatch.DispatchConfig, accountID string) {
	payments, err := horizon.StreamPayments(ctx, cfg.HorizonURL, accountID, cfg.MaxBackoffSecs)
	if err != nil {
		slog.Error("failed to start stream", "account", accountID, "error", err)
		return
	}

	slog.Info("streaming payments", "account", accountID)

	for p := range payments {
		if !matcher.Match(p) {
			slog.Debug("payment filtered out",
				"event_id", p.ID, "asset", p.EffectiveAssetCode(), "amount", p.Amount)
			continue
		}

		slog.Info("payment matched — dispatching",
			"event_id", p.ID,
			"from", p.From,
			"to", p.To,
			"amount", p.Amount,
			"asset", p.EffectiveAssetCode(),
		)

		dispatch.Dispatch(ctx, dispatchCfg, accountID, p)
	}

	slog.Info("payment stream closed", "account", accountID)
}
