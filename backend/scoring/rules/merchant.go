package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// Merchant counts distinct merchants charged on the same card in a short
// window. A legitimate cardholder rarely transacts with many different
// merchants in 10 minutes; fraudsters running stolen cards through
// automation commonly do.
func Merchant(txn scoring.Transaction, h scoring.HistoryReader, now time.Time) (float64, string) {
	if txn.CardID == "" {
		return 0, ""
	}
	seen := map[string]struct{}{txn.MerchantID: {}}
	for _, t := range h.GetByCard(txn.CardID, now.Add(-10*time.Minute)) {
		if t.MerchantID != "" {
			seen[t.MerchantID] = struct{}{}
		}
	}
	n := len(seen)
	score := 0.0
	switch {
	case n > 5:
		score = 100
	case n > 3:
		score = 70
	case n > 2:
		score = 40
	}
	if score == 0 {
		return 0, ""
	}
	return score, fmt.Sprintf("%d distinct merchants on same card in last 10m", n)
}
