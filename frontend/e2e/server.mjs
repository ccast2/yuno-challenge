// Lightweight stand-in for the Go backend during e2e tests.
// Serves the built SPA from ./dist, exposes /api/webhooks/test that broadcasts
// over a WebSocket on /ws, so the frontend can be tested without the Go binary.

import http from "node:http"
import { createReadStream, statSync } from "node:fs"
import { resolve, join, normalize } from "node:path"
import { WebSocketServer } from "ws"

const port = Number(process.env.PORT ?? 4173)
const distDir = resolve(process.cwd(), "dist")

const mime = {
  ".html": "text/html; charset=utf-8",
  ".js": "application/javascript",
  ".mjs": "application/javascript",
  ".css": "text/css",
  ".svg": "image/svg+xml",
  ".png": "image/png",
  ".jpg": "image/jpeg",
  ".woff2": "font/woff2",
  ".json": "application/json",
  ".ico": "image/x-icon",
}

const clients = new Set()

function broadcast(notification) {
  const payload = JSON.stringify(notification)
  for (const ws of clients) {
    if (ws.readyState === ws.OPEN) ws.send(payload)
  }
}

function serveStatic(req, res) {
  const url = decodeURIComponent(req.url.split("?")[0])
  let filePath = join(distDir, normalize(url))
  try {
    const stat = statSync(filePath)
    if (stat.isDirectory()) throw new Error("dir")
  } catch {
    filePath = join(distDir, "index.html")
  }
  const ext = filePath.slice(filePath.lastIndexOf("."))
  res.setHeader("Content-Type", mime[ext] ?? "application/octet-stream")
  createReadStream(filePath).pipe(res)
}

const server = http.createServer((req, res) => {
  if (req.url === "/api/health") {
    res.setHeader("Content-Type", "application/json")
    res.end(
      JSON.stringify({
        status: "ok",
        time: new Date().toISOString(),
        ws_clients: clients.size,
      }),
    )
    return
  }

  if (req.method === "POST" && req.url === "/api/webhooks/test") {
    let body = ""
    req.on("data", (chunk) => (body += chunk))
    req.on("end", () => {
      let parsed = {}
      try {
        parsed = body ? JSON.parse(body) : {}
      } catch {}
      const notification = {
        id: Math.random().toString(36).slice(2, 10),
        type: "webhook.test",
        title: parsed.title ?? "Test webhook",
        message: parsed.message ?? "triggered from e2e server",
        severity: parsed.severity ?? "info",
        timestamp: new Date().toISOString(),
        payload: parsed.payload,
      }
      broadcast(notification)
      res.setHeader("Content-Type", "application/json")
      res.end(
        JSON.stringify({
          ok: true,
          delivered: clients.size,
          connected_clients: clients.size,
          notification,
        }),
      )
    })
    return
  }

  if (req.url.startsWith("/api/")) {
    res.statusCode = 404
    res.end()
    return
  }

  serveStatic(req, res)
})

const wss = new WebSocketServer({ server, path: "/ws" })
wss.on("connection", (ws) => {
  clients.add(ws)
  ws.send(
    JSON.stringify({
      id: "hello",
      type: "connected",
      title: "Connected",
      message: "WebSocket connection established",
      severity: "info",
      timestamp: new Date().toISOString(),
    }),
  )
  ws.on("close", () => clients.delete(ws))
})

server.listen(port, () => {
  console.log(`e2e server listening on http://localhost:${port}`)
})
