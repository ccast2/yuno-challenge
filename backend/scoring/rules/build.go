package rules

import "github.com/ccast2/yuno-challenge/backend/scoring"

// Build returns the full ordered list of rules wired with weights from cfg.
// Order matches the display order in the API response and in docs; weighting
// is independent of order.
func Build(cfg *scoring.Config) []scoring.Rule {
	return []scoring.Rule{
		{Code: "AMOUNT_ANOMALY", Name: "Amount anomaly", Weight: cfg.Weights["AMOUNT_ANOMALY"], Fn: Amount},
		{Code: "VELOCITY", Name: "Velocity check", Weight: cfg.Weights["VELOCITY"], Fn: Velocity},
		{Code: "IMPOSSIBLE_TRAVEL", Name: "Impossible travel", Weight: cfg.Weights["IMPOSSIBLE_TRAVEL"], Fn: ImpossibleTravel},
		{Code: "MANILA_PATTERN", Name: "Manila fraud pattern", Weight: cfg.Weights["MANILA_PATTERN"], Fn: Manila},
		{Code: "NIGHT_OWL", Name: "Night-owl transaction hour", Weight: cfg.Weights["NIGHT_OWL"], Fn: NightOwl},
		{Code: "DEVICE_ANOMALY", Name: "Device fingerprint anomaly", Weight: cfg.Weights["DEVICE_ANOMALY"], Fn: Device},
		{Code: "CURRENCY_MISMATCH", Name: "Currency / country mismatch", Weight: cfg.Weights["CURRENCY_MISMATCH"], Fn: Currency(cfg)},
		{Code: "MERCHANT_SWITCHING", Name: "Rapid merchant switching", Weight: cfg.Weights["MERCHANT_SWITCHING"], Fn: Merchant},
		{Code: "BIN_REPUTATION", Name: "High-risk card BIN", Weight: cfg.Weights["BIN_REPUTATION"], Fn: BIN(cfg)},
		{Code: "NEW_ACCOUNT_HIGH_VALUE", Name: "New account placing high-value order", Weight: cfg.Weights["NEW_ACCOUNT_HIGH_VALUE"], Fn: NewAccount},
	}
}
