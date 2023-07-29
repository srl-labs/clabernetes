package util

import "math/rand"

const (
	alphabet      = "abcdefghijklmnopqrstuvwxyz"
	alphabetCount = 26
)

// RandomString returns a string of random alphabet characters `length` long.
func RandomString(length int) string {
	s := make([]byte, length)

	for i := 0; i < length; i++ {
		s[i] = alphabet[rand.Intn(alphabetCount)] //nolint: gosec
	}

	return string(s)
}
