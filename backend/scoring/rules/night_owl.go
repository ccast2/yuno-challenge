package rules

import (
	"fmt"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// countryTZ maps ISO country codes to IANA time zones so NightOwl can evaluate
// the *local* hour at the merchant or cardholder location, not UTC. This is
// the table SariMart's data sits in — extend it when new markets land. Lookups
// fall through to UTC if the country is missing.
var countryTZ map[string]*time.Location

func init() {
	pairs := map[string]string{
		"PH": "Asia/Manila",
		"ID": "Asia/Jakarta",
		"VN": "Asia/Ho_Chi_Minh",
		"US": "America/New_York",
		"CO": "America/Bogota",
		"MX": "America/Mexico_City",
		"GB": "Europe/London",
		"JP": "Asia/Tokyo",
		"BR": "America/Sao_Paulo",
		"AR": "America/Argentina/Buenos_Aires",
	}
	countryTZ = make(map[string]*time.Location, len(pairs))
	for code, name := range pairs {
		if loc, err := time.LoadLocation(name); err == nil {
			countryTZ[code] = loc
		}
	}
}

// localHour returns the hour-of-day for the transaction's timestamp at the
// best location we can resolve. Priority: merchant country, then billing
// country, then UTC fallback. The fallback case is logged via the reason
// string so reviewers can tell when we lost precision.
func localHour(txn scoring.Transaction) (int, string) {
	for _, code := range []string{txn.MerchantCountry, txn.BillingCountry} {
		if code == "" {
			continue
		}
		if loc, ok := countryTZ[code]; ok {
			return txn.Timestamp.In(loc).Hour(), loc.String()
		}
	}
	return txn.Timestamp.UTC().Hour(), "UTC"
}

// NightOwl scores transactions by hour-of-day in the cardholder's local zone.
// SariMart's fraud team measured that 67% of confirmed fraud landed between
// 1AM and 6AM local time. We treat the 01:00–05:00 local window as the
// primary signal and the adjacent hours (00, 06) as a softer indicator.
func NightOwl(txn scoring.Transaction, _ scoring.HistoryReader, _ time.Time) (float64, string) {
	h, zone := localHour(txn)
	switch {
	case h >= 1 && h <= 5:
		return 60, fmt.Sprintf("transaction at %02d:00 %s (high-fraud window)", h, zone)
	case h == 0 || h == 6:
		return 25, fmt.Sprintf("transaction at %02d:00 %s (adjacent window)", h, zone)
	}
	return 0, ""
}
