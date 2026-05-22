package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// Velocity detects bursts of transactions sharing card, email, or device. The
// 30-minute window catches the specific pattern SariMart's fraud team called
// out: "3+ transactions within 30 minutes from same email or device".
//
// Shorter windows (2m, 5m, 10m) catch sharper bursts that are even more
// indicative of automated fraud rings.
func Velocity(txn scoring.Transaction, h scoring.HistoryReader, now time.Time) (float64, string) {
	if txn.CardID == "" && txn.CustomerEmail == "" && txn.DeviceFingerprint == "" {
		return 0, ""
	}
	c2m := countAny(h, txn, now.Add(-2*time.Minute))
	c5m := countAny(h, txn, now.Add(-5*time.Minute))
	c10m := countAny(h, txn, now.Add(-10*time.Minute))
	c30m := countAny(h, txn, now.Add(-30*time.Minute))

	score := 0.0
	if c2m > 2 {
		score = max(score, 60)
	}
	if c5m > 3 {
		score = max(score, 80)
	}
	if c10m > 5 {
		score = max(score, 100)
	}
	if c30m >= 3 {
		score = max(score, 70)
	}
	if score == 0 {
		return 0, ""
	}
	return score, fmt.Sprintf("%d related transactions in last 30m (%d in 5m, %d in 2m)", c30m, c5m, c2m)
}
