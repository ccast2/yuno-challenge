package rules

import (
	"fmt"
	"strings"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// Currency covers two related geographic-mismatch signals:
//  1. Billing, shipping, and IP countries diverging — billing in one country,
//     shipping in another, IP in a third is one of the patterns SariMart's
//     team observed.
//  2. The transaction currency not matching the merchant country (e.g. a USD
//     charge on a PH merchant). Uses the country→currency map from config.
//
// Returned as a closure because it needs the config-loaded country map.
func Currency(cfg *scoring.Config) scoring.RuleFn {
	return func(txn scoring.Transaction, _ scoring.HistoryReader, _ time.Time) (float64, string) {
		countries := uniqueNonEmpty(txn.BillingCountry, txn.ShippingCountry, txn.IPCountry)
		score := 0.0
		reasons := []string{}
		if len(countries) >= 3 {
			score = 90
			reasons = append(reasons, "billing/shipping/IP in three different countries")
		} else if len(countries) == 2 {
			score = 50
			reasons = append(reasons, "billing/shipping/IP countries diverge")
		}
		if txn.MerchantCountry != "" && txn.Currency != "" {
			if expected, ok := cfg.CountryCurrency[txn.MerchantCountry]; ok && expected != txn.Currency {
				score = max(score, 60)
				reasons = append(reasons, fmt.Sprintf("currency %s mismatches merchant country %s", txn.Currency, txn.MerchantCountry))
			}
		}
		if score == 0 {
			return 0, ""
		}
		return score, strings.Join(reasons, "; ")
	}
}
