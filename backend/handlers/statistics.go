package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

type Statistics struct {
	Store scoring.DecisionReader
	Now   func() time.Time
}

func (s *Statistics) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/statistics", s.Handle)
}

func (s *Statistics) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now().UTC()
}

func (s *Statistics) Handle(w http.ResponseWriter, r *http.Request) {
	window := time.Hour
	if v := r.URL.Query().Get("window_minutes"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			window = time.Duration(n) * time.Minute
		}
	}
	since := s.now().Add(-window)
	writeJSON(w, http.StatusOK, s.Store.Statistics(since))
}
