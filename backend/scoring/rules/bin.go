package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// BIN matches the card BIN (first 6 digits) against a configured list of
// high-risk issuers. SariMart's fraud team identified that certain prepaid
// BINs carry 8x higher fraud rates than the baseline.
//
// Returned as a closure so the BIN list comes from runtime config.
func BIN(cfg *scoring.Config) scoring.RuleFn {
	risky := map[string]struct{}{}
	for _, b := range cfg.HighRiskBINs {
		risky[b] = struct{}{}
	}
	return func(txn scoring.Transaction, _ scoring.HistoryReader, _ time.Time) (float64, string) {
		if txn.CardBIN == "" {
			return 0, ""
		}
		if _, ok := risky[txn.CardBIN]; ok {
			return 80, fmt.Sprintf("card BIN %s is on the high-risk list (prepaid issuers)", txn.CardBIN)
		}
		return 0, ""
	}
}
