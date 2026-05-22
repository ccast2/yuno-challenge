package scoring

import (
	"sort"
	"time"
)

type Engine struct {
	rules   []Rule
	cfg     *Config
	history HistoryReader
}

// NewEngine wires the engine with caller-supplied rules. The rule set is
// assembled by the rules subpackage (rules.Build) and injected here so the
// engine has no dependency on individual rule implementations.
func NewEngine(cfg *Config, history HistoryReader, ruleSet []Rule) *Engine {
	return &Engine{
		rules:   ruleSet,
		cfg:     cfg,
		history: history,
	}
}

// Score evaluates a transaction against all rules and produces a final decision.
// Determinism: rule outputs depend only on (txn, history, now) — no time.Now() inside.
// Latency is measured separately against the wall clock for observability.
func (e *Engine) Score(txn Transaction, now time.Time) ScoreResult {
	wallStart := time.Now()
	triggered := make([]ReasonCode, 0, len(e.rules))
	var weightedSum, firedWeight float64

	for _, r := range e.rules {
		s, desc := r.Fn(txn, e.history, now)
		if s <= 0 {
			continue
		}
		if s > 100 {
			s = 100
		}
		triggered = append(triggered, ReasonCode{
			Code:        r.Code,
			Description: desc,
			Score:       s,
			Weight:      r.Weight,
		})
		weightedSum += s * r.Weight
		firedWeight += r.Weight
	}

	final := 0.0
	if firedWeight > 0 {
		final = weightedSum / firedWeight
	}
	if final > 100 {
		final = 100
	}

	sort.Slice(triggered, func(i, j int) bool {
		return triggered[i].Score*triggered[i].Weight > triggered[j].Score*triggered[j].Weight
	})

	level := e.riskLevel(final)
	return ScoreResult{
		TransactionID:  txn.ID,
		Score:          round1(final),
		RiskLevel:      level,
		Recommendation: e.recommendation(level),
		TriggeredRules: triggered,
		// ProcessedAt is wall-clock — when scoring ran, not when the txn happened.
		// Rule evaluations use `now` for determinism; bookkeeping uses real time
		// so stats windows behave correctly with historical seed data.
		ProcessedAt: wallStart,
		LatencyMs:   time.Since(wallStart).Milliseconds(),
	}
}

func (e *Engine) Rules() []Rule { return e.rules }

func (e *Engine) riskLevel(score float64) RiskLevel {
	t := e.cfg.Thresholds
	switch {
	case score >= t.Critical:
		return RiskCritical
	case score >= t.High:
		return RiskHigh
	case score >= t.Medium:
		return RiskMedium
	}
	return RiskLow
}

func (e *Engine) recommendation(level RiskLevel) Recommendation {
	switch level {
	case RiskCritical, RiskHigh:
		return Block
	case RiskMedium:
		return Review
	}
	return Approve
}

func round1(v float64) float64 {
	return float64(int(v*10+0.5)) / 10
}
