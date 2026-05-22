// Package config provides repository implementations for the scoring engine's
// runtime configuration. The current implementation is file-backed (embedded
// JSON with optional on-disk override and env-var overrides). A future
// implementation can swap to Postgres or any other store as long as it
// satisfies scoring.ConfigRepository.
package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

//go:embed scoring.json
var defaultScoringConfig []byte

// FileRepository loads scoring config from an embedded JSON document and
// optionally overrides it with a file referenced by SCORING_CONFIG_PATH.
// Individual values can be further tuned with SCORING_THRESHOLD_* and
// SCORING_WEIGHT_<CODE> env vars without redeploying.
//
// Stateless: Load reads from disk on every call so an operator can edit the
// override file and the next reload picks it up. The hot path does not call
// Load — wire it once at boot or behind a cached reloader.
type FileRepository struct {
	OverridePath string // SCORING_CONFIG_PATH at construction time
}

func NewFileRepository() *FileRepository {
	return &FileRepository{OverridePath: os.Getenv("SCORING_CONFIG_PATH")}
}

func (r *FileRepository) Load() (*scoring.Config, error) {
	raw := defaultScoringConfig
	if r.OverridePath != "" {
		data, err := os.ReadFile(r.OverridePath)
		if err != nil {
			return nil, fmt.Errorf("read scoring config %s: %w", r.OverridePath, err)
		}
		raw = data
	}
	var cfg scoring.Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("parse scoring config: %w", err)
	}
	applyEnvOverrides(&cfg)
	return &cfg, nil
}

func applyEnvOverrides(cfg *scoring.Config) {
	if v, ok := envFloat("SCORING_THRESHOLD_MEDIUM"); ok {
		cfg.Thresholds.Medium = v
	}
	if v, ok := envFloat("SCORING_THRESHOLD_HIGH"); ok {
		cfg.Thresholds.High = v
	}
	if v, ok := envFloat("SCORING_THRESHOLD_CRITICAL"); ok {
		cfg.Thresholds.Critical = v
	}
	for code := range cfg.Weights {
		if v, ok := envFloat("SCORING_WEIGHT_" + code); ok {
			cfg.Weights[code] = v
		}
	}
}

func envFloat(key string) (float64, bool) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, false
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}
