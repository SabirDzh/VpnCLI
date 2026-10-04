package uri

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

func TestParseVLESSReality(t *testing.T) {
	raw := "vless://11111111-2222-4333-8444-555555555555@203.0.113.10:443" +
		"?encryption=none&flow=xtls-rprx-vision&security=reality&sni=example.com" +
		"&fp=chrome&pbk=PUBKEY123&sid=abcd&type=tcp#Test-VLESS"
	p, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolVLESS {
		t.Fatalf("protocol = %s", p.Protocol)
	}
	if p.Endpoint.Host != "203.0.113.10" || p.Endpoint.Port != 443 {
		t.Fatalf("endpoint = %+v", p.Endpoint)
	}
	s := p.Settings
	if s.UUID != "11111111-2222-4333-8444-555555555555" || s.Flow != "xtls-rprx-vision" {
		t.Fatalf("settings = %+v", s)
	}
	if s.Security != "reality" || s.RealityPublicKey != "PUBKEY123" || s.RealityShortID != "abcd" {
		t.Fatalf("reality = %+v", s)
	}
	if p.Name != "Test-VLESS" || p.ID == "" {
		t.Fatalf("name/id = %q %q", p.Name, p.ID)
	}
}

func TestParseVLESSWsTLS(t *testing.T) {
	raw := "vless://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee@host.example:8443" +
		"?security=tls&sni=host.example&type=ws&path=%2Fws&host=host.example#WS"
	p, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Settings.Transport != "ws" || p.Settings.Path != "/ws" {
		t.Fatalf("transport = %+v", p.Settings)
	}
	if p.Settings.Security != "tls" || p.Settings.SNI != "host.example" {
		t.Fatalf("tls = %+v", p.Settings)
	}
}

func TestParseVMess(t *testing.T) {
	js := `{"add":"198.51.100.7","port":1234,"id":"1222e38e-5c9a-4228-9a1d-2d3dc1234567",` +
		`"aid":0,"net":"ws","type":"none","tls":"tls","host":"cdn.example.com",` +
		`"path":"/path","ps":"vmess-test","scy":"auto"}`
	raw := "vmess://" + base64.StdEncoding.EncodeToString([]byte(js))
	p, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolVMess || p.Endpoint.Port != 1234 {
		t.Fatalf("got %+v", p)
	}
	if p.Settings.Transport != "ws" || p.Settings.Security != "tls" {
		t.Fatalf("settings = %+v", p.Settings)
	}
	if p.Name != "vmess-test" {
		t.Fatalf("name = %q", p.Name)
	}
}

func TestParseTrojan(t *testing.T) {
	raw := "trojan://secret-pass@192.0.2.5:443?sni=front.example.com#Trojan1"
	p, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolTrojan || p.Settings.Password != "secret-pass" {
		t.Fatalf("got %+v", p)
	}
	if p.Settings.Security != "tls" || p.Settings.SNI != "front.example.com" {
		t.Fatalf("tls = %+v", p.Settings)
	}
}

func TestParseShadowsocksSIP002(t *testing.T) {
	cred := base64.URLEncoding.EncodeToString([]byte("aes-256-gcm:pass123"))
	raw := "ss://" + cred + "@192.0.2.9:8388#SS1"
	p, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolShadowsocks {
		t.Fatalf("protocol = %s", p.Protocol)
	}
	if p.Settings.Method != "aes-256-gcm" || p.Settings.Password != "pass123" {
		t.Fatalf("settings = %+v", p.Settings)
	}
	if p.Endpoint.Port != 8388 || p.Name != "SS1" {
		t.Fatalf("got %+v", p)
	}
}

func TestParseShadowsocksLegacy(t *testing.T) {
	inner := base64.StdEncoding.EncodeToString([]byte("chacha20-ietf-poly1305:pw@10.0.0.3:8388"))
	raw := "ss://" + inner + "#Legacy"
	p, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Settings.Method != "chacha20-ietf-poly1305" || p.Endpoint.Host != "10.0.0.3" {
		t.Fatalf("got %+v", p)
	}
}

func TestParseUnknownScheme(t *testing.T) {
	if _, err := Parse("wireguard://foo@bar:123"); err == nil {
		t.Fatal("expected error")
	}
}

func TestStableID(t *testing.T) {
	a, _ := Parse("trojan://pw@h.example:443#x")
	b, _ := Parse("trojan://pw@h.example:443#y")
	if a.ID != b.ID {
		t.Fatal("ID must survive rename (subscription refresh)")
	}
	c, _ := Parse("trojan://other@h.example:443#x")
	if a.ID == c.ID {
		t.Fatal("ID must change with credentials")
	}
	if !strings.HasPrefix(a.ID, "") || len(a.ID) != 16 {
		t.Fatalf("ID shape: %q", a.ID)
	}
}

func TestParseHysteria2(t *testing.T) {
	raw := "hysteria2://pass123@203.0.113.20:8443?sni=front.example.com" +
		"&obfs=salamander&obfs-password=obfpass&upmbps=100&downmbps=200#HY2"
	p, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolHysteria2 {
		t.Fatalf("protocol = %s", p.Protocol)
	}
	s := p.Settings
	if s.Password != "pass123" || s.SNI != "front.example.com" {
		t.Fatalf("settings = %+v", s)
	}
	if s.ObfsType != "salamander" || s.ObfsPassword != "obfpass" {
		t.Fatalf("obfs = %+v", s)
	}
	if s.UpMbps != 100 || s.DownMbps != 200 {
		t.Fatalf("speed = %+v", s)
	}
	if p.Name != "HY2" || p.Endpoint.Port != 8443 {
		t.Fatalf("got %+v", p)
	}
}

func TestParseHysteria2Minimal(t *testing.T) {
	p, err := Parse("hy2://pw@h.example:443#x")
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolHysteria2 || p.Settings.Password != "pw" {
		t.Fatalf("got %+v", p)
	}
	if _, err := Parse("hysteria2://h.example:443#x"); err == nil {
		t.Fatal("expected missing password error")
	}
}

func TestParseTrojanAllowInsecure(t *testing.T) {
	p, err := Parse("trojan://pw@h.example:443?security=tls&sni=h.example&allowInsecure=1&type=tcp#t")
	if err != nil {
		t.Fatal(err)
	}
	if !p.Settings.Insecure {
		t.Fatalf("insecure not parsed: %+v", p.Settings)
	}
	p2, err := Parse("trojan://pw@h.example:443#t2")
	if err != nil {
		t.Fatal(err)
	}
	if p2.Settings.Insecure {
		t.Fatal("insecure must default to false")
	}
}

func TestParseTUIC(t *testing.T) {
	p, err := Parse("tuic://uuid-x:pass-w@vpn.example:8443?congestion_control=bbr&sni=s.example&alpn=h3#TUIC%20node")
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolTUIC || p.Settings.UUID != "uuid-x" || p.Settings.Password != "pass-w" ||
		p.Settings.Congestion != "bbr" || p.Settings.SNI != "s.example" || p.Settings.ALPN != "h3" ||
		p.Endpoint.Host != "vpn.example" || p.Endpoint.Port != 8443 || p.Name != "TUIC node" {
		t.Fatalf("parsed: %+v", p)
	}
	if _, err := Parse("tuic://vpn.example:8443#x"); err == nil {
		t.Fatal("expected missing user:password error")
	}
}

func TestParseAnyTLSSSH(t *testing.T) {
	p, err := Parse("anytls://pass-w@vpn.example:443?sni=s.example&insecure=1#any")
	if err != nil || p.Protocol != domain.ProtocolAnyTLS || p.Settings.Password != "pass-w" ||
		p.Settings.SNI != "s.example" || !p.Settings.Insecure {
		t.Fatalf("anytls: %+v err=%v", p, err)
	}
	s, err := Parse("ssh://root@vpn.example:22?password=secret#box")
	if err != nil || s.Protocol != domain.ProtocolSSH || s.Settings.User != "root" ||
		s.Settings.Password != "secret" || s.Endpoint.Port != 22 {
		t.Fatalf("ssh: %+v err=%v", s, err)
	}
}
