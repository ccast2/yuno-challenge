import { useState } from "react"
import { Download, Play, Sparkles } from "lucide-react"

import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { generateDataset, previewDataset } from "@/features/scoring/api"
import type { RiskBias } from "@/features/scoring/types"
import { cn } from "@/lib/utils"

const PRESETS = [50, 183, 500, 1000]
const MIN_BY_BIAS: Record<RiskBias, number> = {
  mixed: 17,
  medium: 1,
  high: 17,
}
const MAX = 5000

const BIAS_OPTIONS: {
  value: RiskBias
  label: string
  hint: string
}[] = [
  {
    value: "mixed",
    label: "Mixed (70/20/10)",
    hint: "Default demo dataset: clean traffic plus medium signals and the full fraud-scenario cluster.",
  },
  {
    value: "medium",
    label: "Medium-risk only",
    hint: "Single-signal transactions (night-owl, new account, currency mix). Expect mostly REVIEW.",
  },
  {
    value: "high",
    label: "High-risk only",
    hint: "Repeats the fraud cluster (velocity burst, card testing, storm, impossible travel, device share). Expect mostly BLOCK.",
  },
]

type Props = {
  onGenerated: () => void | Promise<void>
}

type ActionKind = "download" | "run" | null

export function GenerateDatasetDialog({ onGenerated }: Props) {
  const [open, setOpen] = useState(false)
  const [count, setCount] = useState<number>(183)
  const [bias, setBias] = useState<RiskBias>("mixed")
  const [activeAction, setActiveAction] = useState<ActionKind>(null)
  const [summary, setSummary] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  function validate(): boolean {
    const min = MIN_BY_BIAS[bias]
    if (!Number.isFinite(count) || count < min || count > MAX) {
      setError(`count must be between ${min} and ${MAX} for ${bias}`)
      return false
    }
    return true
  }

  async function handleDownload() {
    if (!validate()) return
    setActiveAction("download")
    setError(null)
    setSummary(null)
    try {
      const txns = await previewDataset(count, bias)
      const blob = new Blob([JSON.stringify(txns, null, 2)], {
        type: "application/json",
      })
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = `transactions-${bias}-${count}.json`
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      setSummary(`Downloaded transactions-${bias}-${count}.json (${txns.length} tx)`)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setActiveAction(null)
    }
  }

  async function handleRun() {
    if (!validate()) return
    setActiveAction("run")
    setError(null)
    setSummary(null)
    try {
      const res = await generateDataset(count, bias)
      setSummary(
        `Generated ${res.generated} (${res.bias}) · ` +
          Object.entries(res.by_recommendation)
            .map(([rec, n]) => `${rec}: ${n}`)
            .join(" · "),
      )
      await onGenerated()
      setOpen(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
    } finally {
      setActiveAction(null)
    }
  }

  const busy = activeAction !== null
  const activeBias = BIAS_OPTIONS.find((b) => b.value === bias)!

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        setOpen(o)
        if (!o) {
          setError(null)
          setSummary(null)
        }
      }}
    >
      <DialogTrigger asChild>
        <Button size="sm" data-testid="open-generate-dialog">
          <Sparkles className="size-3.5" /> Generate dataset
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Generate synthetic dataset</DialogTitle>
          <DialogDescription>
            Pick a risk profile and how many transactions. You can download the
            JSON file or push them through the scoring engine right away.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="space-y-2">
            <span className="text-sm font-medium">Risk profile</span>
            <div
              className="flex flex-wrap gap-2"
              role="radiogroup"
              data-testid="bias-selector"
            >
              {BIAS_OPTIONS.map((opt) => {
                const selected = opt.value === bias
                return (
                  <button
                    key={opt.value}
                    type="button"
                    role="radio"
                    aria-checked={selected}
                    disabled={busy}
                    onClick={() => setBias(opt.value)}
                    data-testid={`bias-${opt.value}`}
                    className={cn(
                      "rounded-md border px-3 py-1.5 text-sm transition-colors disabled:opacity-50",
                      selected
                        ? "border-primary bg-primary text-primary-foreground"
                        : "border-input bg-background hover:bg-accent hover:text-accent-foreground",
                    )}
                  >
                    {opt.label}
                  </button>
                )
              })}
            </div>
            <p className="text-xs text-muted-foreground">{activeBias.hint}</p>
          </div>

          <div className="space-y-1.5">
            <label htmlFor="dataset-count" className="text-sm font-medium">
              Number of transactions
            </label>
            <Input
              id="dataset-count"
              type="number"
              min={MIN_BY_BIAS[bias]}
              max={MAX}
              value={count}
              onChange={(e) => setCount(Number(e.target.value))}
              data-testid="dataset-count-input"
              disabled={busy}
            />
            <p className="text-xs text-muted-foreground">
              Between {MIN_BY_BIAS[bias]} and {MAX}.
            </p>
          </div>

          <div className="flex flex-wrap gap-2">
            {PRESETS.map((n) => (
              <Button
                key={n}
                variant="outline"
                size="sm"
                onClick={() => setCount(n)}
                disabled={busy}
                type="button"
              >
                {n}
              </Button>
            ))}
          </div>

          {error && (
            <p className="text-sm text-rose-500" data-testid="generate-error">
              {error}
            </p>
          )}
          {summary && (
            <p
              className="text-sm text-muted-foreground"
              data-testid="generate-summary"
            >
              {summary}
            </p>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={handleDownload}
            disabled={busy}
            data-testid="download-dataset"
          >
            <Download className="size-3.5" />
            {activeAction === "download" ? "Preparing…" : "Download JSON"}
          </Button>
          <Button
            onClick={handleRun}
            disabled={busy}
            data-testid="run-dataset"
          >
            <Play className="size-3.5" />
            {activeAction === "run" ? "Running…" : `Generate & run ${count}`}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
