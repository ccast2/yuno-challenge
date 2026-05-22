import { CheckCircle2 } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { severityVariant } from "@/features/scoring/format"
import type { Notification, WSStatus } from "@/lib/useWebSocket"

type Props = {
  status: WSStatus
  notifications: Notification[]
  onClear: () => void
}

export function NotificationsPanel({ status, notifications, onClear }: Props) {
  return (
    <Card data-testid="notifications-panel">
      <CardHeader className="flex flex-row items-center justify-between space-y-0">
        <div>
          <CardTitle>Live notifications</CardTitle>
          <CardDescription>
            Pushed via /ws · {notifications.length} event
            {notifications.length === 1 ? "" : "s"}
          </CardDescription>
        </div>
        <Button
          variant="ghost"
          size="sm"
          onClick={onClear}
          disabled={notifications.length === 0}
        >
          Clear
        </Button>
      </CardHeader>
      <CardContent className="max-h-[480px] space-y-3 overflow-y-auto">
        {notifications.length === 0 && (
          <div className="flex items-center gap-2 text-sm text-muted-foreground">
            <CheckCircle2 className="size-4 text-emerald-500" />
            {status === "open"
              ? "Waiting for transaction.scored events."
              : `Socket ${status} — events will appear once it's open.`}
          </div>
        )}
        {notifications.map((n, idx) => (
          <div key={`${n.id}-${idx}`} className="space-y-1">
            <div className="flex items-center justify-between gap-2">
              <span className="text-sm font-medium">{n.title}</span>
              <Badge variant={severityVariant(n.severity)}>{n.severity}</Badge>
            </div>
            <p className="text-xs text-muted-foreground">{n.message}</p>
            {n.payload?.transaction_id ? (
              <p className="font-mono text-[10px] text-muted-foreground">
                {String(n.payload.transaction_id)} ·{" "}
                {new Date(n.timestamp).toLocaleTimeString()}
              </p>
            ) : (
              <p className="font-mono text-[10px] text-muted-foreground">
                {new Date(n.timestamp).toLocaleTimeString()}
              </p>
            )}
            {idx < notifications.length - 1 && <Separator className="mt-2" />}
          </div>
        ))}
      </CardContent>
    </Card>
  )
}
