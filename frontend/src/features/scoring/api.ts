import type {
  RiskBias,
  ScoreResult,
  Statistics,
  StoredDecision,
  Transaction,
} from "./types"

async function jsonFetch<T>(
  input: RequestInfo,
  init?: RequestInit,
): Promise<T> {
  const res = await fetch(input, {
    ...init,
    headers: {
      ...(init?.body ? { "Content-Type": "application/json" } : {}),
      ...init?.headers,
    },
  })
  if (!res.ok) {
    const body = await res.text().catch(() => "")
    throw new Error(`${res.status} ${res.statusText}${body ? ` — ${body}` : ""}`)
  }
  return (await res.json()) as T
}

export function fetchStatistics(windowMinutes = 60): Promise<Statistics> {
  return jsonFetch<Statistics>(
    `/api/statistics?window_minutes=${windowMinutes}`,
  )
}

export function fetchTransactions(
  limit = 50,
  riskLevel?: string,
): Promise<StoredDecision[]> {
  const params = new URLSearchParams({ limit: String(limit) })
  if (riskLevel) params.set("risk_level", riskLevel)
  return jsonFetch<StoredDecision[]>(`/api/transactions?${params.toString()}`)
}

export function scoreTransaction(txn: Transaction): Promise<ScoreResult> {
  return jsonFetch<ScoreResult>("/api/transactions", {
    method: "POST",
    body: JSON.stringify(txn),
  })
}

export function generateDataset(
  count: number,
  bias: RiskBias = "mixed",
): Promise<{
  generated: number
  by_recommendation: Record<string, number>
  bias: RiskBias
}> {
  return jsonFetch("/api/transactions/generate", {
    method: "POST",
    body: JSON.stringify({ count, bias }),
  })
}

export function previewDataset(
  count: number,
  bias: RiskBias = "mixed",
): Promise<Transaction[]> {
  return jsonFetch<Transaction[]>("/api/datasets", {
    method: "POST",
    body: JSON.stringify({ count, bias }),
  })
}

export function resetAll(): Promise<{ ok: boolean }> {
  return jsonFetch("/api/reset", { method: "POST" })
}
