package main

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Notification struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	Severity  string         `json:"severity"`
	Timestamp time.Time      `json:"timestamp"`
	Payload   map[string]any `json:"payload,omitempty"`
}

type client struct {
	conn *websocket.Conn
	send chan Notification
}

type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

func (h *Hub) add(c *client) {
	h.mu.Lock()
	h.clients[c] = struct{}{}
	h.mu.Unlock()
}

func (h *Hub) remove(c *client) {
	h.mu.Lock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	h.mu.Unlock()
}

func (h *Hub) Broadcast(n Notification) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	delivered := 0
	for c := range h.clients {
		select {
		case c.send <- n:
			delivered++
		default:
			// client too slow; drop the message rather than block the hub
		}
	}
	return delivered
}

func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) ServeWS(ctx context.Context, conn *websocket.Conn) {
	c := &client{conn: conn, send: make(chan Notification, 16)}
	h.add(c)
	defer h.remove(c)

	connCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// reader: discards anything from the client but detects close
	go func() {
		defer cancel()
		for {
			if _, _, err := conn.Read(connCtx); err != nil {
				return
			}
		}
	}()

	// send a hello
	hello := Notification{
		ID:        "hello",
		Type:      "connected",
		Title:     "Connected",
		Message:   "WebSocket connection established",
		Severity:  "info",
		Timestamp: time.Now().UTC(),
	}
	_ = writeJSONMessage(connCtx, conn, hello)

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-connCtx.Done():
			return
		case n, ok := <-c.send:
			if !ok {
				return
			}
			if err := writeJSONMessage(connCtx, conn, n); err != nil {
				return
			}
		case <-ping.C:
			pctx, pcancel := context.WithTimeout(connCtx, 10*time.Second)
			err := conn.Ping(pctx)
			pcancel()
			if err != nil {
				return
			}
		}
	}
}

func writeJSONMessage(ctx context.Context, conn *websocket.Conn, v any) error {
	wctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("ws marshal error: %v", err)
		return err
	}
	return conn.Write(wctx, websocket.MessageText, b)
}
