package rules

import (
	"fmt"
	"testing"
	"time"

	"github.com/ccast2/yuno-challenge/backend/config"
	"github.com/ccast2/yuno-challenge/backend/scoring"
)

func testNow() time.Time {
	return time.Date(2026, 5, 21, 12, 0, 0, 0, time.UTC)
}

func testCfg(t *testing.T) *scoring.Config {
	t.Helper()
	cfg, err := config.NewFileRepository().Load()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	return cfg
}

type emptyHistory struct{}

func (emptyHistory) GetByCard(string, time.Time) []scoring.Transaction   { return nil }
func (emptyHistory) GetByDevice(string, time.Time) []scoring.Transaction { return nil }
func (emptyHistory) GetByEmail(string, time.Time) []scoring.Transaction  { return nil }
func (emptyHistory) GetByBIN(string, time.Time) []scoring.Transaction    { return nil }
func (emptyHistory) GetByIP(string, time.Time) []scoring.Transaction     { return nil }
func (emptyHistory) LastByCard(string, time.Time) (scoring.Transaction, bool) {
	return scoring.Transaction{}, false
}

func TestAmount(t *testing.T) {
	cases := []struct {
		name     string
		amount   int64
		wantGT   float64
		wantLT   float64
		wantZero bool
	}{
		{"clean small purchase", 250_00, 0, 0, true},
		{"borderline mid", 5_500_00, 50, 80, false},
		{"blowout high", 25_000_00, 99, 101, false},
		{"round number bonus", 10_000_00, 70, 101, false},
		{"card testing micro", 5_00, 25, 50, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			score, _ := Amount(scoring.Transaction{Amount: c.amount, Currency: "PHP"}, emptyHistory{}, testNow())
			if c.wantZero {
				if score != 0 {
					t.Fatalf("expected 0, got %v", score)
				}
				return
			}
			if score <= c.wantGT || score > c.wantLT {
				t.Fatalf("score %v not in (%v, %v]", score, c.wantGT, c.wantLT)
			}
		})
	}
}

func TestNightOwl(t *testing.T) {
	// Without a country code, the rule falls back to UTC hour.
	mk := func(h int) scoring.Transaction {
		return scoring.Transaction{Timestamp: time.Date(2026, 5, 21, h, 0, 0, 0, time.UTC)}
	}
	if s, _ := NightOwl(mk(12), emptyHistory{}, testNow()); s != 0 {
		t.Fatalf("noon UTC fallback should be 0, got %v", s)
	}
	if s, _ := NightOwl(mk(3), emptyHistory{}, testNow()); s < 50 {
		t.Fatalf("3AM UTC fallback should be high, got %v", s)
	}
	if s, _ := NightOwl(mk(0), emptyHistory{}, testNow()); s == 0 {
		t.Fatalf("midnight UTC fallback should be adjacent window, got 0")
	}
}

// TestNightOwlLocalTime verifies the country→timezone resolution. A txn at
// 03:00 UTC on a PH merchant is 11:00 Manila and must NOT fire; a txn at
// 19:00 UTC on a PH merchant is 03:00 Manila and MUST fire.
func TestNightOwlLocalTime(t *testing.T) {
	mk := func(utcHour int) scoring.Transaction {
		return scoring.Transaction{
			MerchantCountry: "PH",
			Timestamp:       time.Date(2026, 5, 21, utcHour, 0, 0, 0, time.UTC),
		}
	}
	if s, _ := NightOwl(mk(3), emptyHistory{}, testNow()); s != 0 {
		t.Fatalf("03:00 UTC = 11:00 Manila must not fire, got %v", s)
	}
	if s, _ := NightOwl(mk(19), emptyHistory{}, testNow()); s < 50 {
		t.Fatalf("19:00 UTC = 03:00 Manila must fire high, got %v", s)
	}
}

func TestImpossibleTravel(t *testing.T) {
	now := testNow()
	store := scoring.NewMemoryStore(time.Hour, 100)
	store.Record(scoring.Transaction{
		ID: "prev", CardID: "card1",
		Lat: 14.5995, Lng: 120.9842,
		Timestamp: now.Add(-30 * time.Minute),
	})
	current := scoring.Transaction{
		ID: "cur", CardID: "card1",
		Lat: 40.7128, Lng: -74.0060,
		Timestamp: now,
	}
	if s, _ := ImpossibleTravel(current, store, now); s < 99 {
		t.Fatalf("Manila→NYC in 30m should be 100, got %v", s)
	}

	store.Record(scoring.Transaction{ID: "p2", CardID: "c2", Lat: 14.5995, Lng: 120.9842, Timestamp: now.Add(-1 * time.Hour)})
	cur2 := scoring.Transaction{ID: "c2cur", CardID: "c2", Lat: 12.8797, Lng: 121.7740, Timestamp: now}
	s2, _ := ImpossibleTravel(cur2, store, now)
	if s2 < 40 || s2 > 80 {
		t.Fatalf("borderline travel expected 50-75, got %v", s2)
	}
}

func TestVelocityCluster(t *testing.T) {
	now := testNow()
	store := scoring.NewMemoryStore(time.Hour, 100)
	for i := range 4 {
		store.Record(scoring.Transaction{
			ID:        "t" + string(rune('a'+i)),
			CardID:    "card-spike",
			Timestamp: now.Add(-time.Duration(i+1) * time.Minute),
		})
	}
	current := scoring.Transaction{ID: "current", CardID: "card-spike", Timestamp: now}
	if s, _ := Velocity(current, store, now); s < 60 {
		t.Fatalf("4 tx in 5m should fire high, got %v", s)
	}
}

func TestManila(t *testing.T) {
	now := testNow()
	store := scoring.NewMemoryStore(time.Hour, 100)
	if s, _ := Manila(scoring.Transaction{Lat: 40, Lng: -74}, store, now); s != 0 {
		t.Fatalf("NYC tx should not fire manila, got %v", s)
	}
	s, _ := Manila(scoring.Transaction{
		Lat: 14.6, Lng: 120.98, Amount: amt10K, Timestamp: now,
		BillingCountry: "US", ShippingCountry: "PH", IPCountry: "RU",
	}, store, now)
	if s < 70 {
		t.Fatalf("Manila + geo mismatch should be >= 70, got %v", s)
	}
	if s, _ := Manila(scoring.Transaction{
		Lat: 14.6, Lng: 120.98, Amount: 500_00,
		BillingCountry: "PH", ShippingCountry: "PH", IPCountry: "PH",
		Timestamp: now,
	}, store, now); s != 0 {
		t.Fatalf("clean Manila tx must not fire, got %v", s)
	}
}

func TestEngineCleanTxApproved(t *testing.T) {
	cfg := testCfg(t)
	store := scoring.NewMemoryStore(time.Hour, 100)
	engine := scoring.NewEngine(cfg, store, Build(cfg))
	clean := scoring.Transaction{
		ID: "clean", CardID: "card-clean", Amount: 500_00, Currency: "PHP",
		Timestamp:        time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
		AccountCreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		BillingCountry:   "PH", ShippingCountry: "PH", IPCountry: "PH",
		MerchantCountry:  "PH",
	}
	res := engine.Score(clean, clean.Timestamp)
	if res.Recommendation != scoring.Approve {
		t.Fatalf("clean tx should APPROVE, got %s (score %v, rules %+v)", res.Recommendation, res.Score, res.TriggeredRules)
	}
}

// TestEngineCriticalStorm verifies that a "perfect storm" payload — with
// velocity history already in the store — produces a critical-risk decision
// with at least seven rules firing. This is the demo case reviewers run to
// confirm "see multiple risk signals firing": when N concerning patterns
// stack, the engine reaches the top bucket, not just the borderline.
func TestEngineCriticalStorm(t *testing.T) {
	cfg := testCfg(t)
	store := scoring.NewMemoryStore(time.Hour, 100)
	engine := scoring.NewEngine(cfg, store, Build(cfg))

	// timestamp at 19:00 UTC = 03:00 Manila local → fires NightOwl.
	now := time.Date(2026, 5, 21, 19, 0, 0, 0, time.UTC)

	// Pre-seed 5 history records on the same card/email/device so the storm
	// has cluster context — Velocity, Device, and Merchant rules need this.
	for i := range 5 {
		store.Record(scoring.Transaction{
			ID:                fmt.Sprintf("prep-%d", i),
			CardID:            "card-storm",
			CustomerEmail:     "bad@tempmail.com",
			DeviceFingerprint: "dev-storm",
			MerchantID:        fmt.Sprintf("merch-%d", i),
			Timestamp:         now.Add(-time.Duration(i+1) * time.Minute),
		})
	}

	storm := scoring.Transaction{
		ID:                "storm",
		CardID:            "card-storm",
		CardBIN:           "411111",
		CustomerEmail:     "bad@tempmail.com",
		DeviceFingerprint: "dev-storm",
		Amount:            20_000_00,
		Currency:          "USD",
		Lat:               14.6, Lng: 120.98,
		BillingCountry:   "US",
		ShippingCountry:  "VN",
		IPCountry:        "RU",
		MerchantCountry:  "PH",
		MerchantID:       "merch-storm",
		AccountCreatedAt: now.Add(-2 * time.Hour),
		Timestamp:        now,
		PreviousDeclines: 3,
	}
	res := engine.Score(storm, now)

	if res.Score < 80 {
		t.Fatalf("storm score should be >= 80, got %v · rules: %+v", res.Score, ruleCodes(res.TriggeredRules))
	}
	if res.RiskLevel != scoring.RiskCritical {
		t.Fatalf("storm risk_level should be critical, got %s (score %v)", res.RiskLevel, res.Score)
	}
	if res.Recommendation != scoring.Block {
		t.Fatalf("storm recommendation should be BLOCK, got %s", res.Recommendation)
	}
	if len(res.TriggeredRules) < 7 {
		t.Fatalf("storm should trigger >= 7 rules, got %d: %v",
			len(res.TriggeredRules), ruleCodes(res.TriggeredRules))
	}
}

func ruleCodes(rules []scoring.ReasonCode) []string {
	out := make([]string, len(rules))
	for i, r := range rules {
		out[i] = r.Code
	}
	return out
}

func TestEngineNoRulesFireIsZero(t *testing.T) {
	cfg := testCfg(t)
	store := scoring.NewMemoryStore(time.Hour, 100)
	engine := scoring.NewEngine(cfg, store, Build(cfg))
	res := engine.Score(scoring.Transaction{
		ID: "boring", Amount: 100_00, Currency: "PHP",
		Timestamp:        time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC),
		AccountCreatedAt: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}, time.Date(2026, 5, 21, 14, 0, 0, 0, time.UTC))
	if res.Score != 0 || res.Recommendation != scoring.Approve {
		t.Fatalf("boring tx should score 0 / APPROVE, got %v / %s", res.Score, res.Recommendation)
	}
}

func TestStoreResetClearsStateAndStats(t *testing.T) {
	cfg := testCfg(t)
	store := scoring.NewMemoryStore(time.Hour, 100)
	engine := scoring.NewEngine(cfg, store, Build(cfg))
	now := time.Now()

	txn := scoring.Transaction{
		ID: "to-reset", CardID: "card-x", Amount: 25_000_00, Currency: "USD",
		Timestamp:        now,
		AccountCreatedAt: now.Add(-1 * time.Hour),
		PreviousDeclines: 3,
	}
	res := engine.Score(txn, now)
	store.Record(txn)
	store.SaveDecision(txn, res)

	if got := store.Statistics(now.Add(-1 * time.Hour)).TotalScored; got != 1 {
		t.Fatalf("expected 1 scored after save, got %d", got)
	}

	store.Reset()

	if got := store.Statistics(now.Add(-1 * time.Hour)).TotalScored; got != 0 {
		t.Fatalf("expected 0 scored after reset, got %d", got)
	}
	if _, ok := store.GetDecision("to-reset"); ok {
		t.Fatal("expected GetDecision miss after reset, got hit")
	}
	if got := store.GetByCard("card-x", now.Add(-1*time.Hour)); len(got) != 0 {
		t.Fatalf("expected card history empty after reset, got %d entries", len(got))
	}
}

func TestEngineScoreCapped(t *testing.T) {
	cfg := testCfg(t)
	store := scoring.NewMemoryStore(time.Hour, 100)
	engine := scoring.NewEngine(cfg, store, Build(cfg))
	res := engine.Score(scoring.Transaction{
		ID: "max", Amount: 100_000_00, Currency: "USD",
		CardBIN: "411111", CustomerEmail: "a@tempmail.com",
		Lat: 14.6, Lng: 120.98, BillingCountry: "US", ShippingCountry: "PH", IPCountry: "RU",
		Timestamp:        time.Date(2026, 5, 21, 3, 0, 0, 0, time.UTC),
		AccountCreatedAt: time.Date(2026, 5, 21, 2, 0, 0, 0, time.UTC),
		PreviousDeclines: 5,
	}, time.Date(2026, 5, 21, 3, 0, 0, 0, time.UTC))
	if res.Score > 100 {
		t.Fatalf("score must be capped at 100, got %v", res.Score)
	}
}
