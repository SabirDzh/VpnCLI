package singbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/core"
	"github.com/SabirDzh/VpnCLI/internal/domain"
)

func testOpts() core.Options {
	return core.Options{
		TUNEnabled: true, TUNName: "tun0", MTU: 9000, Stack: "system",
		AutoRoute: true, StrictRoute: true,
		MixedEnabled: true, MixedPort: 10808, LogLevel: "info",
	}
}

func TestGolden(t *testing.T) {
	cases := map[string]domain.Profile{
		"vless-reality": {
			Protocol: domain.ProtocolVLESS,
			Endpoint: domain.Endpoint{Host: "203.0.113.10", Port: 443},
			Settings: domain.ProtocolSettings{
				UUID: "11111111-2222-4333-8444-555555555555", Flow: "xtls-rprx-vision",
				Security: "reality", SNI: "example.com", FP: "chrome",
				RealityPublicKey: "AAECAwQFBgcICQoLDA0ODxAREhMUFRYXGBkaGxwdHh8", RealityShortID: "abcd",
			},
		},
		"vmess-ws-tls": {
			Protocol: domain.ProtocolVMess,
			Endpoint: domain.Endpoint{Host: "198.51.100.7", Port: 1234},
			Settings: domain.ProtocolSettings{
				UUID:     "1222e38e-5c9a-4228-9a1d-2d3dc1234567",
				Security: "tls", Transport: "ws", SNI: "cdn.example.com",
				Path: "/path", Host: "cdn.example.com",
			},
		},
		"trojan": {
			Protocol: domain.ProtocolTrojan,
			Endpoint: domain.Endpoint{Host: "192.0.2.5", Port: 443},
			Settings: domain.ProtocolSettings{Password: "secret", SNI: "front.example.com"},
		},
		"shadowsocks": {
			Protocol: domain.ProtocolShadowsocks,
			Endpoint: domain.Endpoint{Host: "192.0.2.9", Port: 8388},
			Settings: domain.ProtocolSettings{Method: "aes-256-gcm", Password: "pass123"},
		},
		"hysteria2": {
			Protocol: domain.ProtocolHysteria2,
			Endpoint: domain.Endpoint{Host: "203.0.113.20", Port: 8443},
			Settings: domain.ProtocolSettings{
				Password: "hy2pass", SNI: "front.example.com",
				ObfsType: "salamander", ObfsPassword: "obfpass",
				UpMbps: 100, DownMbps: 200,
			},
		},
		"hysteria2-min": {
			Protocol: domain.ProtocolHysteria2,
			Endpoint: domain.Endpoint{Host: "198.51.100.9", Port: 443},
			Settings: domain.ProtocolSettings{Password: "pw"},
		},
	}
	var b Builder
	for name, p := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := b.Build(p, testOpts())
			if err != nil {
				t.Fatal(err)
			}
			golden := filepath.Join("..", "..", "..", "testdata", "singbox", name+".golden")
			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.WriteFile(golden, append(got, '\n'), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(golden)
			if err != nil {
				t.Fatalf("read golden (run with UPDATE_GOLDEN=1): %v", err)
			}
			// trailing newline tolerance
			gotS := string(got) + "\n"
			if gotS != string(want) {
				t.Fatalf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s",
					name, gotS, want)
			}
		})
	}
}

func TestRawPassthrough(t *testing.T) {
	var b Builder
	raw := []byte(`{"log":{"level":"info"}}`)
	p := domain.Profile{Protocol: domain.ProtocolVLESS, Raw: raw}
	got, err := b.Build(p, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(raw) {
		t.Fatalf("raw not passed through: %s", got)
	}
}

func TestBuildValidation(t *testing.T) {
	var b Builder
	if _, err := b.Build(domain.Profile{Protocol: domain.ProtocolVLESS}, testOpts()); err == nil {
		t.Fatal("expected error for vless without uuid")
	}
	if _, err := b.Build(domain.Profile{Protocol: "wireguard"}, testOpts()); err == nil {
		t.Fatal("expected error for unknown protocol")
	}
	if _, err := b.Build(domain.Profile{Protocol: domain.ProtocolHysteria2}, testOpts()); err == nil {
		t.Fatal("expected error for hysteria2 without password")
	}
}

func TestCheckMinVersion(t *testing.T) {
	if err := CheckMinVersion("1.14.2", "1.14.0"); err != nil {
		t.Fatal(err)
	}
	if err := CheckMinVersion("1.13.9", "1.14.0"); err == nil {
		t.Fatal("expected below-minimum error")
	}
	if err := CheckMinVersion("1.14.0", "1.14.0"); err != nil {
		t.Fatal(err)
	}
}

func TestSupportsAndFactory(t *testing.T) {
	c, err := New(core.Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if c.Name() != "sing-box" {
		t.Fatalf("name: %s", c.Name())
	}
	if !c.Supports(domain.Profile{Protocol: domain.ProtocolVLESS}) {
		t.Fatal("should support vless")
	}
	if c.Supports(domain.Profile{Protocol: "wireguard"}) {
		t.Fatal("should not support wireguard")
	}
	if !c.Supports(domain.Profile{Protocol: "wireguard", Raw: []byte(`{}`)}) {
		t.Fatal("raw should pass through")
	}
	if _, err := FindBinary(""); err == nil {
		t.Log("sing-box found in PATH (optional)")
	} else {
		t.Logf("sing-box missing as expected in dev env: %v", err)
	}
}

func TestTUNAutoName(t *testing.T) {
	var b Builder
	p := domain.Profile{
		Protocol: domain.ProtocolTrojan,
		Endpoint: domain.Endpoint{Host: "h.example", Port: 443},
		Settings: domain.ProtocolSettings{Password: "pw"},
	}
	opts := testOpts()
	opts.TUNName = "" // auto: no interface_name, sing-box picks utunX/tun0
	got, err := b.Build(p, opts)
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Inbounds []map[string]any `json:"inbounds"`
	}
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	if _, ok := cfg.Inbounds[0]["interface_name"]; ok {
		t.Fatal("interface_name must be omitted for auto mode")
	}
}
