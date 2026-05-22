# yuno-challenge — Real-Time Transaction Scoring Engine

Backend service that scores card transactions for fraud risk in real time and returns an
`APPROVE` / `REVIEW` / `BLOCK` recommendation with sub-millisecond latency. Built for the
SariMart "Manila Fraud Ring" challenge.

Single Go binary serves the API and the SPA. Deployable on Railway from one Dockerfile.

## Quick start

```bash
cd backend
go test ./...      # 10 rules + 6 engine/store tests
go run .           # listens on :8080
```

```bash
# Score one transaction
curl -s -X POST http://localhost:8080/api/transactions \
  -H "Content-Type: application/json" \
  -d '{ "id":"demo", "card_id":"c1", "customer_email":"alice@gmail.com",
        "amount":250000, "currency":"PHP",
        "billing_country":"PH", "shipping_country":"PH", "merchant_country":"PH",
        "lat":14.6, "lng":120.98, "timestamp":"2026-05-21T14:00:00Z" }' | jq
```

Generate 200 synthetic transactions across a 70/20/10 risk mix and score them all:

```bash
curl -s -X POST :8080/api/transactions/generate \
  -H "Content-Type: application/json" \
  -d '{"count":200,"bias":"mixed"}' | jq
```

## Endpoints

| Method + path                       | Purpose                                                              |
| ----------------------------------- | -------------------------------------------------------------------- |
| `POST /api/transactions`            | Score one transaction, persist, return decision + reason codes      |
| `GET  /api/transactions`            | List recent decisions. `?limit=50&risk_level=high`                  |
| `GET  /api/transactions/{id}`       | Fetch a single decision by transaction id                            |
| `POST /api/transactions/generate`   | `{count, bias}` → generate + score + persist a synthetic dataset    |
| `POST /api/datasets`                | `{count, bias}` → generate only (returns the JSON, dry-run)         |
| `POST /api/reset`                   | Clear all decisions and history (demo reset)                         |
| `GET  /api/statistics`              | Aggregates over a rolling window. `?window_minutes=60`              |
| `GET  /api/config`                  | Inspect active weights, thresholds, BIN / temp-email / currency lists |
| `GET  /api/health`                  | Liveness probe                                                       |
| `GET  /ws`                          | WebSocket stream of `transaction.scored` events                      |

## Scoring logic

The engine evaluates 10 pure-function rules against the transaction and its history. Each
rule returns a score on **0–100** plus a human-readable description. The final decision is
built in four steps:

1. **Per-rule score** — each rule fires independently and returns its own 0–100 number, or 0
   if it doesn't apply (e.g. velocity returns 0 with no prior history).
2. **Weighted average over rules that fired** — see formula below.
3. **`score` → `risk_level`** via configurable thresholds.
4. **`risk_level` → `recommendation`** (APPROVE / REVIEW / BLOCK).

### Weights — what each rule catches and why this weight

| Code                     | Weight | What it catches                                                          | Why this weight |
| ------------------------ | -----: | ------------------------------------------------------------------------ | --------------- |
| `VELOCITY`               |  0.18  | 3+ tx in 30m sharing card / email / device (also 2/5/10m windows)        | Highest — SariMart's fraud team listed it first; cluster behavior is the strongest single tell. |
| `IMPOSSIBLE_TRAVEL`      |  0.15  | Implied km/h between consecutive tx on the same card exceeds plausibility | Hard to fake without coordinated fraud rings; nearly false-positive-free. |
| `AMOUNT_ANOMALY`         |  0.12  | High-value or micro (card-testing) amounts; round-number bonus           | Common pattern but also a noisy one — moderate weight. |
| `MANILA_PATTERN`         |  0.10  | Manila-area tx with cross-border / velocity / device-change signal        | Composite signal; SariMart's main market. Only fires with another trigger so noise is low. |
| `DEVICE_ANOMALY`         |  0.10  | Distinct card count on same device fingerprint in last 1h                | Single device + many cards = ring behavior. |
| `NEW_ACCOUNT_HIGH_VALUE` |  0.10  | Account younger than 24h placing a high-value order                      | Exactly the pattern SariMart documented. |
| `BIN_REPUTATION`         |  0.08  | Card BIN on the high-risk prepaid list                                   | High precision, low recall — important when it fires but not always available. |
| `NIGHT_OWL`              |  0.07  | 01–05 in the *cardholder's local time* (PH/ID/VN/...)                    | Time-of-day is a real but soft signal; deliberately lower weight than behavioural ones. |
| `CURRENCY_MISMATCH`      |  0.05  | Billing / shipping / IP diverge, or currency ≠ merchant-country currency | Lower weight because legitimate travellers can trip this. |
| `MERCHANT_SWITCHING`     |  0.05  | 3+ distinct merchants on same card in last 10m                            | Last-mile signal; only fires deep into card-testing runs. |

### Formula — weighted average over fired rules

```
final_score = Σ(rule.score × rule.weight)  / Σ(rule.weight)
              for every rule where rule.score > 0
```

**Why an average and not a sum.** Summing penalises transactions where many rules don't
apply (e.g. a first-time customer with no history can't trigger velocity, device, or
travel rules) and over-rewards transactions where every rule happens to be evaluable.
Averaging over the rules that *did* fire keeps the score in the 0–100 range and stays
fair across transactions with different signal availability.

**Implication.** A transaction with a single rule firing at 100 produces `final_score = 100`.
This is intentional: if one rule is highly confident (e.g. impossible travel at 980 km/h), we
shouldn't dilute it just because other rules had no data.

### `score` vs `risk_level` vs `recommendation`

The response surfaces three related but distinct fields. They exist as separate concepts so
the orchestrator that consumes the API has the right level of abstraction for each decision:

| Field            | Type           | What it is                                                                 |
| ---------------- | -------------- | -------------------------------------------------------------------------- |
| `score`          | float 0–100    | Raw weighted-average output. Use for ranking, analytics, A/B testing.      |
| `risk_level`     | enum           | Bucket derived from `score` via thresholds. Use for dashboards & alerts.   |
| `recommendation` | enum           | Operational action. **This is the only field the payment orchestrator should branch on.** |

Default thresholds (configurable via env vars — see Configuration):

| Score    | `risk_level` | `recommendation` |
| -------- | ------------ | ---------------- |
| 0–29     | low          | APPROVE          |
| 30–59    | medium       | REVIEW           |
| 60–79    | high         | BLOCK            |
| 80–100   | critical     | BLOCK            |

## Test scenarios — reproducible curl

These four flows cover all the "What we're looking for" acceptance criteria. Run them
against `localhost:8080` after `go run .` to verify end-to-end.

### 1) Clean transaction → APPROVE

```bash
curl -s -X POST :8080/api/transactions -H "Content-Type: application/json" -d '{
  "id":"clean-1","card_id":"card-a","customer_email":"alice@gmail.com",
  "account_created_at":"2024-01-01T00:00:00Z",
  "billing_country":"PH","shipping_country":"PH","merchant_country":"PH","ip_country":"PH",
  "amount":250000,"currency":"PHP",
  "lat":14.6,"lng":120.98,
  "timestamp":"2026-05-21T14:00:00Z"
}' | jq '{score, risk_level, recommendation, n_rules: (.triggered_rules|length)}'
```

Expected: `score: 0`, `risk_level: "low"`, `recommendation: "APPROVE"`, `n_rules: 0`.

### 2) Velocity detection → REVIEW or BLOCK after 3 POSTs

```bash
for i in 1 2 3; do
  curl -s -X POST :8080/api/transactions -H "Content-Type: application/json" -d "{
    \"id\":\"v-$i\",\"card_id\":\"card-velo\",\"customer_email\":\"velo@ex.com\",
    \"device_fingerprint\":\"dev-velo\",
    \"merchant_country\":\"PH\",
    \"amount\":150000,\"currency\":\"PHP\",
    \"timestamp\":\"2026-05-21T14:0$i:00Z\"
  }" | jq '{id:.transaction_id, score, recommendation, rules:[.triggered_rules[].code]}';
done
```

Expected: the third request's response includes `"VELOCITY"` in `rules`, and the score climbs visibly.

### 3) Impossible travel → BLOCK

Two POSTs on the same card, NYC → Manila 30 minutes later (≈ 13,500 km in 0.5 h):

```bash
curl -s -X POST :8080/api/transactions -d '{
  "id":"trv-1","card_id":"card-trv","amount":300000,"currency":"USD",
  "lat":40.7128,"lng":-74.0060,"timestamp":"2026-05-21T14:00:00Z",
  "merchant_country":"US"
}' -H "Content-Type: application/json" | jq '{score, recommendation}'

curl -s -X POST :8080/api/transactions -d '{
  "id":"trv-2","card_id":"card-trv","amount":400000,"currency":"PHP",
  "lat":14.6,"lng":120.98,"timestamp":"2026-05-21T14:30:00Z",
  "merchant_country":"PH"
}' -H "Content-Type: application/json" | jq '{score, recommendation, rules:[.triggered_rules[].code]}'
```

Expected: second response has `"IMPOSSIBLE_TRAVEL"` in `rules`, with `recommendation: "BLOCK"`.

### 4) Perfect storm → critical / BLOCK with 7+ rules

Seed 5 rapid-fire tx (so velocity fires), then the actual storm payload:

```bash
curl -s -X POST :8080/api/reset > /dev/null

for i in 1 2 3 4 5; do
  curl -s -X POST :8080/api/transactions -d "{
    \"id\":\"prep-$i\",\"card_id\":\"card-storm\",
    \"customer_email\":\"bad@tempmail.com\",\"device_fingerprint\":\"dev-storm\",
    \"merchant_id\":\"m-$i\",\"merchant_country\":\"PH\",
    \"amount\":300000,\"currency\":\"PHP\",
    \"timestamp\":\"2026-05-21T18:5${i}:00Z\"
  }" -H "Content-Type: application/json" > /dev/null;
done

curl -s -X POST :8080/api/transactions -d '{
  "id":"storm-final","card_id":"card-storm","card_bin":"411111",
  "customer_email":"bad@tempmail.com","device_fingerprint":"dev-storm",
  "account_created_at":"2026-05-21T17:00:00Z",
  "billing_country":"US","shipping_country":"VN","ip_country":"RU","merchant_country":"PH",
  "merchant_id":"m-storm","amount":2000000,"currency":"USD",
  "lat":14.6,"lng":120.98,"previous_declines":3,
  "timestamp":"2026-05-21T19:00:00Z"
}' -H "Content-Type: application/json" | jq '{score, risk_level, recommendation, n_rules: (.triggered_rules|length), rules: [.triggered_rules[].code]}'
```

Expected (asserted by `TestEngineCriticalStorm`):

- `score ≥ 80`
- `risk_level: "critical"`
- `recommendation: "BLOCK"`
- `n_rules ≥ 7` including `VELOCITY`, `AMOUNT_ANOMALY`, `MANILA_PATTERN`,
  `BIN_REPUTATION`, `CURRENCY_MISMATCH`, `NIGHT_OWL`, `NEW_ACCOUNT_HIGH_VALUE`,
  `DEVICE_ANOMALY`, `MERCHANT_SWITCHING`.

> **Note on NightOwl.** The storm uses timestamp `19:00 UTC`, which is `03:00 Manila local`
> (UTC+8). The rule resolves the local hour from `merchant_country` so it fires correctly.
> A transaction at `03:00 UTC` from PH (= `11:00 Manila`) will *not* fire NightOwl — verified
> by `TestNightOwlLocalTime`.

## Generating synthetic datasets

Three risk profiles, exposed via the `bias` field in `/api/transactions/generate` and
`/api/datasets`:

| `bias`   | What you get                                                                  |
| -------- | ----------------------------------------------------------------------------- |
| `mixed`  | ~70% low / 20% medium / 10% high. The reference demo dataset.                 |
| `medium` | Only medium-risk tx (single suspicious signal each). Useful for REVIEW path.  |
| `high`   | Repeated high-risk clusters: velocity burst, card testing, storm, impossible travel, device share. Mostly BLOCK. |

```bash
# Score 100 mixed-bias tx — expect ~70 APPROVE / 15 REVIEW / 15 BLOCK
curl -X POST :8080/api/transactions/generate -d '{"count":100,"bias":"mixed"}'

# Score 51 high-bias tx (3 cluster repeats) — expect ~all BLOCK
curl -X POST :8080/api/transactions/generate -d '{"count":51,"bias":"high"}'

# Just download a 200-tx JSON file (no scoring) for use with the Bulk Upload UI
curl -X POST :8080/api/datasets -d '{"count":200,"bias":"mixed"}' > demo.json
```

From the dashboard: **Generate dataset** button opens a dialog with a risk profile selector
plus a count input. Two actions: **Download JSON** (just the file) or **Generate & run**
(POST + score).

## Configuration

All weights, thresholds, and lookup tables ship in `backend/config/scoring.json` (embedded
via `//go:embed`) so the binary boots with zero external config. Two override paths:

1. `SCORING_CONFIG_PATH` — path to a JSON file on disk that replaces the embedded defaults.
2. Per-value env overrides:
   - `SCORING_THRESHOLD_MEDIUM`, `SCORING_THRESHOLD_HIGH`, `SCORING_THRESHOLD_CRITICAL`
   - `SCORING_WEIGHT_<CODE>` (e.g. `SCORING_WEIGHT_VELOCITY=0.25`)

`GET /api/config` returns the active config so you can verify overrides took effect.

## Architecture

Pragmatic separation, no over-engineering. Domain code never imports `net/http`.

```
backend/
├── main.go                       # wiring + HTTP mux + WS hub
├── config/                       # ConfigRepository adapters
│   ├── file_repository.go        # embedded JSON + env overrides
│   └── scoring.json
├── handlers/                     # HTTP layer (parse → engine → marshal)
│   ├── transactions.go statistics.go generate.go datasets.go reset.go
│   ├── bias.go                   # shared bias parser
│   └── http.go                   # writeJSON / writeError
├── datagen/                      # synthetic data generator (library)
│   └── generator.go              # Generate(opts) — mixed / medium / high bias
├── seedgen/                      # CLI wrapper around datagen
│   └── main.go                   # `go run ./seedgen -n 50 -bias high > out.json`
└── scoring/                      # domain
    ├── types.go                  # Transaction, ScoreResult, Rule, ReasonCode
    ├── config.go                 # Config struct + ConfigRepository interface
    ├── store.go                  # MemoryStore + Reader/Writer interfaces
    ├── engine.go                 # weighted scoring, risk buckets, recommendation
    └── rules/                    # one rule per file, all pure functions
        ├── build.go              # rules.Build(cfg) → []scoring.Rule
        ├── helpers.go            # haversine, countAny, country→timezone, ...
        ├── amount.go bin.go currency.go device.go impossible_travel.go
        ├── manila.go merchant.go new_account.go night_owl.go velocity.go
        └── rules_test.go         # pass / borderline / blowout + engine storm test
```

### `datagen` vs `seedgen` — what's the difference?

- **`datagen/`** is a Go *library*. It exposes `Generate(opts Options) []Transaction` and
  is called by the `/api/transactions/generate` and `/api/datasets` endpoints, by the
  scoring tests, and by the `seedgen` CLI. All generation logic lives here.
- **`seedgen/`** is a *binary* CLI — a thin wrapper that calls `datagen.Generate(...)` and
  prints JSON to stdout. Useful for producing files outside the running server (CI
  fixtures, sharable demo data, scripted bulk uploads).

### Design principles

- **Rules as pure functions**: no I/O, no `time.Now()` inside. Determinism is enforced by
  passing `now` and the `HistoryReader` interface in. Trivially unit-testable.
- **Engine has no rule knowledge**: it receives `[]scoring.Rule` from the caller. Adding,
  removing, or A/B-testing rules is a wiring-only change in `rules.Build`.
- **Store behind interfaces**: `HistoryReader` / `DecisionWriter` / `DecisionReader` /
  `Resetter`. The in-memory implementation swaps for Redis / Postgres without touching rules.
- **Latency vs determinism separated**: rule outputs are deterministic from
  `(txn, history, now)`; latency and `processed_at` use wall-clock so dashboards behave
  correctly even when scoring historical data.

## Operational policy: fail-open vs fail-closed

| Component               | Policy       | Why                                                              |
| ----------------------- | ------------ | ---------------------------------------------------------------- |
| Request validation      | Fail-closed  | Garbage input → 400, never silently score                        |
| Engine itself           | Fail-safe    | Returns `REVIEW` on internal failure — never default-`APPROVE`   |
| History store reads     | Fail-open    | Missing history yields 0 for that signal; other rules still fire |
| Audit / decision write  | Fail-open    | Persistence failure must not block a payment-path decision       |
| Notification broadcast  | Fail-open    | WS hub errors do not affect scoring                              |

## Testing

```bash
cd backend
go test ./... -v
```

Coverage:

- **Per rule** (`scoring/rules/rules_test.go`): pass / borderline / blowout cases per rule.
- **NightOwl timezone** (`TestNightOwlLocalTime`): verifies country→TZ resolution.
- **Critical storm** (`TestEngineCriticalStorm`): pre-seeds 5 history records and asserts
  the storm payload reaches `score ≥ 80`, `risk_level: critical`, `recommendation: BLOCK`
  with `≥ 7` rules firing. This is the automated equivalent of test scenario #4.
- **Store reset** (`TestStoreResetClearsStateAndStats`): score → stats > 0 → reset → stats == 0.
- **Engine edge cases** (`TestEngineCleanTxApproved`, `TestEngineNoRulesFireIsZero`,
  `TestEngineScoreCapped`).

## Frontend dev

The Go binary serves the Vite SPA from `./web` (or `STATIC_DIR`). For hot-reload:

```bash
cd frontend
pnpm install
pnpm dev   # proxies /api → :8080
```

The dashboard shows live statistics, a recent decisions table, a notifications panel that
streams `transaction.scored` events over WebSocket, and the **Generate dataset** dialog
with the three risk profiles.

## Deploy

Railway auto-detects the multi-stage `Dockerfile`. Healthcheck path is `/api/health`. The
only required env var is `PORT` (Railway injects it). Optional: `SCORING_CONFIG_PATH`,
`SCORING_THRESHOLD_*`, `SCORING_WEIGHT_*`.

## Production evolution path

The repo's three top-level design notes (`01-ARQUITECTURA-SCORING-ENGINE.md`,
`02-GUIA-IMPLEMENTACION-PASO-A-PASO.md`, `03-PATRONES-Y-DECISIONES.md`) describe how to
extend this prototype: strict hexagonal layout, expr-lang rule DSL with hot-reload from
Postgres, Redis-backed velocity store, atomic-pointer rule reload, blocklist + shadow
rules, and ML scoring as a sidecar. None of that is required for the challenge; the seams
here (`HistoryReader` / `ConfigRepository` interfaces, weights via config, rules in a
subpackage) make those swaps contained refactors rather than rewrites.
