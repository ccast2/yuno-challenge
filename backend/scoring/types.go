package scoring

import "time"

type Transaction struct {
	ID                 string    `json:"id"`
	CustomerID         string    `json:"customer_id"`
	CustomerEmail      string    `json:"customer_email"`
	AccountCreatedAt   time.Time `json:"account_created_at"`
	CardID             string    `json:"card_id"`
	CardBIN            string    `json:"card_bin"`
	CardLast4          string    `json:"card_last4"`
	DeviceFingerprint  string    `json:"device_fingerprint"`
	IP                 string    `json:"ip"`
	IPCountry          string    `json:"ip_country"`
	BillingCountry     string    `json:"billing_country"`
	ShippingCountry    string    `json:"shipping_country"`
	MerchantID         string    `json:"merchant_id"`
	MerchantCountry    string    `json:"merchant_country"`
	Amount             int64     `json:"amount"`
	Currency           string    `json:"currency"`
	Lat                float64   `json:"lat"`
	Lng                float64   `json:"lng"`
	PreviousDeclines   int       `json:"previous_declines"`
	Timestamp          time.Time `json:"timestamp"`
}

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

type Recommendation string

const (
	Approve Recommendation = "APPROVE"
	Review  Recommendation = "REVIEW"
	Block   Recommendation = "BLOCK"
)

type ReasonCode struct {
	Code        string  `json:"code"`
	Description string  `json:"description"`
	Score       float64 `json:"score"`
	Weight      float64 `json:"weight"`
}

type ScoreResult struct {
	TransactionID  string         `json:"transaction_id"`
	Score          float64        `json:"score"`
	RiskLevel      RiskLevel      `json:"risk_level"`
	Recommendation Recommendation `json:"recommendation"`
	TriggeredRules []ReasonCode   `json:"triggered_rules"`
	ProcessedAt    time.Time      `json:"processed_at"`
	LatencyMs      int64          `json:"latency_ms"`
}

type RuleFn func(txn Transaction, h HistoryReader, now time.Time) (score float64, description string)

type Rule struct {
	Code        string
	Name        string
	Weight      float64
	Fn          RuleFn
}
