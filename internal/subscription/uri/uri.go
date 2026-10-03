// Package uri parses proxy URIs (vless/vmess/trojan/ss) into domain.Profile.
// New schemes register here without touching callers.
package uri

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// Parse dispatches by URI scheme.
func Parse(raw string) (domain.Profile, error) {
	scheme := schemeOf(raw)
	fn, ok := parsers[scheme]
	if !ok {
		return domain.Profile{}, fmt.Errorf("%w: unknown scheme %q", domain.ErrParse, scheme)
	}
	return fn(raw)
}

var parsers = map[string]func(string) (domain.Profile, error){
	"vless":     ParseVLESS,
	"vmess":     ParseVMess,
	"trojan":    ParseTrojan,
	"ss":        ParseShadowsocks,
	"hysteria2": ParseHysteria2,
	"hy2":       ParseHysteria2,
}

// Schemes lists supported URI schemes.
func Schemes() []string {
	out := make([]string, 0, len(parsers))
	for s := range parsers {
		out = append(out, s)
	}
	return out
}

func schemeOf(raw string) string {
	i := strings.Index(raw, "://")
	if i < 0 {
		return ""
	}
	return strings.ToLower(raw[:i])
}

func query(u *url.URL) url.Values { return u.Query() }

func nameOf(u *url.URL, fallback string) string {
	if u.Fragment != "" {
		if n, err := url.PathUnescape(u.Fragment); err == nil {
			return n
		}
		return u.Fragment
	}
	return fallback
}

func finish(p *domain.Profile) {
	if p.Name == "" {
		p.Name = fmt.Sprintf("%s:%d", p.Endpoint.Host, p.Endpoint.Port)
	}
	p.ID = domain.ComputeID(p.Protocol, p.Endpoint.Host, p.Endpoint.Port, p.Settings.Identity())
}
