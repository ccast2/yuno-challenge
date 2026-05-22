import { useEffect, useRef, useState } from "react"

export type WSStatus = "connecting" | "open" | "closed" | "error"

export type Notification = {
  id: string
  type: string
  title: string
  message: string
  severity: "info" | "warning" | "critical" | "success" | string
  timestamp: string
  payload?: Record<string, unknown>
}

function resolveWsUrl(path: string): string {
  const proto = window.location.protocol === "https:" ? "wss:" : "ws:"
  return `${proto}//${window.location.host}${path}`
}

export function useNotificationSocket(path = "/ws") {
  const [status, setStatus] = useState<WSStatus>("connecting")
  const [notifications, setNotifications] = useState<Notification[]>([])
  const [lastEvent, setLastEvent] = useState<Notification | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const retryRef = useRef<number>(0)

  useEffect(() => {
    let cancelled = false

    const connect = () => {
      if (cancelled) return
      setStatus("connecting")
      const ws = new WebSocket(resolveWsUrl(path))
      wsRef.current = ws

      ws.onopen = () => {
        retryRef.current = 0
        setStatus("open")
      }
      ws.onmessage = (event) => {
        try {
          const n = JSON.parse(event.data) as Notification
          setNotifications((prev) => [n, ...prev].slice(0, 50))
          setLastEvent(n)
        } catch {
          // ignore malformed payload
        }
      }
      ws.onerror = () => setStatus("error")
      ws.onclose = () => {
        setStatus("closed")
        if (cancelled) return
        const delay = Math.min(15000, 500 * 2 ** retryRef.current)
        retryRef.current += 1
        window.setTimeout(connect, delay)
      }
    }

    connect()

    return () => {
      cancelled = true
      wsRef.current?.close()
    }
  }, [path])

  function clear() {
    setNotifications([])
  }

  return { status, notifications, lastEvent, clear }
}
