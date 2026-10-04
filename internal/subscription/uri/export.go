package uri

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// ToURI serializes a profile back into its share-URI form. The result is
// re-importable by Parse: Parse(ToURI(p)) preserves endpoint, name and
// the protocol's identity fields.
func ToURI(p domain.Profile) (string, error) {
	s := p.Settings
	q := url.Values{}
	var u url.URL
	switch p.Protocol {
	case domain.ProtocolVLESS:
		u = url.URL{Scheme: "vless", User: url.User(s.UUID), Host: hostPort(p)}
		q.Set("type", transportParam(s.Transport))
		if s.Security != "" && s.Security != "none" {
			q.Set("security", s.Security)
		}
		if s.SNI != "" {
			q.Set("sni", s.SNI)
		}
		if s.FP != "" {
			q.Set("fp", s.FP)
		}
		if s.Flow != "" {
			q.Set("flow", s.Flow)
		}
		if s.RealityPublicKey != "" {
			q.Set("pbk", s.RealityPublicKey)
		}
		if s.RealityShortID != "" {
			q.Set("sid", s.RealityShortID)
		}
		if s.ALPN != "" {
			q.Set("alpn", s.ALPN)
		}
	case domain.ProtocolVMess:
		return vmessURI(p)
	case domain.ProtocolTrojan:
		u = url.URL{Scheme: "trojan", User: url.User(s.Password), Host: hostPort(p)}
		if s.SNI != "" {
			q.Set("sni", s.SNI)
		}
		if s.Transport != "" && s.Transport != "tcp" {
			q.Set("type", s.Transport)
		}
	case domain.ProtocolShadowsocks:
		cred := base64.RawURLEncoding.EncodeToString([]byte(s.Method + ":" + s.Password))
		u = url.URL{Scheme: "ss", User: url.User(cred), Host: hostPort(p)}
	case domain.ProtocolHysteria2:
		u = url.URL{Scheme: "hy2", User: url.User(s.Password), Host: hostPort(p)}
		if s.SNI != "" {
			q.Set("sni", s.SNI)
		}
		if s.ObfsType != "" {
			q.Set("obfs", s.ObfsType)
		}
		if s.ObfsPassword != "" {
			q.Set("obfs-password", s.ObfsPassword)
		}
		if s.Insecure {
			q.Set("insecure", "1")
		}
	case domain.ProtocolTUIC:
		u = url.URL{Scheme: "tuic", User: url.UserPassword(s.UUID, s.Password), Host: hostPort(p)}
		if s.Congestion != "" {
			q.Set("congestion_control", s.Congestion)
		}
		if s.SNI != "" {
			q.Set("sni", s.SNI)
		}
		if s.ALPN != "" {
			q.Set("alpn", s.ALPN)
		}
	case domain.ProtocolAnyTLS:
		u = url.URL{Scheme: "anytls", User: url.User(s.Password), Host: hostPort(p)}
		if s.SNI != "" {
			q.Set("sni", s.SNI)
		}
		if s.Insecure {
			q.Set("insecure", "1")
		}
	case domain.ProtocolSSH:
		u = url.URL{Scheme: "ssh", User: url.User(s.User), Host: hostPort(p)}
		if s.Password != "" {
			q.Set("password", s.Password)
		}
	default:
		return "", fmt.Errorf("%w: cannot serialize %s", domain.ErrParse, p.Protocol)
	}
	u.RawQuery = q.Encode()
	u.Fragment = p.Name
	return u.String(), nil
}

func hostPort(p domain.Profile) string {
	return fmt.Sprintf("%s:%d", p.Endpoint.Host, p.Endpoint.Port)
}

// transportParam maps the neutral transport to URI param; tcp is the
// default and can be omitted.
func transportParam(t string) string {
	if t == "" {
		return "tcp"
	}
	return t
}

// vmessURI renders the classic V2Ray base64 JSON form.
func vmessURI(p domain.Profile) (string, error) {
	s := p.Settings
	tls := ""
	if s.Security == "tls" {
		tls = "tls"
	}
	v := vmessJSON{
		Add:  p.Endpoint.Host,
		Port: p.Endpoint.Port,
		ID:   s.UUID,
		Aid:  s.AlterID,
		Net:  transportParam(s.Transport),
		TLS:  tls,
		Host: s.Host,
		Path: s.Path,
		SNI:  s.SNI,
		PS:   p.Name,
		Scy:  s.Cipher,
		FP:   s.FP,
	}
	if v.Scy == "" {
		v.Scy = "auto"
	}
	data, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("%w: vmess json: %w", domain.ErrParse, err)
	}
	return "vmess://" + base64.RawURLEncoding.EncodeToString(data), nil
}

