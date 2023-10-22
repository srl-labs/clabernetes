package kubernetes

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const (
	// NameMaxLen is the maximum length for a kubernetes name.
	NameMaxLen = 63
)

// SafeConcatNameKubernetes concats all provided strings into a string joined by "-" - if the final
// string is greater than 63 characters, the string will be shortened, and a hash will be used at
// the end of the string to keep it unique, but safely within allowed lengths.
func SafeConcatNameKubernetes(name ...string) string {
	return SafeConcatNameMax(name, NameMaxLen)
}

// SafeConcatNameMax concats all provided strings into a string joined by "-" - if the final string
// is greater than max characters, the string will be shortened, and a hash will be used at the end
// of the string to keep it unique, but safely within allowed lengths.
func SafeConcatNameMax(name []string, max int) string {
	finalName := strings.Join(name, "-")

	if len(finalName) <= max {
		return finalName
	}

	digest := sha256.Sum256([]byte(finalName))

	return finalName[0:max-8] + "-" + hex.EncodeToString(digest[0:])[0:7]
}
