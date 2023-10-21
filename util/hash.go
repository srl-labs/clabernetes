package util

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashBytes accepts a bytes object and returns a string sha256 hash representing that object.
func HashBytes(b []byte) string {
	hash := sha256.New()
	hash.Write(b)

	return hex.EncodeToString(hash.Sum(nil))
}
