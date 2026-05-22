package scoring

// Thresholds maps a final 0–100 score to a risk bucket. Scores >= Critical
// become BLOCK with risk_level=critical; >= High become BLOCK with high;
// >= Medium become REVIEW; everything below is APPROVE.
type Thresholds struct {
	Medium   float64 `json:"medium"`
	High     float64 `json:"high"`
	Critical float64 `json:"critical"`
}

// Config holds the tunable knobs of the scoring engine: per-rule weights,
// risk-bucket thresholds, and lookup tables consumed by individual rules.
// It is a pure data type — loading and persistence belong to a repository.
type Config struct {
	Weights          map[string]float64 `json:"weights"`
	Thresholds       Thresholds         `json:"thresholds"`
	HighRiskBINs     []string           `json:"high_risk_bins"`
	TempEmailDomains []string           `json:"temp_email_domains"`
	CountryCurrency  map[string]string  `json:"country_currency"`
}

// ConfigRepository is the contract for retrieving the active scoring Config.
// The current implementation reads from an embedded JSON file with env-var
// overrides; a future implementation can read from Postgres without changing
// any code that depends on this interface.
type ConfigRepository interface {
	Load() (*Config, error)
}
