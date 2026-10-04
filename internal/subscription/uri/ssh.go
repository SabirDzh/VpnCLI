package uri

import (
	"fmt"
	"net/url"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ParseSSH parses ssh://user@host:port?password=..#name
func ParseSSH(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: ssh: %w", domain.ErrParse, err)
	}
	if u.User == nil || u.User.Username() == "" {
		return domain.Profile{}, fmt.Errorf("%w: ssh: missing user", domain.ErrParse)
	}
	host := u.Hostname()
	port, err := portOf(u)
	if err != nil {
		return domain.Profile{}, err
	}
	p := domain.Profile{
		Protocol: domain.ProtocolSSH,
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{
			User:     u.User.Username(),
			Password: query(u).Get("password"),
		},
	}
	p.Name = nameOf(u, "")
	finish(&p)
	return p, nil
}
