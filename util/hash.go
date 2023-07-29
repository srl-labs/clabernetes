package util

import (
	"crypto/sha256"
	"fmt"
)

// HashBytes accepts a bytes object and returns a string sha256 hash representing that object.
func HashBytes(b []byte) string {
	hash := sha256.New()
	hash.Write(b)

	return fmt.Sprintf("%x", hash.Sum(nil))
}
