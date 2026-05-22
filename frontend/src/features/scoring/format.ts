import type { Recommendation, RiskLevel } from "./types"

export function formatMoney(minorUnits: number, currency: string): string {
  const major = minorUnits / 100
  try {
    return new Intl.NumberFormat("en-US", {
      style: "currency",
      currency,
      maximumFractionDigits: 2,
    }).format(major)
  } catch {
    return `${major.toFixed(2)} ${currency}`
  }
}

export function riskBadgeVariant(
  risk: RiskLevel,
): "default" | "secondary" | "destructive" | "outline" {
  switch (risk) {
    case "critical":
    case "high":
      return "destructive"
    case "medium":
      return "secondary"
    default:
      return "outline"
  }
}

export function recommendationVariant(
  rec: Recommendation,
): "default" | "secondary" | "destructive" | "outline" {
  switch (rec) {
    case "BLOCK":
      return "destructive"
    case "REVIEW":
      return "secondary"
    case "APPROVE":
      return "default"
    default:
      return "outline"
  }
}

export function severityVariant(
  severity: string,
): "default" | "secondary" | "destructive" | "outline" {
  switch (severity) {
    case "error":
    case "critical":
      return "destructive"
    case "warn":
    case "warning":
      return "secondary"
    case "success":
      return "default"
    default:
      return "outline"
  }
}

export function ruleLabel(code: string): string {
  return code
    .replace(/_/g, " ")
    .toLowerCase()
    .replace(/\b\w/g, (c) => c.toUpperCase())
}
