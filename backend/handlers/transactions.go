package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/ccast2/yuno-challenge/backend/scoring"
)

type Transactions struct {
	Engine  *scoring.Engine
	History scoring.HistoryWriter
	Store   interface {
		scoring.DecisionWriter
		scoring.DecisionReader
	}
	Notify func(result scoring.ScoreResult, txn scoring.Transaction)
	Now    func() time.Time
}

func (t *Transactions) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/transactions", t.Score)
	mux.HandleFunc("GET /api/transactions", t.List)
	mux.HandleFunc("GET /api/transactions/{id}", t.Get)
}

func (t *Transactions) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now().UTC()
}

func (t *Transactions) Score(w http.ResponseWriter, r *http.Request) {
	var txn scoring.Transaction
	if err := json.NewDecoder(r.Body).Decode(&txn); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := validateTransaction(&txn, t.now()); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_transaction", err.Error())
		return
	}

	result := t.Engine.Score(txn, t.now())
	t.History.Record(txn)
	t.Store.SaveDecision(txn, result)
	if t.Notify != nil {
		t.Notify(result, txn)
	}
	writeJSON(w, http.StatusOK, result)
}

func (t *Transactions) List(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	risk := r.URL.Query().Get("risk_level")
	writeJSON(w, http.StatusOK, t.Store.RecentDecisions(limit, risk))
}

func (t *Transactions) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	d, ok := t.Store.GetDecision(id)
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "transaction not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func validateTransaction(txn *scoring.Transaction, now time.Time) error {
	if txn.ID == "" {
		return errors.New("id is required")
	}
	if txn.Amount <= 0 {
		return errors.New("amount must be > 0 (minor units)")
	}
	if txn.Currency == "" {
		return errors.New("currency is required")
	}
	if txn.Timestamp.IsZero() {
		txn.Timestamp = now
	}
	return nil
}
