// Package rules holds the individual fraud scoring rules. Each rule is a pure
// function matching scoring.RuleFn; the engine in the scoring package composes
// them with weights from Config. Helpers and shared constants live here.
package rules

import (
	"fmt"
	"math"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// Amount thresholds in minor units of whatever currency the transaction uses.
// We treat magnitude alone as a signal regardless of FX — a $20K USD charge
// and a ₱20,000 PHP charge both clear amt20K. Names are currency-agnostic so
// future rules don't accidentally assume PHP.
const (
	amt3K  int64 = 3_000_00
	amt5K  int64 = 5_000_00
	amt10K int64 = 10_000_00
	amt20K int64 = 20_000_00
)

// countAny returns the number of distinct prior transactions sharing card,
// email, or device with txn within the given window. The current txn itself
// is excluded.
func countAny(h scoring.HistoryReader, txn scoring.Transaction, since time.Time) int {
	seen := map[string]struct{}{}
	collect := func(list []scoring.Transaction) {
		for _, t := range list {
			if t.ID == txn.ID {
				continue
			}
			seen[t.ID] = struct{}{}
		}
	}
	collect(h.GetByCard(txn.CardID, since))
	collect(h.GetByEmail(txn.CustomerEmail, since))
	collect(h.GetByDevice(txn.DeviceFingerprint, since))
	return len(seen)
}

func uniqueNonEmpty(values ...string) []string {
	seen := map[string]struct{}{}
	out := []string{}
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371.0
	toRad := func(d float64) float64 { return d * math.Pi / 180 }
	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func formatMoney(minor int64, currency string) string {
	if currency == "" {
		currency = "PHP"
	}
	return fmt.Sprintf("%.2f %s", float64(minor)/100, currency)
}
