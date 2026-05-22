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
import { ruleLabel } from "@/features/scoring/format"
import type { Statistics } from "@/features/scoring/types"

const config: ChartConfig = {
  count: { label: "Activations", color: "var(--chart-1)" },
}

type Props = {
  stats: Statistics | null
}

export function RuleActivations({ stats }: Props) {
  const data = (stats?.top_rules ?? []).map((r) => ({
    rule: ruleLabel(r.code),
    count: r.count,
  }))

  return (
    <Card className="lg:col-span-2">
      <CardHeader>
        <CardTitle>Rule activations</CardTitle>
        <CardDescription>
          Number of transactions in the window that triggered each rule
        </CardDescription>
      </CardHeader>
      <CardContent>
        {data.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No rule activations recorded yet. Load the seed dataset or upload a
            transaction file to populate.
          </p>
        ) : (
          <ChartContainer config={config} className="h-[360px] w-full">
            <BarChart
              data={data}
              layout="vertical"
              margin={{ left: 24, right: 16, top: 8, bottom: 8 }}
            >
              <CartesianGrid horizontal={false} />
              <XAxis type="number" tickLine={false} axisLine={false} />
              <YAxis
                type="category"
                dataKey="rule"
                tickLine={false}
                axisLine={false}
                width={140}
              />
              <ChartTooltip content={<ChartTooltipContent />} />
              <Bar dataKey="count" radius={4} fill="var(--color-count)" />
            </BarChart>
          </ChartContainer>
        )}
      </CardContent>
    </Card>
  )
}
