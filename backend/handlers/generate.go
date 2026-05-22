package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ccast2/yuno-challenge/backend/datagen"
	"github.com/ccast2/yuno-challenge/backend/scoring"
)

type Generate struct {
	Engine  *scoring.Engine
	History scoring.HistoryWriter
	Store   scoring.DecisionWriter
}

func (g *Generate) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/transactions/generate", g.Handle)
}

type generateRequest struct {
	Count int    `json:"count"`
	Bias  string `json:"bias"`
}

func (g *Generate) Handle(w http.ResponseWriter, r *http.Request) {
	var body generateRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && r.ContentLength > 0 {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if body.Count <= 0 {
		body.Count = 183
	}
	if body.Count > 5000 {
		writeError(w, http.StatusBadRequest, "count_too_large", "count must be <= 5000")
		return
	}
	bias, err := parseBias(body.Bias)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_bias", err.Error())
		return
	}

	txns := datagen.Generate(datagen.Options{Total: body.Count, Bias: bias})
	summary := map[string]int{"APPROVE": 0, "REVIEW": 0, "BLOCK": 0}
	for _, txn := range txns {
		result := g.Engine.Score(txn, txn.Timestamp)
		g.History.Record(txn)
		g.Store.SaveDecision(txn, result)
		summary[string(result.Recommendation)]++
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"generated":         len(txns),
		"by_recommendation": summary,
		"bias":              string(bias),
	})
}
