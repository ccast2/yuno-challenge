import { Upload } from "lucide-react"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { recommendationVariant } from "@/features/scoring/format"
import type { BulkRunState } from "@/features/scoring/useBulkUpload"

type ControlsProps = {
  inputRef: React.RefObject<HTMLInputElement | null>
  fileName: string | null
  parsed: { length: number } | null
  isRunning: boolean
  run: BulkRunState
  onFile: (file: File) => void
  onExecute: () => void
  onReset: () => void
}

export function BulkUploadControls({
  inputRef,
  fileName,
  parsed,
  isRunning,
  run,
  onFile,
  onExecute,
  onReset,
}: ControlsProps) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      <label className="inline-flex items-center gap-2 rounded-md border border-input bg-background px-3 py-1.5 text-sm font-medium hover:bg-accent hover:text-accent-foreground cursor-pointer">
        <Upload className="size-3.5" />
        <span className="max-w-[160px] truncate">
          {fileName ?? "Upload transactions"}
        </span>
        <input
          ref={inputRef}
          type="file"
          accept="application/json,.json"
          className="hidden"
          data-testid="bulk-upload-input"
          onChange={(e) => {
            const f = e.target.files?.[0]
            if (f) void onFile(f)
          }}
        />
      </label>
      <Button
        size="sm"
        onClick={onExecute}
        disabled={!parsed || isRunning || parsed.length === 0}
        data-testid="bulk-upload-run"
      >
        {isRunning
          ? `Scoring ${run.processed}/${run.total}…`
          : parsed
            ? `Score ${parsed.length}`
            : "Score"}
      </Button>
      <Button
        size="sm"
        variant="ghost"
        onClick={onReset}
        disabled={isRunning || (!parsed && !fileName)}
      >
        Reset
      </Button>
    </div>
  )
}

type StatusProps = {
  run: BulkRunState
  parseError: string | null
}

export function BulkUploadStatus({ run, parseError }: StatusProps) {
  if (parseError) {
    return (
      <p className="text-xs text-rose-500" data-testid="bulk-upload-error">
        Could not parse file: {parseError}
      </p>
    )
  }
  if (run.total === 0) return null

  const progressPct = Math.round((run.processed / run.total) * 100)

  return (
    <div className="space-y-2" data-testid="bulk-upload-progress">
      <div className="flex items-center justify-between text-xs text-muted-foreground">
        <span>
          Bulk score · {run.processed}/{run.total} processed
          {run.errors > 0 ? ` · ${run.errors} errors` : ""}
        </span>
        <span>{progressPct}%</span>
      </div>
      <div className="h-1.5 w-full overflow-hidden rounded-full bg-muted">
        <div
          className="h-full bg-primary transition-all"
          style={{ width: `${progressPct}%` }}
        />
      </div>
      <div className="flex flex-wrap gap-2 pt-1">
        <Badge variant={recommendationVariant("APPROVE")}>
          APPROVE · {run.byRecommendation.APPROVE}
        </Badge>
        <Badge variant={recommendationVariant("REVIEW")}>
          REVIEW · {run.byRecommendation.REVIEW}
        </Badge>
        <Badge variant={recommendationVariant("BLOCK")}>
          BLOCK · {run.byRecommendation.BLOCK}
        </Badge>
        {run.errors > 0 && (
          <Badge variant="destructive">Errors · {run.errors}</Badge>
        )}
      </div>
      {run.lastError && (
        <p className="text-xs text-rose-500">Last error: {run.lastError}</p>
      )}
    </div>
  )
}
