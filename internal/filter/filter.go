// Package filter provides payment filtering logic for asset code and amount.
package filter

import (
	"strconv"
	"strings"

	"github.com/AstroForgeLabs/stellar-sentinel/internal/horizon"
)

// Matcher holds the configured filter criteria.
type Matcher struct {
	// allowedAssets is a set of uppercase asset codes. Empty = accept all.
	allowedAssets map[string]struct{}
	// minAmount is the minimum payment amount. 0 = no minimum.
	minAmount float64
}

// New creates a new Matcher from the given asset codes and minimum amount.
func New(assetCodes []string, minAmount float64) *Matcher {
	allowed := make(map[string]struct{}, len(assetCodes))
	for _, code := range assetCodes {
		allowed[strings.ToUpper(code)] = struct{}{}
	}
	return &Matcher{allowedAssets: allowed, minAmount: minAmount}
}

// Match returns true if the payment passes all configured filters.
func (m *Matcher) Match(p horizon.Payment) bool {
	if !m.matchAsset(p) {
		return false
	}
	if !m.matchAmount(p) {
		return false
	}
	return true
}

func (m *Matcher) matchAsset(p horizon.Payment) bool {
	if len(m.allowedAssets) == 0 {
		return true // no filter — accept all
	}
	code := strings.ToUpper(p.EffectiveAssetCode())
	_, ok := m.allowedAssets[code]
	return ok
}

func (m *Matcher) matchAmount(p horizon.Payment) bool {
	if m.minAmount <= 0 {
		return true // no minimum
	}
	amount, err := strconv.ParseFloat(p.Amount, 64)
	if err != nil {
		return false // unparseable amount — skip
	}
	return amount >= m.minAmount
}
