package filter_test

import (
	"testing"

	"github.com/AstroForgeLabs/stellar-sentinel/internal/filter"
	"github.com/AstroForgeLabs/stellar-sentinel/internal/horizon"
)

func makePayment(assetType, assetCode, amount string) horizon.Payment {
	return horizon.Payment{
		ID:        "100-1-0",
		Type:      "payment",
		From:      "GABC1234567890123456789012345678901234567890123456789012",
		To:        "GBCD1234567890123456789012345678901234567890123456789012",
		Amount:    amount,
		AssetType: assetType,
		AssetCode: assetCode,
	}
}

func TestMatcher_NoFilter_MatchesAll(t *testing.T) {
	m := filter.New(nil, 0)
	p := makePayment("native", "", "100")
	if !m.Match(p) {
		t.Error("no-filter matcher should match all payments")
	}
}

func TestMatcher_AssetFilter_MatchesXLM(t *testing.T) {
	m := filter.New([]string{"XLM"}, 0)
	p := makePayment("native", "", "50")
	if !m.Match(p) {
		t.Error("XLM filter should match native payments")
	}
}

func TestMatcher_AssetFilter_RejectsUSDC(t *testing.T) {
	m := filter.New([]string{"XLM"}, 0)
	p := makePayment("credit_alphanum4", "USDC", "50")
	if m.Match(p) {
		t.Error("XLM-only filter should reject USDC payments")
	}
}

func TestMatcher_AssetFilter_MatchesUSDC(t *testing.T) {
	m := filter.New([]string{"USDC", "EURC"}, 0)
	p := makePayment("credit_alphanum4", "USDC", "10")
	if !m.Match(p) {
		t.Error("USDC filter should match USDC payments")
	}
}

func TestMatcher_AssetFilter_CaseInsensitive(t *testing.T) {
	m := filter.New([]string{"usdc"}, 0) // lowercase in config
	p := makePayment("credit_alphanum4", "USDC", "10")
	if !m.Match(p) {
		t.Error("asset filter should be case-insensitive")
	}
}

func TestMatcher_MinAmount_PassesAbove(t *testing.T) {
	m := filter.New(nil, 10.0)
	p := makePayment("native", "", "100")
	if !m.Match(p) {
		t.Error("100 XLM should pass a 10 XLM minimum")
	}
}

func TestMatcher_MinAmount_RejectsBelow(t *testing.T) {
	m := filter.New(nil, 50.0)
	p := makePayment("native", "", "5")
	if m.Match(p) {
		t.Error("5 XLM should be rejected by a 50 XLM minimum")
	}
}

func TestMatcher_MinAmount_ExactBoundary(t *testing.T) {
	m := filter.New(nil, 10.0)
	p := makePayment("native", "", "10")
	if !m.Match(p) {
		t.Error("exact boundary amount should pass")
	}
}

func TestMatcher_CombinedFilter(t *testing.T) {
	m := filter.New([]string{"USDC"}, 100.0)
	// USDC but below min amount
	p1 := makePayment("credit_alphanum4", "USDC", "50")
	if m.Match(p1) {
		t.Error("USDC below min amount should be rejected")
	}
	// USDC above min amount
	p2 := makePayment("credit_alphanum4", "USDC", "200")
	if !m.Match(p2) {
		t.Error("USDC above min amount should pass")
	}
	// XLM above min amount but wrong asset
	p3 := makePayment("native", "", "500")
	if m.Match(p3) {
		t.Error("XLM should be rejected when only USDC is allowed")
	}
}
