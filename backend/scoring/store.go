package scoring

import (
	"sort"
	"sync"
	"time"
)

type HistoryReader interface {
	GetByCard(cardID string, since time.Time) []Transaction
	GetByDevice(device string, since time.Time) []Transaction
	GetByEmail(email string, since time.Time) []Transaction
	GetByBIN(bin string, since time.Time) []Transaction
	GetByIP(ip string, since time.Time) []Transaction
	LastByCard(cardID string, before time.Time) (Transaction, bool)
}

type HistoryWriter interface {
	Record(txn Transaction)
}

type DecisionWriter interface {
	SaveDecision(txn Transaction, result ScoreResult)
}

type DecisionReader interface {
	RecentDecisions(limit int, riskLevel string) []StoredDecision
	GetDecision(id string) (StoredDecision, bool)
	Statistics(since time.Time) Statistics
}

type StoredDecision struct {
	Transaction Transaction `json:"transaction"`
	Result      ScoreResult `json:"result"`
}

type Statistics struct {
	TotalScored      int                `json:"total_scored"`
	ByRecommendation map[string]int     `json:"by_recommendation"`
	ByRiskLevel      map[string]int     `json:"by_risk_level"`
	AverageScore     float64            `json:"average_score"`
	AverageLatencyMs float64            `json:"average_latency_ms"`
	TopRules         []RuleFrequency    `json:"top_rules"`
	WindowSince      time.Time          `json:"window_since"`
}

type RuleFrequency struct {
	Code  string `json:"code"`
	Count int    `json:"count"`
}

type MemoryStore struct {
	mu        sync.RWMutex
	byID      map[string]StoredDecision
	byCard    map[string][]Transaction
	byDevice  map[string][]Transaction
	byEmail   map[string][]Transaction
	byBIN     map[string][]Transaction
	byIP      map[string][]Transaction
	recent    []StoredDecision
	retention time.Duration
	maxRecent int
}

func NewMemoryStore(retention time.Duration, maxRecent int) *MemoryStore {
	if retention <= 0 {
		retention = time.Hour
	}
	if maxRecent <= 0 {
		maxRecent = 10000
	}
	return &MemoryStore{
		byID:      make(map[string]StoredDecision),
		byCard:    make(map[string][]Transaction),
		byDevice:  make(map[string][]Transaction),
		byEmail:   make(map[string][]Transaction),
		byBIN:     make(map[string][]Transaction),
		byIP:      make(map[string][]Transaction),
		recent:    make([]StoredDecision, 0, maxRecent),
		retention: retention,
		maxRecent: maxRecent,
	}
}

// Reset wipes all decisions and history, returning the store to an empty
// state. Useful for the dashboard's "reset" action so a demo run does not
// carry stats from a previous session.
func (s *MemoryStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID = make(map[string]StoredDecision)
	s.byCard = make(map[string][]Transaction)
	s.byDevice = make(map[string][]Transaction)
	s.byEmail = make(map[string][]Transaction)
	s.byBIN = make(map[string][]Transaction)
	s.byIP = make(map[string][]Transaction)
	s.recent = s.recent[:0]
}

func (s *MemoryStore) Record(txn Transaction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	appendIfKey(&s.byCard, txn.CardID, txn)
	appendIfKey(&s.byDevice, txn.DeviceFingerprint, txn)
	appendIfKey(&s.byEmail, txn.CustomerEmail, txn)
	appendIfKey(&s.byBIN, txn.CardBIN, txn)
	appendIfKey(&s.byIP, txn.IP, txn)
}

func (s *MemoryStore) SaveDecision(txn Transaction, result ScoreResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := StoredDecision{Transaction: txn, Result: result}
	s.byID[txn.ID] = stored
	s.recent = append(s.recent, stored)
	if len(s.recent) > s.maxRecent {
		s.recent = s.recent[len(s.recent)-s.maxRecent:]
	}
}

func (s *MemoryStore) GetByCard(cardID string, since time.Time) []Transaction {
	return s.filterIndex(s.byCard, cardID, since)
}
func (s *MemoryStore) GetByDevice(device string, since time.Time) []Transaction {
	return s.filterIndex(s.byDevice, device, since)
}
func (s *MemoryStore) GetByEmail(email string, since time.Time) []Transaction {
	return s.filterIndex(s.byEmail, email, since)
}
func (s *MemoryStore) GetByBIN(bin string, since time.Time) []Transaction {
	return s.filterIndex(s.byBIN, bin, since)
}
func (s *MemoryStore) GetByIP(ip string, since time.Time) []Transaction {
	return s.filterIndex(s.byIP, ip, since)
}

func (s *MemoryStore) LastByCard(cardID string, before time.Time) (Transaction, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := s.byCard[cardID]
	var last Transaction
	found := false
	for _, t := range entries {
		if t.Timestamp.Before(before) && (!found || t.Timestamp.After(last.Timestamp)) {
			last = t
			found = true
		}
	}
	return last, found
}

func (s *MemoryStore) filterIndex(index map[string][]Transaction, key string, since time.Time) []Transaction {
	if key == "" {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := index[key]
	out := make([]Transaction, 0, len(entries))
	for _, t := range entries {
		if !t.Timestamp.Before(since) {
			out = append(out, t)
		}
	}
	return out
}

func (s *MemoryStore) RecentDecisions(limit int, riskLevel string) []StoredDecision {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if limit <= 0 || limit > len(s.recent) {
		limit = len(s.recent)
	}
	out := make([]StoredDecision, 0, limit)
	for i := len(s.recent) - 1; i >= 0 && len(out) < limit; i-- {
		d := s.recent[i]
		if riskLevel != "" && string(d.Result.RiskLevel) != riskLevel {
			continue
		}
		out = append(out, d)
	}
	return out
}

func (s *MemoryStore) GetDecision(id string) (StoredDecision, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.byID[id]
	return d, ok
}

func (s *MemoryStore) Statistics(since time.Time) Statistics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := Statistics{
		ByRecommendation: map[string]int{},
		ByRiskLevel:      map[string]int{},
		WindowSince:      since,
	}
	ruleCounts := map[string]int{}
	var sumScore, sumLatency float64
	for _, d := range s.recent {
		if d.Result.ProcessedAt.Before(since) {
			continue
		}
		stats.TotalScored++
		stats.ByRecommendation[string(d.Result.Recommendation)]++
		stats.ByRiskLevel[string(d.Result.RiskLevel)]++
		sumScore += d.Result.Score
		sumLatency += float64(d.Result.LatencyMs)
		for _, r := range d.Result.TriggeredRules {
			ruleCounts[r.Code]++
		}
	}
	if stats.TotalScored > 0 {
		stats.AverageScore = sumScore / float64(stats.TotalScored)
		stats.AverageLatencyMs = sumLatency / float64(stats.TotalScored)
	}
	stats.TopRules = topRules(ruleCounts, 10)
	return stats
}

// StartCleanup launches a background goroutine that purges entries older than retention.
// Returns a stop func.
func (s *MemoryStore) StartCleanup(interval time.Duration) func() {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				s.purgeExpired(time.Now())
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	return func() { close(done) }
}

func (s *MemoryStore) purgeExpired(now time.Time) {
	cutoff := now.Add(-s.retention)
	s.mu.Lock()
	defer s.mu.Unlock()
	purgeIndex(s.byCard, cutoff)
	purgeIndex(s.byDevice, cutoff)
	purgeIndex(s.byEmail, cutoff)
	purgeIndex(s.byBIN, cutoff)
	purgeIndex(s.byIP, cutoff)
}

func appendIfKey(index *map[string][]Transaction, key string, txn Transaction) {
	if key == "" {
		return
	}
	(*index)[key] = append((*index)[key], txn)
}

func purgeIndex(index map[string][]Transaction, cutoff time.Time) {
	for k, entries := range index {
		kept := entries[:0]
		for _, t := range entries {
			if !t.Timestamp.Before(cutoff) {
				kept = append(kept, t)
			}
		}
		if len(kept) == 0 {
			delete(index, k)
		} else {
			index[k] = kept
		}
	}
}

func topRules(counts map[string]int, n int) []RuleFrequency {
	out := make([]RuleFrequency, 0, len(counts))
	for code, c := range counts {
		out = append(out, RuleFrequency{Code: code, Count: c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > n {
		out = out[:n]
	}
	return out
}
