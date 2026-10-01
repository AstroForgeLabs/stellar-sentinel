// Package horizon provides a streaming Horizon SSE payment subscriber with
// automatic reconnect and exponential backoff.
package horizon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"strings"
	"time"
)

// Payment represents a decoded Stellar payment from Horizon's SSE stream.
type Payment struct {
	ID              string    `json:"id"`
	Type            string    `json:"type"`
	SourceAccount   string    `json:"source_account"`
	To              string    `json:"to"`
	From            string    `json:"from"`
	Amount          string    `json:"amount"`
	AssetCode       string    `json:"asset_code"`
	AssetIssuer     string    `json:"asset_issuer"`
	AssetType       string    `json:"asset_type"`
	CreatedAt       time.Time `json:"created_at"`
	TransactionHash string    `json:"transaction_hash"`
}

// IsNative returns true when the payment is in native XLM.
func (p Payment) IsNative() bool {
	return p.AssetType == "native"
}

// EffectiveAssetCode returns "XLM" for native payments, asset_code otherwise.
func (p Payment) EffectiveAssetCode() string {
	if p.IsNative() {
		return "XLM"
	}
	return p.AssetCode
}

// StreamPayments subscribes to Horizon's SSE payment stream for accountID and
// sends each decoded Payment on the returned channel. It reconnects with
// exponential backoff when the stream drops. The caller must cancel ctx to stop.
func StreamPayments(ctx context.Context, horizonURL, accountID string, maxBackoffSecs int) (<-chan Payment, error) {
	out := make(chan Payment, 64)

	go func() {
		defer close(out)
		var cursor string
		attempt := 0

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			err := streamOnce(ctx, horizonURL, accountID, cursor, out, &cursor)
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				backoff := backoffDuration(attempt, maxBackoffSecs)
				slog.Warn("horizon stream disconnected, reconnecting",
					"account", accountID, "attempt", attempt, "backoff", backoff, "error", err)
				attempt++
				select {
				case <-ctx.Done():
					return
				case <-time.After(backoff):
				}
			} else {
				attempt = 0
			}
		}
	}()

	return out, nil
}

// streamOnce opens a single SSE connection and reads events until EOF or error.
// It updates *lastCursor to allow resumption on reconnect.
func streamOnce(ctx context.Context, horizonURL, accountID, cursor string, out chan<- Payment, lastCursor *string) error {
	url := fmt.Sprintf("%s/accounts/%s/payments?order=asc&limit=200", horizonURL, accountID)
	if cursor != "" {
		url += "&cursor=" + cursor
	} else {
		url += "&cursor=now"
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d from Horizon", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var dataLine string

	for scanner.Scan() {
		line := scanner.Text()

		if strings.HasPrefix(line, "data: ") {
			dataLine = strings.TrimPrefix(line, "data: ")
			continue
		}

		if line == "" && dataLine != "" {
			// End of SSE event — parse it
			if dataLine == `"hello"` || dataLine == `"bye"` {
				dataLine = ""
				continue
			}

			var p Payment
			if err := json.Unmarshal([]byte(dataLine), &p); err == nil {
				if p.ID != "" {
					*lastCursor = p.ID
					select {
					case out <- p:
					case <-ctx.Done():
						return nil
					}
				}
			}
			dataLine = ""
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("stream read error: %w", err)
	}
	return nil
}

// backoffDuration returns an exponentially increasing duration capped at maxSecs.
func backoffDuration(attempt, maxSecs int) time.Duration {
	secs := math.Min(float64(maxSecs), math.Pow(2, float64(attempt)))
	return time.Duration(secs) * time.Second
}
