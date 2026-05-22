import { Badge } from "@/components/ui/badge"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  formatMoney,
  recommendationVariant,
  riskBadgeVariant,
} from "@/features/scoring/format"
import type { StoredDecision } from "@/features/scoring/types"

type Props = {
  decisions: StoredDecision[]
  isLoading: boolean
}

export function TransactionTable({ decisions, isLoading }: Props) {
  return (
    <Card className="lg:col-span-2">
      <CardHeader>
        <CardTitle>Recent decisions</CardTitle>
        <CardDescription>
          {decisions.length > 0
            ? `Last ${decisions.length} scored transactions (newest first)`
            : "Nothing scored yet"}
        </CardDescription>
      </CardHeader>
      <CardContent>
        {decisions.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {isLoading
              ? "Loading…"
              : "Load the seed dataset or upload a transaction file to see decisions here."}
          </p>
        ) : (
          <div className="max-h-[480px] overflow-y-auto rounded-md border">
            <table className="w-full text-sm">
              <thead className="sticky top-0 z-10 bg-muted/80 text-xs uppercase text-muted-foreground backdrop-blur">
                <tr>
                  <th className="px-3 py-2 text-left">Txn</th>
                  <th className="px-3 py-2 text-left">Card</th>
                  <th className="px-3 py-2 text-right">Amount</th>
                  <th className="px-3 py-2 text-left">Merchant</th>
                  <th className="px-3 py-2 text-left">Country</th>
                  <th className="px-3 py-2 text-right">Score</th>
                  <th className="px-3 py-2 text-left">Risk</th>
                  <th className="px-3 py-2 text-left">Decision</th>
                  <th className="px-3 py-2 text-left">Top rules</th>
                </tr>
              </thead>
              <tbody>
                {decisions.map((d) => (
                  <tr key={d.transaction.id} className="border-t align-top">
                    <td className="px-3 py-2 font-mono text-xs">
                      {d.transaction.id}
                    </td>
                    <td className="px-3 py-2 font-mono text-xs">
                      {d.transaction.card_id}
                    </td>
                    <td className="px-3 py-2 text-right">
                      {formatMoney(
                        d.transaction.amount,
                        d.transaction.currency,
                      )}
                    </td>
                    <td className="px-3 py-2">{d.transaction.merchant_id}</td>
                    <td className="px-3 py-2">
                      {d.transaction.merchant_country}
                    </td>
                    <td className="px-3 py-2 text-right font-semibold">
                      {d.result.score.toFixed(1)}
                    </td>
                    <td className="px-3 py-2">
                      <Badge variant={riskBadgeVariant(d.result.risk_level)}>
                        {d.result.risk_level}
                      </Badge>
                    </td>
                    <td className="px-3 py-2">
                      <Badge
                        variant={recommendationVariant(d.result.recommendation)}
                      >
                        {d.result.recommendation}
                      </Badge>
                    </td>
                    <td className="px-3 py-2 text-xs text-muted-foreground">
                      {d.result.triggered_rules
                        .slice(0, 3)
                        .map((r) => r.code)
                        .join(", ") || "—"}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
