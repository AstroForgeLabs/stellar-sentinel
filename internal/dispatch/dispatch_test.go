package dispatch_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AstroForgeLabs/stellar-sentinel/internal/dispatch"
	"github.com/AstroForgeLabs/stellar-sentinel/internal/horizon"
)

func testPayment() horizon.Payment {
	return horizon.Payment{
		ID:              "100-1-0",
		Type:            "payment",
		From:            "GABC1234567890123456789012345678901234567890123456789012",
		To:              "GBCD1234567890123456789012345678901234567890123456789012",
		Amount:          "50.0000000",
		AssetType:       "native",
		TransactionHash: "abc123def456",
	}
}

func TestSendWebhook_DeliveredSuccessfully(t *testing.T) {
	var received []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := testPayment()
	err := dispatch.SendWebhook(context.Background(), srv.URL, "", "GBCD...", p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload dispatch.WebhookPayload
	if err := json.Unmarshal(received, &payload); err != nil {
		t.Fatalf("could not unmarshal payload: %v", err)
	}
	if payload.EventID != p.ID {
		t.Errorf("expected event_id %q, got %q", p.ID, payload.EventID)
	}
	if payload.AssetCode != "XLM" {
		t.Errorf("expected asset_code XLM, got %q", payload.AssetCode)
	}
	if payload.Amount != p.Amount {
		t.Errorf("expected amount %q, got %q", p.Amount, payload.Amount)
	}
}

func TestSendWebhook_IncludesHMACSignatureWhenSecretSet(t *testing.T) {
	var sigHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sigHeader = r.Header.Get("X-Sentinel-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := testPayment()
	err := dispatch.SendWebhook(context.Background(), srv.URL, "my-secret", "GBCD...", p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sigHeader == "" {
		t.Error("expected X-Sentinel-Signature header when secret is set")
	}
	if len(sigHeader) < 7 || sigHeader[:7] != "sha256=" {
		t.Errorf("signature should start with sha256=, got %q", sigHeader)
	}
}

func TestSendWebhook_NoSignatureWhenSecretEmpty(t *testing.T) {
	var sigHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sigHeader = r.Header.Get("X-Sentinel-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	p := testPayment()
	dispatch.SendWebhook(context.Background(), srv.URL, "", "GBCD...", p)
	if sigHeader != "" {
		t.Error("expected no signature header when secret is empty")
	}
}

func TestSendWebhook_ReturnsErrorOnNonSuccessStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := testPayment()
	err := dispatch.SendWebhook(context.Background(), srv.URL, "", "GBCD...", p)
	if err == nil {
		t.Fatal("expected error for HTTP 500 response")
	}
}

func TestSendDiscord_DeliveredSuccessfully(t *testing.T) {
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusNoContent) // Discord returns 204
	}))
	defer srv.Close()

	p := testPayment()
	err := dispatch.SendDiscord(context.Background(), srv.URL, "GBCD...", p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var msg map[string]string
	if err := json.Unmarshal(body, &msg); err != nil {
		t.Fatalf("discord payload not valid JSON: %v", err)
	}
	if msg["content"] == "" {
		t.Error("discord payload missing content field")
	}
}
