package util

import (
	"regexp"
)

// RegexStringSubMatchToMap accepts a regexp pattern and a string and returns a mapping of named
// capture groups to their found value. Obviously this only works with named capture groups.
func RegexStringSubMatchToMap(p *regexp.Regexp, s string) map[string]string {
	match := p.FindStringSubmatch(s)

	paramsMap := make(map[string]string)

	for i, name := range p.SubexpNames() {
		if i > 0 && i <= len(match) {
			paramsMap[name] = match[i]
		}
	}

	return paramsMap
}
