package kubernetes

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"sync"
)

var (
	validDNSLabelConventionPatternsObj    *validDNSLabelConventionPatterns //nolint:gochecknoglobals
	validNSLabelConventionPatternsObjOnce sync.Once                        //nolint:gochecknoglobals
)

const (
	// NameMaxLen is the maximum length for a kubernetes name.
	NameMaxLen = 63
)

// ref. https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names
type validDNSLabelConventionPatterns struct {
	invalidChars          *regexp.Regexp
	startsWithNonAlphaNum *regexp.Regexp
	endsWithNonAlphaNum   *regexp.Regexp
}

func getDNSLabelConventionPatterns() *validDNSLabelConventionPatterns {
	validNSLabelConventionPatternsObjOnce.Do(func() {
		validDNSLabelConventionPatternsObj = &validDNSLabelConventionPatterns{
			invalidChars:          regexp.MustCompile(`[^a-z0-9\-]`),
			startsWithNonAlphaNum: regexp.MustCompile(`^[^a-z0-9]`),
			endsWithNonAlphaNum:   regexp.MustCompile(`[^a-z0-9]$`),
		}
	})

	return validDNSLabelConventionPatternsObj
}

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

// EnforceDNSLabelConvention attempts to enforce the RFC 1123 label name requirements on s.
func EnforceDNSLabelConvention(s string) string {
	p := getDNSLabelConventionPatterns()

	s = strings.ToLower(s)
	s = p.invalidChars.ReplaceAllString(s, "-")
	s = p.startsWithNonAlphaNum.ReplaceAllString(s, "z")
	s = p.endsWithNonAlphaNum.ReplaceAllString(s, "z")

	return s
}
