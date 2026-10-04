package uri

import (
	"fmt"
	"net/url"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ParseAnyTLS parses anytls://password@host:port?sni=..&insecure=..#name
func ParseAnyTLS(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: anytls: %w", domain.ErrParse, err)
	}
	if u.User == nil || u.User.Username() == "" {
		return domain.Profile{}, fmt.Errorf("%w: anytls: missing password", domain.ErrParse)
	}
	host := u.Hostname()
	port, err := portOf(u)
	if err != nil {
		return domain.Profile{}, err
	}
	q := query(u)
	p := domain.Profile{
		Protocol: domain.ProtocolAnyTLS,
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{
			Password: u.User.Username(),
			Security: "tls",
			SNI:      first(q, "sni", host),
			Insecure: boolParam(q, "allowInsecure", "insecure"),
		},
	}
	p.Name = nameOf(u, "")
	finish(&p)
	return p, nil
}
