package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// NewAccount fires when an account younger than 24 hours places a high-value
// order — a pattern the SariMart team called out explicitly ("accounts
// created within the last 24 hours making purchases over ₱5,000"). Prior
// declines on the same transaction compound the signal (card testing
// followed by approval is another flagged pattern).
func NewAccount(txn scoring.Transaction, _ scoring.HistoryReader, now time.Time) (float64, string) {
	if txn.AccountCreatedAt.IsZero() {
		return 0, ""
	}
	age := now.Sub(txn.AccountCreatedAt)
	if age > 24*time.Hour {
		return 0, ""
	}
	if txn.Amount < amt5K {
		return 0, ""
	}
	score := 70.0
	if txn.Amount > amt10K {
		score = 95
	}
	if txn.PreviousDeclines >= 2 {
		score = 100
	}
	return score, fmt.Sprintf("account %s old, amount %s, %d prior declines",
		age.Round(time.Minute), formatMoney(txn.Amount, txn.Currency), txn.PreviousDeclines)
}
