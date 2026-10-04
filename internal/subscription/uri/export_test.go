package uri

import (
	"strings"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

func TestToURIRoundtrip(t *testing.T) {
	fixtures := []domain.Profile{
		{
			Name:     "my vless",
			Protocol: domain.ProtocolVLESS,
			Endpoint: domain.Endpoint{Host: "v.example", Port: 443},
			Settings: domain.ProtocolSettings{
				UUID: "11111111-2222-4333-8444-555555555555", Flow: "xtls-rprx-vision",
				Security: "reality", Transport: "tcp", SNI: "front.example", FP: "chrome",
				RealityPublicKey: "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8", RealityShortID: "abcd",
			},
		},
		{
			Name:     "vmess node",
			Protocol: domain.ProtocolVMess,
			Endpoint: domain.Endpoint{Host: "m.example", Port: 8443},
			Settings: domain.ProtocolSettings{
				UUID: "1222e38e-5c9a-4228-9a1d-2d3dc1234567", Cipher: "auto",
				Security: "tls", Transport: "ws", SNI: "cdn.example", Path: "/wpath", Host: "cdn.example",
			},
		},
		{
			Name:     "troj",
			Protocol: domain.ProtocolTrojan,
			Endpoint: domain.Endpoint{Host: "t.example", Port: 443},
			Settings: domain.ProtocolSettings{Password: "pass", Security: "tls", SNI: "front.example"},
		},
		{
			Name:     "ss node",
			Protocol: domain.ProtocolShadowsocks,
			Endpoint: domain.Endpoint{Host: "s.example", Port: 8388},
			Settings: domain.ProtocolSettings{Method: "aes-256-gcm", Password: "sspass"},
		},
		{
			Name:     "hy2",
			Protocol: domain.ProtocolHysteria2,
			Endpoint: domain.Endpoint{Host: "h.example", Port: 8443},
			Settings: domain.ProtocolSettings{Password: "hy2pass", SNI: "front.example",
				ObfsType: "salamander", ObfsPassword: "obfpass", Insecure: true},
		},
		{
			Name:     "tuic",
			Protocol: domain.ProtocolTUIC,
			Endpoint: domain.Endpoint{Host: "q.example", Port: 8443},
			Settings: domain.ProtocolSettings{UUID: "11111111-2222-4333-8444-555555555555",
				Password: "tqp", Congestion: "bbr", SNI: "front.example", ALPN: "h3"},
		},
		{
			Name:     "anytls",
			Protocol: domain.ProtocolAnyTLS,
			Endpoint: domain.Endpoint{Host: "a.example", Port: 443},
			Settings: domain.ProtocolSettings{Password: "atpass", SNI: "front.example"},
		},
		{
			Name:     "ssh box",
			Protocol: domain.ProtocolSSH,
			Endpoint: domain.Endpoint{Host: "sh.example", Port: 22},
			Settings: domain.ProtocolSettings{User: "root", Password: "sshpass"},
		},
	}
	// URI scheme may differ from the neutral protocol id.
	schemes := map[domain.Protocol]string{
		domain.ProtocolVLESS:       "vless://",
		domain.ProtocolVMess:       "vmess://",
		domain.ProtocolTrojan:      "trojan://",
		domain.ProtocolShadowsocks: "ss://",
		domain.ProtocolHysteria2:   "hy2://",
		domain.ProtocolTUIC:        "tuic://",
		domain.ProtocolAnyTLS:      "anytls://",
		domain.ProtocolSSH:         "ssh://",
	}
	for _, f := range fixtures {
		uri, err := ToURI(f)
		if err != nil {
			t.Fatalf("%s: %v", f.Protocol, err)
		}
		if !strings.HasPrefix(uri, schemes[f.Protocol]) {
			t.Fatalf("%s: wrong scheme in %q", f.Protocol, uri)
		}
		p, err := Parse(uri)
		if err != nil {
			t.Fatalf("%s: parse(%q): %v", f.Protocol, uri, err)
		}
		if p.Protocol != f.Protocol || p.Endpoint != f.Endpoint || p.Name != f.Name {
			t.Fatalf("%s: roundtrip mismatch:\nwant %+v\ngot  %+v", f.Protocol, f, p)
		}
		compareSettings(t, f, p)
	}
}

// compareSettings checks the fields each protocol round-trips exactly.
func compareSettings(t *testing.T, want, got domain.Profile) {
	t.Helper()
	s, g := want.Settings, got.Settings
	eq := func(name, a, b string) {
		if a != b {
			t.Fatalf("%s: %s want %q got %q", want.Protocol, name, a, b)
		}
	}
	switch want.Protocol {
	case domain.ProtocolVLESS:
		eq("uuid", s.UUID, g.UUID)
		eq("flow", s.Flow, g.Flow)
		eq("security", s.Security, g.Security)
		eq("sni", s.SNI, g.SNI)
		eq("fp", s.FP, g.FP)
		eq("pbk", s.RealityPublicKey, g.RealityPublicKey)
		eq("sid", s.RealityShortID, g.RealityShortID)
	case domain.ProtocolVMess:
		eq("uuid", s.UUID, g.UUID)
		eq("security", s.Security, g.Security)
		eq("transport", s.Transport, g.Transport)
		eq("sni", s.SNI, g.SNI)
		eq("path", s.Path, g.Path)
		eq("host", s.Host, g.Host)
		eq("cipher", s.Cipher, g.Cipher)
	case domain.ProtocolTrojan:
		eq("password", s.Password, g.Password)
		eq("sni", s.SNI, g.SNI)
	case domain.ProtocolShadowsocks:
		eq("method", s.Method, g.Method)
		eq("password", s.Password, g.Password)
	case domain.ProtocolHysteria2:
		eq("password", s.Password, g.Password)
		eq("sni", s.SNI, g.SNI)
		eq("obfs", s.ObfsType, g.ObfsType)
		eq("obfs-password", s.ObfsPassword, g.ObfsPassword)
		if s.Insecure != g.Insecure {
			t.Fatalf("hy2: insecure want %v got %v", s.Insecure, g.Insecure)
		}
	case domain.ProtocolTUIC:
		eq("uuid", s.UUID, g.UUID)
		eq("password", s.Password, g.Password)
		eq("congestion", s.Congestion, g.Congestion)
		eq("sni", s.SNI, g.SNI)
		eq("alpn", s.ALPN, g.ALPN)
	case domain.ProtocolAnyTLS:
		eq("password", s.Password, g.Password)
		eq("sni", s.SNI, g.SNI)
	case domain.ProtocolSSH:
		eq("user", s.User, g.User)
		eq("password", s.Password, g.Password)
	}
}

func TestToURIUnknownProtocol(t *testing.T) {
	if _, err := ToURI(domain.Profile{Protocol: "mystic"}); err == nil {
		t.Fatal("unknown protocol must error")
	}
}
