package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// Device counts how many distinct card IDs have been seen on the same device
// fingerprint in the last hour. A single legitimate customer rarely uses more
// than 1-2 cards from the same device; many cards on one device is a classic
// fraud-ring tell.
func Device(txn scoring.Transaction, h scoring.HistoryReader, now time.Time) (float64, string) {
	if txn.DeviceFingerprint == "" {
		return 0, ""
	}
	since := now.Add(-1 * time.Hour)
	seen := map[string]struct{}{}
	for _, t := range h.GetByDevice(txn.DeviceFingerprint, since) {
		if t.CardID != "" {
			seen[t.CardID] = struct{}{}
		}
	}
	if txn.CardID != "" {
		seen[txn.CardID] = struct{}{}
	}
	n := len(seen)
	score := 0.0
	switch {
	case n > 3:
		score = 100
	case n > 2:
		score = 70
	case n > 1:
		score = 40
	}
	if score == 0 {
		return 0, ""
	}
	return score, fmt.Sprintf("%d distinct cards seen on same device in last 1h", n)
}
