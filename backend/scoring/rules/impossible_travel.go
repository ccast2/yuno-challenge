package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// ImpossibleTravel compares the current transaction location with the
// previous transaction on the same card and computes implied travel speed
// (km/h). If the implied speed exceeds physical plausibility (e.g. faster
// than a commercial flight) the card is probably being used by two parties
// at once.
//
// Requires lat/lng on both transactions; returns 0 when geolocation is
// missing.
func ImpossibleTravel(txn scoring.Transaction, h scoring.HistoryReader, _ time.Time) (float64, string) {
	if txn.CardID == "" || (txn.Lat == 0 && txn.Lng == 0) {
		return 0, ""
	}
	prev, ok := h.LastByCard(txn.CardID, txn.Timestamp)
	if !ok || (prev.Lat == 0 && prev.Lng == 0) {
		return 0, ""
	}
	hours := txn.Timestamp.Sub(prev.Timestamp).Hours()
	if hours <= 0 {
		return 0, ""
	}
	km := haversineKm(prev.Lat, prev.Lng, txn.Lat, txn.Lng)
	if km < 50 {
		return 0, ""
	}
	kmh := km / hours
	score := 0.0
	switch {
	case kmh > 900:
		score = 100
	case kmh > 500:
		score = 75
	case kmh > 200:
		score = 50
	}
	if score == 0 {
		return 0, ""
	}
	return score, fmt.Sprintf("Implied speed %.0f km/h between last two transactions", kmh)
}
