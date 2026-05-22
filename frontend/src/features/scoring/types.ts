export type RiskLevel = "low" | "medium" | "high" | "critical"
export type Recommendation = "APPROVE" | "REVIEW" | "BLOCK"
export type RiskBias = "mixed" | "medium" | "high"

export type Transaction = {
  id: string
  customer_id: string
  customer_email: string
  account_created_at: string
  card_id: string
  card_bin: string
  card_last4?: string
  device_fingerprint: string
  ip: string
  ip_country: string
  billing_country: string
  shipping_country: string
  merchant_id: string
  merchant_country: string
  amount: number
  currency: string
  lat: number
  lng: number
  previous_declines?: number
  timestamp: string
}

export type ReasonCode = {
  code: string
  description: string
  score: number
  weight: number
}

export type ScoreResult = {
  transaction_id: string
  score: number
  risk_level: RiskLevel
  recommendation: Recommendation
  triggered_rules: ReasonCode[]
  processed_at: string
  latency_ms: number
}

export type StoredDecision = {
  transaction: Transaction
  result: ScoreResult
}

export type RuleFrequency = {
  code: string
  count: number
}

export type Statistics = {
  total_scored: number
  by_recommendation: Partial<Record<Recommendation, number>>
  by_risk_level: Partial<Record<RiskLevel, number>>
  average_score: number
  average_latency_ms: number
  top_rules: RuleFrequency[]
  window_since: string
}
