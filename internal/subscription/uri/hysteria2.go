package uri

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ParseHysteria2 parses hysteria2://password@host:port?params#name
// Params: sni/peer, obfs (salamander|gecko), obfs-password, insecure/skip-cert-verify,
// upmbps, downmbps, alpn.
func ParseHysteria2(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: hysteria2: %v", domain.ErrParse, err)
	}
	if u.User == nil {
		return domain.Profile{}, fmt.Errorf("%w: hysteria2: missing password", domain.ErrParse)
	}
	password := u.User.Username()
	if password == "" {
		return domain.Profile{}, fmt.Errorf("%w: hysteria2: missing password", domain.ErrParse)
	}
	host := u.Hostname()
	port, err := portOf(u)
	if err != nil {
		return domain.Profile{}, err
	}
	q := query(u)
	p := domain.Profile{
		Protocol: domain.ProtocolHysteria2,
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{
			Password:     password,
			Security:     "tls",
			SNI:          first(q, "sni", "peer"),
			ObfsType:     q.Get("obfs"),
			ObfsPassword: first(q, "obfs-password", "obfs_password"),
			ALPN:         q.Get("alpn"),
			UpMbps:       intParam(q, "upmbps"),
			DownMbps:     intParam(q, "downmbps"),
			Insecure:     boolParam(q, "insecure", "skip-cert-verify"),
		},
	}
	p.Name = nameOf(u, "")
	finish(&p)
	return p, nil
}

func intParam(q url.Values, key string) int {
	n, _ := strconv.Atoi(q.Get(key))
	if n < 0 {
		return 0
	}
	return n
}

func boolParam(q url.Values, keys ...string) bool {
	for _, k := range keys {
		switch q.Get(k) {
		case "1", "true", "True", "TRUE":
			return true
		}
	}
	return false
}
