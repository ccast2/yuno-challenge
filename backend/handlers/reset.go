package handlers

import "net/http"

// Resetter is the surface a Reset handler needs from a store. Defined here as
// an interface so the handler doesn't depend on the concrete MemoryStore.
type Resetter interface {
	Reset()
}

type Reset struct {
	Store Resetter
}

func (h *Reset) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/reset", h.Handle)
}

func (h *Reset) Handle(w http.ResponseWriter, _ *http.Request) {
	h.Store.Reset()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
