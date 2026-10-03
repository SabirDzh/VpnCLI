package uri

import (
	"fmt"
	"net/url"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ParseTrojan parses trojan://password@host:port?sni=..#name
func ParseTrojan(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: trojan: %w", domain.ErrParse, err)
	}
	if u.User == nil {
		return domain.Profile{}, fmt.Errorf("%w: trojan: missing password", domain.ErrParse)
	}
	pass := u.User.Username()
	host := u.Hostname()
	port, err := portOf(u)
	if err != nil {
		return domain.Profile{}, err
	}
	q := query(u)
	typ := first(q, "type", "tcp")
	p := domain.Profile{
		Protocol: domain.ProtocolTrojan,
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{
			Password:  pass,
			Security:  "tls",
			Transport: normalizeTransport(typ),
			SNI:       first(q, "sni", host),
			FP:        q.Get("fp"),
			Path:      q.Get("path"),
			Host:      q.Get("host"),
			Insecure:  boolParam(q, "allowInsecure", "insecure", "skip-cert-verify"),
		},
	}
	p.Name = nameOf(u, "")
	finish(&p)
	return p, nil
}
