import { useCallback, useRef, useState } from "react"

import { scoreTransaction } from "./api"
import type { Recommendation, Transaction } from "./types"

export type BulkRunState = {
  total: number
  processed: number
  byRecommendation: Record<Recommendation, number>
  errors: number
  lastError?: string
}

const initialRun: BulkRunState = {
  total: 0,
  processed: 0,
  byRecommendation: { APPROVE: 0, REVIEW: 0, BLOCK: 0 },
  errors: 0,
}

export function useBulkUpload(onComplete?: () => void | Promise<void>) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [fileName, setFileName] = useState<string | null>(null)
  const [parsed, setParsed] = useState<Transaction[] | null>(null)
  const [parseError, setParseError] = useState<string | null>(null)
  const [run, setRun] = useState<BulkRunState>(initialRun)
  const [isRunning, setIsRunning] = useState(false)

  const handleFile = useCallback(async (file: File) => {
    setFileName(file.name)
    setParseError(null)
    setRun(initialRun)
    try {
      const text = await file.text()
      const data = JSON.parse(text)
      if (!Array.isArray(data)) {
        throw new Error("file must contain a JSON array of transactions")
      }
      setParsed(data as Transaction[])
    } catch (err) {
      setParsed(null)
      setParseError(err instanceof Error ? err.message : String(err))
    }
  }, [])

  const execute = useCallback(async () => {
    if (!parsed || parsed.length === 0) return
    setIsRunning(true)
    let state: BulkRunState = {
      ...initialRun,
      total: parsed.length,
      byRecommendation: { APPROVE: 0, REVIEW: 0, BLOCK: 0 },
    }
    setRun(state)
    for (const txn of parsed) {
      try {
        const result = await scoreTransaction(txn)
        state = {
          ...state,
          processed: state.processed + 1,
          byRecommendation: {
            ...state.byRecommendation,
            [result.recommendation]:
              state.byRecommendation[result.recommendation] + 1,
          },
        }
      } catch (err) {
        state = {
          ...state,
          processed: state.processed + 1,
          errors: state.errors + 1,
          lastError: err instanceof Error ? err.message : String(err),
        }
      }
      setRun({ ...state })
    }
    setIsRunning(false)
    await onComplete?.()
  }, [parsed, onComplete])

  const reset = useCallback(() => {
    setParsed(null)
    setFileName(null)
    setParseError(null)
    setRun(initialRun)
    if (inputRef.current) inputRef.current.value = ""
  }, [])

  const isActive = run.total > 0 || parseError !== null

  return {
    inputRef,
    fileName,
    parsed,
    parseError,
    run,
    isRunning,
    isActive,
    handleFile,
    execute,
    reset,
  }
}
