package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/ccast2/yuno-challenge/backend/datagen"
)

// Datasets exposes a dry-run endpoint that produces a synthetic dataset
// without scoring it. Useful to grab a JSON file you can later feed back via
// the bulk-upload flow, or share with other tools.
type Datasets struct{}

func (d *Datasets) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/datasets", d.Handle)
}

func (d *Datasets) Handle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Count int    `json:"count"`
		Bias  string `json:"bias"`
	}
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
	writeJSON(w, http.StatusOK, txns)
}
