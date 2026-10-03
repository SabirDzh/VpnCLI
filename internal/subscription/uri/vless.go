package uri

import (
	"fmt"
	"net"
	"net/url"
	"strconv"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ParseVLESS parses vless://uuid@host:port?params#name
// Supports security=tls|reality|none, transports ws/grpc, flow, pbk/sid.
func ParseVLESS(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: vless: %v", domain.ErrParse, err)
	}
	if u.User == nil {
		return domain.Profile{}, fmt.Errorf("%w: vless: missing uuid", domain.ErrParse)
	}
	uuid := u.User.Username()
	host := u.Hostname()
	port, err := portOf(u)
	if err != nil {
		return domain.Profile{}, err
	}
	q := query(u)
	typ := first(q, "type", "tcp")
	security := first(q, "security", "none")
	if security == "" {
		security = "none"
	}
	p := domain.Profile{
		Protocol: domain.ProtocolVLESS,
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{
			UUID:             uuid,
			Flow:             q.Get("flow"),
			Security:         security,
			Transport:        normalizeTransport(typ),
			SNI:              first(q, "sni", q.Get("host")),
			FP:               q.Get("fp"),
			Path:             first(q, "path", q.Get("serviceName")),
			Host:             q.Get("host"),
			RealityPublicKey: first(q, "pbk", q.Get("publicKey")),
			RealityShortID:   first(q, "sid", q.Get("shortId")),
		},
	}
	p.Name = nameOf(u, "")
	finish(&p)
	return p, nil
}

func portOf(u *url.URL) (uint16, error) {
	h, port, err := net.SplitHostPort(u.Host)
	_ = h
	if err != nil {
		// url without explicit port (e.g. ss legacy) — caller handles
		return 0, fmt.Errorf("%w: missing port in %s", domain.ErrParse, u.Redacted())
	}
	n, err := strconv.Atoi(port)
	if err != nil || n <= 0 || n > 65535 {
		return 0, fmt.Errorf("%w: bad port %q", domain.ErrParse, port)
	}
	return uint16(n), nil
}

func first(q url.Values, keys ...string) string {
	for _, k := range keys {
		if v := q.Get(k); v != "" {
			return v
		}
	}
	return ""
}

func normalizeTransport(t string) string {
	switch t {
	case "ws", "grpc", "tcp":
		return t
	case "xhttp":
		return "ws"
	default:
		return t
	}
}
