import { Bar, BarChart, CartesianGrid, XAxis, YAxis } from "recharts"

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
  type ChartConfig,
} from "@/components/ui/chart"
import type { RiskLevel, Statistics } from "@/features/scoring/types"

const buckets: { key: RiskLevel; label: string; fill: string }[] = [
  { key: "low", label: "Low", fill: "var(--chart-2)" },
  { key: "medium", label: "Medium", fill: "var(--chart-3)" },
  { key: "high", label: "High", fill: "var(--chart-4)" },
  { key: "critical", label: "Critical", fill: "var(--chart-5)" },
]

const config: ChartConfig = {
  count: { label: "Transactions" },
}

type Props = {
  stats: Statistics | null
}

export function RiskDistribution({ stats }: Props) {
  const data = buckets.map((b) => ({
    bucket: b.label,
    count: stats?.by_risk_level?.[b.key] ?? 0,
    fill: b.fill,
  }))
  const total = data.reduce((s, b) => s + b.count, 0)

  return (
    <Card>
      <CardHeader>
        <CardTitle>Score distribution</CardTitle>
        <CardDescription>By risk bucket</CardDescription>
      </CardHeader>
      <CardContent>
        {total === 0 ? (
          <p className="text-sm text-muted-foreground">
            No decisions scored in this window.
          </p>
        ) : (
          <ChartContainer config={config} className="h-[360px] w-full">
            <BarChart data={data} margin={{ left: 8, right: 8 }}>
              <CartesianGrid vertical={false} />
              <XAxis
                dataKey="bucket"
                tickLine={false}
                axisLine={false}
                tickMargin={8}
              />
              <YAxis hide />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Bar dataKey="count" radius={6} />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  )
}
