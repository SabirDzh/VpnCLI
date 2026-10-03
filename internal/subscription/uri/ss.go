package uri

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ParseShadowsocks parses both SIP002 and legacy forms:
// SIP002: ss://base64(method:password)@host:port#name
// legacy: ss://base64(method:password@host:port)#name
func ParseShadowsocks(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: ss: %v", domain.ErrParse, err)
	}
	name := nameOf(u, "")
	var method, password, host string
	var port uint16

	if u.User != nil && u.Hostname() != "" {
		// SIP002
		cred, err := decodeB64(u.User.Username())
		if err != nil {
			// unencoded method:password
			cred = []byte(u.User.Username())
		}
		m, pw, ok := splitCred(string(cred))
		if !ok {
			return domain.Profile{}, fmt.Errorf("%w: ss: bad credentials", domain.ErrParse)
		}
		method, password = m, pw
		host = u.Hostname()
		p, err := portOf(u)
		if err != nil {
			return domain.Profile{}, err
		}
		port = p
	} else {
		// legacy: whole part after ss:// is base64
		body := strings.TrimPrefix(raw, "ss://")
		if i := strings.Index(body, "#"); i >= 0 {
			body = body[:i]
		}
		decoded, err := decodeB64(strings.TrimSpace(body))
		if err != nil {
			return domain.Profile{}, fmt.Errorf("%w: ss legacy base64: %v", domain.ErrParse, err)
		}
		s := string(decoded)
		at := strings.LastIndex(s, "@")
		if at < 0 {
			return domain.Profile{}, fmt.Errorf("%w: ss: missing @", domain.ErrParse)
		}
		m, pw, ok := splitCred(s[:at])
		if !ok {
			return domain.Profile{}, fmt.Errorf("%w: ss: bad credentials", domain.ErrParse)
		}
		method, password = m, pw
		hp := s[at+1:]
		h, ps, err := net.SplitHostPort(hp)
		if err != nil {
			return domain.Profile{}, fmt.Errorf("%w: ss: bad host:port", domain.ErrParse)
		}
		n, err := strconv.Atoi(ps)
		if err != nil || n <= 0 || n > 65535 {
			return domain.Profile{}, fmt.Errorf("%w: ss: bad port", domain.ErrParse)
		}
		host, port = h, uint16(n)
	}

	p := domain.Profile{
		Protocol: domain.ProtocolShadowsocks,
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{Method: method, Password: password},
	}
	p.Name = name
	// plugin params (obfs) live in fragment/query of SIP002; ignored in MVP.
	finish(&p)
	return p, nil
}

func splitCred(s string) (method, password string, ok bool) {
	i := strings.Index(s, ":")
	if i <= 0 {
		return "", "", false
	}
	return s[:i], s[i+1:], true
}
