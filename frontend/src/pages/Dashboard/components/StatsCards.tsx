import { Activity, AlertTriangle, ShieldAlert, Zap } from "lucide-react"

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import type { Statistics } from "@/features/scoring/types"

type Props = {
  stats: Statistics | null
}

export function StatsCards({ stats }: Props) {
  const total = stats?.total_scored ?? 0
  const block = stats?.by_recommendation?.BLOCK ?? 0
  const review = stats?.by_recommendation?.REVIEW ?? 0
  const blockRate = total ? ((block / total) * 100).toFixed(1) : "0.0"
  const reviewRate = total ? ((review / total) * 100).toFixed(1) : "0.0"
  const avgScore = stats ? stats.average_score.toFixed(1) : "—"
  const avgLatency = stats ? Math.round(stats.average_latency_ms) : 0

  const cards = [
    {
      label: "Transactions scored",
      value: total.toLocaleString(),
      sub: stats?.window_since
        ? `since ${new Date(stats.window_since).toLocaleTimeString()}`
        : "no data yet",
      icon: Activity,
      accent: "text-emerald-500",
    },
    {
      label: "Block rate",
      value: `${blockRate}%`,
      sub: `${block.toLocaleString()} BLOCK · ${review.toLocaleString()} REVIEW (${reviewRate}%)`,
      icon: AlertTriangle,
      accent: "text-rose-500",
    },
    {
      label: "Avg. score",
      value: avgScore,
      sub: "0–100, weighted by fired rules",
      icon: ShieldAlert,
      accent: "text-indigo-500",
    },
    {
      label: "Avg. latency",
      value: `${avgLatency} ms`,
      sub: "engine end-to-end",
      icon: Zap,
      accent: "text-amber-500",
    },
  ]

  return (
    <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4">
      {cards.map((c) => (
        <Card key={c.label} data-testid={`stat-${c.label}`}>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium text-muted-foreground">
              {c.label}
            </CardTitle>
            <c.icon className={`size-4 ${c.accent}`} />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold">{c.value}</div>
            <p className="text-xs text-muted-foreground">{c.sub}</p>
          </CardContent>
        </Card>
      ))}
    </section>
  )
}
