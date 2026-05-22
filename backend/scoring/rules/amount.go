package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// Amount scores based on raw transaction amount. Two failure modes are
// covered: oversized purchases (typical chargeback fraud) and micro purchases
// (card testing, the precursor pattern SariMart's team observed).
//
// Round-number amounts (e.g. exactly ₱10,000.00) get an additive bonus — the
// fraud team flagged these as a recurring tell.
func Amount(txn scoring.Transaction, _ scoring.HistoryReader, _ time.Time) (float64, string) {
	amt := txn.Amount
	if amt <= 0 {
		return 0, ""
	}
	score := 0.0
	switch {
	case amt > amt20K:
		score = 100
	case amt > amt10K:
		score = 75
	case amt > amt5K:
		score = 60
	case amt > amt3K:
		score = 40
	case amt < 10_00:
		score = 30
	}
	if amt >= 1_000_00 && amt%1_000_00 == 0 {
		score += 15
	}
	if score > 100 {
		score = 100
	}
	if score == 0 {
		return 0, ""
	}
	return score, fmt.Sprintf("Amount %s is anomalous", formatMoney(amt, txn.Currency))
}
