import { useCallback, useEffect, useMemo, useRef, useState } from "react"
import { RefreshCw, Trash2 } from "lucide-react"

import { Button } from "@/components/ui/button"
import { Separator } from "@/components/ui/separator"
import { fetchStatistics, fetchTransactions, resetAll } from "@/features/scoring/api"
import type { Statistics, StoredDecision } from "@/features/scoring/types"
import { useBulkUpload } from "@/features/scoring/useBulkUpload"
import { useNotificationSocket } from "@/lib/useWebSocket"

import {
  BulkUploadControls,
  BulkUploadStatus,
} from "./components/BulkUpload"
import { GenerateDatasetDialog } from "./components/GenerateDatasetDialog"
import { NotificationsPanel } from "./components/NotificationsPanel"
import { RiskDistribution } from "./components/RiskDistribution"
import { RuleActivations } from "./components/RuleActivations"
import { StatsCards } from "./components/StatsCards"
import { TransactionTable } from "./components/TransactionTable"

const REFRESH_INTERVAL_MS = 5000
const TXN_LIMIT = 25

export function DashboardPage() {
  const { status, notifications, lastEvent, clear } =
    useNotificationSocket("/ws")
  const [stats, setStats] = useState<Statistics | null>(null)
  const [decisions, setDecisions] = useState<StoredDecision[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [resetBusy, setResetBusy] = useState(false)
  const [notice, setNotice] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const inFlight = useRef(false)

  const refresh = useCallback(async () => {
    if (inFlight.current) return
    inFlight.current = true
    try {
      const [s, d] = await Promise.all([
        fetchStatistics(60),
        fetchTransactions(TXN_LIMIT),
      ])
      setStats(s)
      setDecisions(d)
      setError(null)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      inFlight.current = false
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    void refresh()
    const id = window.setInterval(refresh, REFRESH_INTERVAL_MS)
    return () => window.clearInterval(id)
  }, [refresh])

  useEffect(() => {
    if (!lastEvent) return
    void refresh()
  }, [lastEvent, refresh])

  const bulk = useBulkUpload(refresh)

  const handleGenerated = useCallback(async () => {
    setNotice("Dataset generated. Stats and table updated.")
    await refresh()
  }, [refresh])

  async function handleReset() {
    if (!window.confirm("Reset all decisions and history? This cannot be undone.")) return
    setResetBusy(true)
    setError(null)
    try {
      await resetAll()
      bulk.reset()
      setNotice("Engine state cleared.")
      await refresh()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setResetBusy(false)
    }
  }

  const statusBadge = useMemo(() => {
    const color =
      status === "open"
        ? "bg-emerald-500"
        : status === "connecting"
          ? "bg-amber-500"
          : "bg-rose-500"
    return (
      <span className="inline-flex items-center gap-2 text-xs text-muted-foreground">
        <span className={`size-2 rounded-full ${color}`} /> ws: {status}
      </span>
    )
  }, [status])

  return (
    <div className="min-h-svh bg-background text-foreground">
      <header className="border-b">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center gap-4 px-6 py-4">
          <div className="mr-auto">
            <h1 className="text-lg font-semibold">yuno · fraud control</h1>
            <p className="text-xs text-muted-foreground">
              Real-time transaction scoring · live data from /api
            </p>
          </div>

          <BulkUploadControls
            inputRef={bulk.inputRef}
            fileName={bulk.fileName}
            parsed={bulk.parsed}
            isRunning={bulk.isRunning}
            run={bulk.run}
            onFile={bulk.handleFile}
            onExecute={bulk.execute}
            onReset={bulk.reset}
          />

          <Separator orientation="vertical" className="h-6" />

          <div className="flex flex-wrap items-center gap-3">
            {statusBadge}
            <Button
              variant="outline"
              size="sm"
              onClick={refresh}
              data-testid="refresh"
            >
              <RefreshCw className="size-3" /> Refresh
            </Button>
            <GenerateDatasetDialog onGenerated={handleGenerated} />
            <Button
              size="sm"
              variant="outline"
              onClick={handleReset}
              disabled={resetBusy}
              data-testid="reset-all"
            >
              <Trash2 className="size-3" />
              {resetBusy ? "Resetting…" : "Reset"}
            </Button>
          </div>
        </div>
        {(bulk.isActive || notice || error) && (
          <div className="mx-auto max-w-7xl space-y-2 px-6 pb-3 text-xs">
            <BulkUploadStatus run={bulk.run} parseError={bulk.parseError} />
            {notice && (
              <p className="text-muted-foreground" data-testid="notice">
                {notice}
              </p>
            )}
            {error && (
              <p className="text-rose-500" data-testid="api-error">
                API error: {error}
              </p>
            )}
          </div>
        )}
      </header>

      <main className="mx-auto max-w-7xl space-y-6 px-6 py-6">
        <StatsCards stats={stats} />

        <section className="grid gap-6 lg:grid-cols-3">
          <RuleActivations stats={stats} />
          <RiskDistribution stats={stats} />
        </section>

        <section className="grid gap-6 lg:grid-cols-3">
          <TransactionTable decisions={decisions} isLoading={isLoading} />
          <NotificationsPanel
            status={status}
            notifications={notifications}
            onClear={clear}
          />
        </section>
      </main>
    </div>
  )
}

export default DashboardPage
