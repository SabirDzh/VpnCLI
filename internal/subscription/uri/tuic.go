package uri

import (
	"fmt"
	"net/url"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ParseTUIC parses tuic://uuid:password@host:port?congestion_control=..&sni=..&alpn=..#name
func ParseTUIC(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: tuic: %w", domain.ErrParse, err)
	}
	if u.User == nil || u.User.Username() == "" {
		return domain.Profile{}, fmt.Errorf("%w: tuic: missing user:password", domain.ErrParse)
	}
	host := u.Hostname()
	port, err := portOf(u)
	if err != nil {
		return domain.Profile{}, err
	}
	q := query(u)
	pass, _ := u.User.Password()
	p := domain.Profile{
		Protocol: domain.ProtocolTUIC,
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{
			UUID:       u.User.Username(),
			Password:   pass,
			Congestion: q.Get("congestion_control"),
			SNI:        first(q, "sni", host),
			ALPN:       q.Get("alpn"),
			Insecure:   boolParam(q, "allowInsecure", "insecure"),
		},
	}
	p.Name = nameOf(u, "")
	finish(&p)
	return p, nil
}
