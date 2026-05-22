package handlers

import (
	"fmt"

	"github.com/ccast2/yuno-challenge/backend/datagen"
)

// parseBias normalizes the bias query-string / body value into a datagen
// RiskBias. Empty input means "use the default mix". Unknown values produce a
// 400-grade error so the handler can surface them to the client.
func parseBias(raw string) (datagen.RiskBias, error) {
	switch raw {
	case "":
		return datagen.BiasMixed, nil
	case string(datagen.BiasMixed),
		string(datagen.BiasMedium),
		string(datagen.BiasHigh):
		return datagen.RiskBias(raw), nil
	default:
		return "", fmt.Errorf("bias must be one of: mixed, medium, high (got %q)", raw)
	}
}
