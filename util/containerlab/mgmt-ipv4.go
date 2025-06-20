package containerlab

import (
    "fmt"
    "net"
    "regexp"
    "sync"

    claberneteserrors "github.com/srl-labs/clabernetes/errors"
    clabernetesutil  "github.com/srl-labs/clabernetes/util"
)

// TypedMgmtIPv4 holds a parsed IPv4 management address.
type TypedMgmtIPv4 struct {
    IP net.IP
}

// AsContainerlabMgmtIPv4Definition renders it back to the raw string form.
func (t *TypedMgmtIPv4) AsContainerlabMgmtIPv4Definition() string {
    return t.IP.String()
}

var (
    mgmtIPv4Pattern     *regexp.Regexp
    mgmtIPv4PatternOnce sync.Once
)

// GetMgmtIPv4Pattern returns a regex that captures a dotted-quad IPv4.
func GetMgmtIPv4Pattern() *regexp.Regexp {
    mgmtIPv4PatternOnce.Do(func() {
        // very simple IPv4 matcher: four groups of 1–3 digits
        mgmtIPv4Pattern = regexp.MustCompile(`(?P<ip>(?:\d{1,3}\.){3}\d{1,3})`)
    })
    return mgmtIPv4Pattern
}

// processMgmtIPv4Definition applies the regex and ensures it’s a valid IPv4.
func processMgmtIPv4Definition(re *regexp.Regexp, raw string) (*TypedMgmtIPv4, error) {
    if !re.MatchString(raw) {
        return nil, fmt.Errorf(
            "%w: mgmt-ipv4 %q doesn’t match IPv4 pattern",
            claberneteserrors.ErrParse, raw,
        )
    }

    parts := clabernetesutil.RegexStringSubMatchToMap(re, raw)
    ipStr := parts["ip"]

    ip := net.ParseIP(ipStr)
    if ip == nil || ip.To4() == nil {
        return nil, fmt.Errorf(
            "%w: failed parsing IPv4 address %q",
            claberneteserrors.ErrParse, ipStr,
        )
    }

    return &TypedMgmtIPv4{IP: ip.To4()}, nil
}

// ProcessMgmtIPv4Definition is the entry-point.
// It returns a TypedMgmtIPv4 or an error if the string is not a valid IPv4.
func ProcessMgmtIPv4Definition(raw string) (*TypedMgmtIPv4, error) {
    re := GetMgmtIPv4Pattern()
    return processMgmtIPv4Definition(re, raw)
}