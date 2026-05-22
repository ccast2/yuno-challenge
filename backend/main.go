package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/ccast2/yuno-challenge/backend/config"
	"github.com/ccast2/yuno-challenge/backend/handlers"
	"github.com/ccast2/yuno-challenge/backend/scoring"
	"github.com/ccast2/yuno-challenge/backend/scoring/rules"
)

func main() {
	port := envOr("PORT", "8080")
	staticDir := envOr("STATIC_DIR", "./web")

	cfgRepo := config.NewFileRepository()
	cfg, err := cfgRepo.Load()
	if err != nil {
		log.Fatalf("scoring config: %v", err)
	}
	store := scoring.NewMemoryStore(time.Hour, 10_000)
	store.StartCleanup(30 * time.Second)
	engine := scoring.NewEngine(cfg, store, rules.Build(cfg))

	hub := NewHub()
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ok",
			"time":       time.Now().UTC().Format(time.RFC3339),
			"ws_clients": hub.Count(),
		})
	})

	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, cfg)
	})

	txnHandler := &handlers.Transactions{
		Engine:  engine,
		History: store,
		Store:   store,
		Notify:  func(result scoring.ScoreResult, txn scoring.Transaction) { broadcastDecision(hub, result, txn) },
	}
	txnHandler.Register(mux)

	statsHandler := &handlers.Statistics{Store: store}
	statsHandler.Register(mux)

	generateHandler := &handlers.Generate{Engine: engine, History: store, Store: store}
	generateHandler.Register(mux)

	datasetsHandler := &handlers.Datasets{}
	datasetsHandler.Register(mux)

	resetHandler := &handlers.Reset{Store: store}
	resetHandler.Register(mux)

	mux.HandleFunc("GET /ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
			OriginPatterns:     []string{"*"},
			InsecureSkipVerify: true,
		})
		if err != nil {
			log.Printf("ws accept error: %v", err)
			return
		}
		ctx := r.Context()
		hub.ServeWS(ctx, conn)
		conn.Close(websocket.StatusNormalClosure, "bye")
	})

	mux.Handle("/", spaHandler(staticDir))

	addr := ":" + port
	log.Printf("listening on %s (static: %s)", addr, staticDir)
	srv := &http.Server{
		Addr:              addr,
		Handler:           withLogging(mux),
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func broadcastDecision(hub *Hub, result scoring.ScoreResult, txn scoring.Transaction) {
	hub.Broadcast(Notification{
		ID:        randomID(),
		Type:      "transaction.scored",
		Title:     "Transaction scored",
		Message:   string(result.Recommendation) + " — score " + jsonNumber(result.Score),
		Severity:  severityFor(result.RiskLevel),
		Timestamp: time.Now().UTC(),
		Payload: map[string]any{
			"transaction_id": txn.ID,
			"score":          result.Score,
			"risk_level":     result.RiskLevel,
			"recommendation": result.Recommendation,
			"rules":          result.TriggeredRules,
		},
	})
}

func severityFor(level scoring.RiskLevel) string {
	switch level {
	case scoring.RiskCritical, scoring.RiskHigh:
		return "error"
	case scoring.RiskMedium:
		return "warn"
	}
	return "info"
}

func jsonNumber(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func withLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func spaHandler(root string) http.Handler {
	fs := http.FileServer(http.Dir(root))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == "/ws" {
			http.NotFound(w, r)
			return
		}
		path := filepath.Join(root, filepath.Clean(r.URL.Path))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(root, "index.html"))
	})
}

func randomID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
