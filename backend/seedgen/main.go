// seedgen prints a JSON array of synthetic transactions to stdout. The actual
// generation logic lives in the datagen package; this binary is a thin CLI so
// the same dataset can be produced for tests, demos, and bulk uploads.
//
// Usage:
//
//	go run ./seedgen                       # default 183 mixed transactions
//	go run ./seedgen -n 50                 # 50 mixed transactions
//	go run ./seedgen -n 34 -bias high      # two repeats of the high-risk cluster
//	go run ./seedgen -bias medium -n 30    # 30 medium-risk only
//	go run ./seedgen > demo.json
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/ccast2/yuno-challenge/backend/datagen"
)

func main() {
	count := flag.Int("n", 183, "number of transactions to generate")
	bias := flag.String("bias", "mixed", "risk bias: mixed | medium | high")
	flag.Parse()

	b, err := parseBias(*bias)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}

	txns := datagen.Generate(datagen.Options{Total: *count, Bias: b})
	out, err := json.MarshalIndent(txns, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Stdout.Write(out)
	os.Stdout.WriteString("\n")
}

func parseBias(raw string) (datagen.RiskBias, error) {
	switch raw {
	case "", string(datagen.BiasMixed):
		return datagen.BiasMixed, nil
	case string(datagen.BiasMedium):
		return datagen.BiasMedium, nil
	case string(datagen.BiasHigh):
		return datagen.BiasHigh, nil
	default:
		return "", fmt.Errorf("invalid -bias %q (use mixed|medium|high)", raw)
	}
}
