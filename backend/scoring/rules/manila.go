package rules

import (
	"strings"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// Manila is a context-aware rule for SariMart's main market. Plain
// Manila-area transactions are not signal (most legitimate customers live
// here); the rule only fires when a Manila transaction also carries one of
// the patterns the fraud team flagged: cross-border billing/shipping/IP, a
// velocity cluster, or a device change vs the previous transaction on the
// same card.
func Manila(txn scoring.Transaction, h scoring.HistoryReader, _ time.Time) (float64, string) {
	if txn.Lat == 0 && txn.Lng == 0 {
		return 0, ""
	}
	if haversineKm(txn.Lat, txn.Lng, 14.5995, 120.9842) > 50 {
		return 0, ""
	}
	score := 0.0
	reasons := []string{}
	if len(uniqueNonEmpty(txn.BillingCountry, txn.ShippingCountry, txn.IPCountry)) >= 2 {
		score = max(score, 70)
		reasons = append(reasons, "Manila-area tx with cross-border billing/shipping/IP")
	}
	if c := countAny(h, txn, txn.Timestamp.Add(-30*time.Minute)); c >= 3 {
		score = max(score, 85)
		reasons = append(reasons, "Manila-area tx with velocity cluster")
	}
	prev, ok := h.LastByCard(txn.CardID, txn.Timestamp)
	if ok && prev.DeviceFingerprint != "" && prev.DeviceFingerprint != txn.DeviceFingerprint {
		score = max(score, 95)
		reasons = append(reasons, "Manila-area tx with device change vs previous")
	}
	if score == 0 {
		return 0, ""
	}
	return score, strings.Join(reasons, "; ")
}
