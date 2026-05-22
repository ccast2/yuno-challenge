// Package datagen produces synthetic transaction datasets for demos, seeding,
// and tests. Output is deterministic per (seed, count) so reviewers and tests
// see the same data on every run.
package datagen

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

// HighRiskClusterSize is the number of transactions in the embedded fraud
// scenarios (velocity burst, card testing, storm, impossible travel, device
// share). Below this count, generation can't keep the 70/20/10 mix intact.
const HighRiskClusterSize = 17

// MinCount is the smallest count that still includes all high-risk scenarios.
const MinCount = HighRiskClusterSize

// Defaults used when callers don't customize.
var (
	defaultSeed = [2]uint64{42, 7}
	defaultBase = time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
)

// RiskBias controls how the generator distributes risk levels across the output.
type RiskBias string

const (
	BiasMixed  RiskBias = "mixed"  // ~70/20/10 split + 17-tx high-risk cluster (default)
	BiasMedium RiskBias = "medium" // medium-risk only (single-signal transactions)
	BiasHigh   RiskBias = "high"   // high-risk only (repeated fraud-scenario clusters)
)

// Options tunes generation. Zero values produce the canonical demo dataset.
type Options struct {
	Total int       // total transactions to produce
	Seed  [2]uint64 // PRNG seed (PCG); zero → default deterministic seed
	Base  time.Time // anchor timestamp; zero → 2026-05-21T12:00:00Z
	Bias  RiskBias  // empty → BiasMixed
}

// Generate returns synthetic transactions according to opts.Bias.
//
//   - BiasMixed (default): ~70% low / 20% medium / one full 17-tx high-risk cluster.
//   - BiasMedium: only medium-risk transactions (single suspicious signal each),
//     useful for exercising the REVIEW threshold.
//   - BiasHigh: only high-risk patterns — repeats the 17-tx fraud cluster
//     enough times to fill opts.Total, with suffixed identifiers per repeat so
//     velocity/device rules still cluster within each repeat.
func Generate(opts Options) []scoring.Transaction {
	total := opts.Total
	if total <= 0 {
		total = 183
	}
	seed := opts.Seed
	if seed == ([2]uint64{}) {
		seed = defaultSeed
	}
	base := opts.Base
	if base.IsZero() {
		base = defaultBase
	}
	bias := opts.Bias
	if bias == "" {
		bias = BiasMixed
	}
	r := rand.New(rand.NewPCG(seed[0], seed[1]))

	switch bias {
	case BiasHigh:
		repeats := max(total/HighRiskClusterSize, 1)
		out := make([]scoring.Transaction, 0, repeats*HighRiskClusterSize)
		for k := range repeats {
			out = append(out, highRiskClusterTagged(base, k)...)
		}
		return out

	case BiasMedium:
		out := make([]scoring.Transaction, 0, total)
		for i := range total {
			out = append(out, mediumRisk(r, base, i))
		}
		return out

	default: // BiasMixed
		if total < MinCount {
			total = MinCount
		}
		remainder := total - HighRiskClusterSize
		lowCount := int(float64(remainder) * 0.78)
		medCount := remainder - lowCount
		out := make([]scoring.Transaction, 0, total)
		for i := range lowCount {
			out = append(out, lowRisk(r, base, i))
		}
		for i := range medCount {
			out = append(out, mediumRisk(r, base, i))
		}
		out = append(out, highRiskCluster(base)...)
		return out
	}
}

// highRiskClusterTagged returns the canonical high-risk cluster with all
// identifiers suffixed by tag, so multiple clusters can coexist in the same
// run without ID collisions while keeping each cluster's velocity/device
// patterns intact.
func highRiskClusterTagged(base time.Time, tag int) []scoring.Transaction {
	suffix := fmt.Sprintf("-r%d", tag)
	cluster := highRiskCluster(base.Add(time.Duration(tag) * time.Hour))
	for i := range cluster {
		cluster[i].ID += suffix
		cluster[i].CustomerID += suffix
		cluster[i].CardID += suffix
		cluster[i].DeviceFingerprint += suffix
		// Keep email distinct per cluster too so velocity within doesn't bleed
		// across clusters.
		if cluster[i].CustomerEmail != "" {
			cluster[i].CustomerEmail = fmt.Sprintf("r%d-%s", tag, cluster[i].CustomerEmail)
		}
	}
	return cluster
}

func lowRisk(r *rand.Rand, base time.Time, i int) scoring.Transaction {
	countries := []string{"PH", "ID", "VN"}
	c := countries[i%len(countries)]
	hour := 9 + r.IntN(11)
	ts := base.Add(-time.Duration(i)*30*time.Minute + time.Duration(hour-12)*time.Hour)
	cur := map[string]string{"PH": "PHP", "ID": "IDR", "VN": "VND"}[c]
	return scoring.Transaction{
		ID:                fmt.Sprintf("low-%03d", i),
		CustomerID:        fmt.Sprintf("cust-low-%03d", i),
		CustomerEmail:     fmt.Sprintf("user%03d@gmail.com", i),
		AccountCreatedAt:  base.Add(-time.Duration(60+r.IntN(540)) * 24 * time.Hour),
		CardID:            fmt.Sprintf("card-low-%03d", i),
		CardBIN:           "453985",
		CardLast4:         fmt.Sprintf("%04d", 1000+i),
		DeviceFingerprint: fmt.Sprintf("dev-low-%03d", i),
		IP:                fmt.Sprintf("180.190.%d.%d", i%255, (i*7)%255),
		IPCountry:         c,
		BillingCountry:    c,
		ShippingCountry:   c,
		MerchantID:        fmt.Sprintf("merch-%03d", i%30),
		MerchantCountry:   c,
		Amount:            int64(200_00 + r.IntN(2_500_00)),
		Currency:          cur,
		Lat:               14.5995 + (r.Float64()-0.5)*0.2,
		Lng:               120.9842 + (r.Float64()-0.5)*0.2,
		Timestamp:         ts,
	}
}

func mediumRisk(r *rand.Rand, base time.Time, i int) scoring.Transaction {
	switch i % 3 {
	case 0:
		ts := time.Date(base.Year(), base.Month(), base.Day(), 2+r.IntN(4), r.IntN(60), 0, 0, time.UTC)
		return scoring.Transaction{
			ID: fmt.Sprintf("mid-night-%03d", i), CustomerID: fmt.Sprintf("cust-mid-%03d", i),
			CustomerEmail:     fmt.Sprintf("nightuser%03d@gmail.com", i),
			AccountCreatedAt:  base.Add(-120 * 24 * time.Hour),
			CardID:            fmt.Sprintf("card-mid-%03d", i), CardBIN: "453985",
			DeviceFingerprint: fmt.Sprintf("dev-mid-%03d", i),
			IP:                "180.190.5.5", IPCountry: "PH",
			BillingCountry:    "PH", ShippingCountry: "PH", MerchantCountry: "PH",
			MerchantID:        "merch-night", Amount: int64(900_00 + r.IntN(1_500_00)), Currency: "PHP",
			Lat: 14.60, Lng: 120.98, Timestamp: ts,
		}
	case 1:
		ts := base.Add(-time.Duration(i*11) * time.Minute)
		return scoring.Transaction{
			ID: fmt.Sprintf("mid-newacct-%03d", i), CustomerID: fmt.Sprintf("cust-newacct-%03d", i),
			CustomerEmail:     fmt.Sprintf("brandnew%03d@gmail.com", i),
			AccountCreatedAt:  ts.Add(-6 * time.Hour),
			CardID:            fmt.Sprintf("card-newacct-%03d", i), CardBIN: "453985",
			DeviceFingerprint: fmt.Sprintf("dev-newacct-%03d", i),
			IP:                "180.190.6.6", IPCountry: "PH",
			BillingCountry:    "PH", ShippingCountry: "PH", MerchantCountry: "PH",
			MerchantID:        "merch-newacct", Amount: int64(5_500_00 + r.IntN(2_000_00)), Currency: "PHP",
			Lat: 14.60, Lng: 120.98, Timestamp: ts,
		}
	default:
		ts := base.Add(-time.Duration(i*13) * time.Minute)
		return scoring.Transaction{
			ID: fmt.Sprintf("mid-curmix-%03d", i), CustomerID: fmt.Sprintf("cust-curmix-%03d", i),
			CustomerEmail:     fmt.Sprintf("curmix%03d@gmail.com", i),
			AccountCreatedAt:  base.Add(-200 * 24 * time.Hour),
			CardID:            fmt.Sprintf("card-curmix-%03d", i), CardBIN: "453985",
			DeviceFingerprint: fmt.Sprintf("dev-curmix-%03d", i),
			IP:                "180.190.7.7", IPCountry: "PH",
			BillingCountry:    "PH", ShippingCountry: "ID", MerchantCountry: "PH",
			MerchantID:        "merch-curmix", Amount: int64(1_500_00 + r.IntN(1_000_00)), Currency: "PHP",
			Lat: 14.60, Lng: 120.98, Timestamp: ts,
		}
	}
}

func highRiskCluster(base time.Time) []scoring.Transaction {
	var out []scoring.Transaction

	burstStart := base.Add(-45 * time.Minute)
	for i := range 5 {
		out = append(out, scoring.Transaction{
			ID: fmt.Sprintf("burst-%d", i), CustomerID: "cust-burst",
			CustomerEmail: "burst@tempmail.com",
			AccountCreatedAt: base.Add(-2 * time.Hour),
			CardID: "card-burst", CardBIN: "411111", DeviceFingerprint: "dev-burst",
			IP: "185.220.101.45", IPCountry: "RU",
			BillingCountry: "US", ShippingCountry: "PH", MerchantCountry: "PH",
			MerchantID: fmt.Sprintf("merch-burst-%d", i),
			Amount:     int64(2_500_00 + i*500_00), Currency: "PHP",
			Lat: 14.60, Lng: 120.98, PreviousDeclines: 2,
			Timestamp: burstStart.Add(time.Duration(i) * time.Minute),
		})
	}

	ttStart := base.Add(-25 * time.Minute)
	for i := range 3 {
		out = append(out, scoring.Transaction{
			ID: fmt.Sprintf("test-%d", i), CustomerID: "cust-test",
			CustomerEmail:     "test@guerrillamail.com",
			AccountCreatedAt:  base.Add(-1 * time.Hour),
			CardID:            "card-test", CardBIN: "424242", DeviceFingerprint: "dev-test",
			IP: "185.220.101.46", IPCountry: "RU",
			BillingCountry: "US", ShippingCountry: "PH", MerchantCountry: "PH",
			MerchantID: "merch-test", Amount: int64(5_00 + i*2_00), Currency: "PHP",
			Lat: 14.60, Lng: 120.98,
			Timestamp: ttStart.Add(time.Duration(i) * time.Minute),
		})
	}
	out = append(out, scoring.Transaction{
		ID: "test-big", CustomerID: "cust-test", CustomerEmail: "test@guerrillamail.com",
		AccountCreatedAt: base.Add(-1 * time.Hour),
		CardID:           "card-test", CardBIN: "424242", DeviceFingerprint: "dev-test",
		IP: "185.220.101.46", IPCountry: "RU",
		BillingCountry: "US", ShippingCountry: "PH", MerchantCountry: "PH",
		MerchantID: "merch-test", Amount: 15_000_00, Currency: "PHP",
		Lat: 14.60, Lng: 120.98, PreviousDeclines: 3,
		Timestamp: ttStart.Add(4 * time.Minute),
	})

	out = append(out, scoring.Transaction{
		ID: "storm", CustomerID: "cust-storm",
		CustomerEmail:    "storm@throwaway.io",
		AccountCreatedAt: base.Add(-3 * time.Hour),
		CardID:           "card-storm", CardBIN: "555555", DeviceFingerprint: "dev-storm",
		IP: "185.220.101.47", IPCountry: "RU",
		BillingCountry: "US", ShippingCountry: "VN", MerchantCountry: "PH",
		MerchantID: "merch-storm",
		Amount:     20_000_00, Currency: "USD",
		Lat: 14.5995, Lng: 120.9842, PreviousDeclines: 4,
		Timestamp: time.Date(base.Year(), base.Month(), base.Day(), 3, 45, 0, 0, time.UTC),
	})

	travelTime := base.Add(-5 * time.Minute)
	out = append(out,
		scoring.Transaction{
			ID: "travel-1", CustomerID: "cust-travel", CustomerEmail: "trav@gmail.com",
			AccountCreatedAt: base.Add(-30 * 24 * time.Hour),
			CardID:           "card-travel", CardBIN: "453985", DeviceFingerprint: "dev-travel",
			IP: "8.8.8.8", IPCountry: "US",
			BillingCountry: "US", ShippingCountry: "US", MerchantCountry: "US",
			MerchantID: "merch-nyc", Amount: 3_000_00, Currency: "USD",
			Lat: 40.7128, Lng: -74.0060, Timestamp: travelTime,
		},
		scoring.Transaction{
			ID: "travel-2", CustomerID: "cust-travel", CustomerEmail: "trav@gmail.com",
			AccountCreatedAt: base.Add(-30 * 24 * time.Hour),
			CardID:           "card-travel", CardBIN: "453985", DeviceFingerprint: "dev-travel",
			IP: "180.190.50.50", IPCountry: "PH",
			BillingCountry: "US", ShippingCountry: "PH", MerchantCountry: "PH",
			MerchantID: "merch-mnl", Amount: 4_000_00, Currency: "PHP",
			Lat: 14.6, Lng: 120.98, Timestamp: travelTime.Add(30 * time.Minute),
		},
	)

	devTime := base.Add(-15 * time.Minute)
	for i := range 5 {
		out = append(out, scoring.Transaction{
			ID: fmt.Sprintf("dev-share-%d", i), CustomerID: fmt.Sprintf("cust-share-%d", i),
			CustomerEmail:     fmt.Sprintf("share%d@gmail.com", i),
			AccountCreatedAt:  base.Add(-10 * 24 * time.Hour),
			CardID:            fmt.Sprintf("card-share-%d", i), CardBIN: "601100",
			DeviceFingerprint: "dev-shared", IP: "180.190.99.99", IPCountry: "PH",
			BillingCountry: "PH", ShippingCountry: "PH", MerchantCountry: "PH",
			MerchantID: "merch-share", Amount: int64(2_000_00 + i*500_00), Currency: "PHP",
			Lat: 14.60, Lng: 120.98,
			Timestamp: devTime.Add(time.Duration(i) * time.Minute),
		})
	}

	return out
}
