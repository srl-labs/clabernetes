package util

import (
	"encoding/json"

	"github.com/carlmontanari/difflibgo/difflibgo"
)

// UnifiedDiff accepts a and b as any, dumps them to json then returns a unified diff from
// difflibgo.
func UnifiedDiff(a, b any) (string, error) {
	var aStr string

	var bStr string

	switch aa := a.(type) {
	case string:
		aStr = aa
	case []byte:
		aStr = string(aa)
	default:
		aBytes, err := json.MarshalIndent(a, "", "    ")
		if err != nil {
			return "", err
		}

		aStr = string(aBytes)
	}

	switch bb := b.(type) {
	case string:
		bStr = bb
	case []byte:
		bStr = string(bb)
	default:
		bBytes, err := json.MarshalIndent(b, "", "    ")
		if err != nil {
			return "", err
		}

		bStr = string(bBytes)
	}

	return difflibgo.UnifiedDiff(aStr, bStr), nil
}
