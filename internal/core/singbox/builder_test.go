package singbox

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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

func testTrojan() domain.Profile {
	return domain.Profile{
		Protocol: domain.ProtocolTrojan,
		Endpoint: domain.Endpoint{Host: "192.0.2.5", Port: 443},
		Settings: domain.ProtocolSettings{Password: "x"},
	}
}

func TestDNSSectionAndGuard(t *testing.T) {
	var b Builder
	opts := testOpts()
	opts.DNSServers = []string{"9.9.9.9", "https://dns.quad9.net/dns-query"}
	opts.DNSStrategy = "ipv6_only"
	got, err := b.Build(testTrojan(), opts)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	dns := cfg["dns"].(map[string]any)
	if dns["strategy"] != "ipv6_only" {
		t.Fatalf("strategy: %v", dns["strategy"])
	}
	servers := dns["servers"].([]any)
	first := servers[0].(map[string]any)
	if first["server"] != "9.9.9.9" || first["type"] != "https" || first["detour"] != "proxy" {
		t.Fatalf("first server: %v", first)
	}
	second := servers[1].(map[string]any)
	if second["server"] != "https://dns.quad9.net/dns-query" {
		t.Fatalf("doh url server: %v", second)
	}
	// guard: port-53 hijack is the first route rule
	rules := cfg["route"].(map[string]any)["rules"].([]any)
	r0 := rules[0].(map[string]any)
	if r0["port"] != float64(53) || r0["action"] != "hijack-dns" {
		t.Fatalf("guard rule must be first: %v", r0)
	}
}

func TestExperimentalClashAPI(t *testing.T) {
	var b Builder
	got, err := b.Build(testTrojan(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	exp := cfg["experimental"].(map[string]any)
	api := exp["clash_api"].(map[string]any)
	if api["external_controller"] != "127.0.0.1:9090" {
		t.Fatalf("clash api: %v", api)
	}
	if exp["cache_file"].(map[string]any)["enabled"] != true {
		t.Fatal("cache_file must be enabled")
	}
}

func TestDNSFallbackDefaults(t *testing.T) {
	var b Builder
	got, err := b.Build(testTrojan(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	dns := cfg["dns"].(map[string]any)
	servers := dns["servers"].([]any)
	// 2 fallback remote servers + the trailing local resolver
	if len(servers) != 3 {
		t.Fatalf("fallback dns servers: %v", servers)
	}
	if dns["strategy"] != "prefer_ipv4" {
		t.Fatalf("fallback strategy: %v", dns["strategy"])
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

func buildRouted(t *testing.T, mutate func(*core.Options)) (rules []any, ruleSet []any, final any) {
	t.Helper()
	opts := testOpts()
	if mutate != nil {
		mutate(&opts)
	}
	var b Builder
	p := domain.Profile{
		Protocol: domain.ProtocolTrojan,
		Endpoint: domain.Endpoint{Host: "192.0.2.5", Port: 443},
		Settings: domain.ProtocolSettings{Password: "secret"},
	}
	got, err := b.Build(p, opts)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	route := cfg["route"].(map[string]any)
	if rs, ok := route["rule_set"].([]any); ok {
		ruleSet = rs
	}
	rules, _ = route["rules"].([]any)
	return rules, ruleSet, route["final"]
}

func ruleBySet(t *testing.T, rules []any, tag string) map[string]any {
	t.Helper()
	for _, r := range rules {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		rs, _ := m["rule_set"].([]any)
		for _, x := range rs {
			if x == tag {
				return m
			}
		}
	}
	t.Fatalf("rule for rule_set %q not found", tag)
	return nil
}

func ruleByOutbound(t *testing.T, rules []any, outbound string) map[string]any {
	t.Helper()
	for _, r := range rules {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if m["outbound"] != outbound || m["action"] != "route" {
			continue
		}
		// split rules carry concrete targets; the base private rule does not
		if _, ok := m["domain_suffix"]; ok {
			return m
		}
		if _, ok := m["ip_cidr"]; ok {
			return m
		}
	}
	t.Fatalf("route rule to %q not found", outbound)
	return nil
}

func TestBlocklistRules(t *testing.T) {
	rules, ruleSet, _ := buildRouted(t, func(o *core.Options) {
		o.Adblock = true
		o.TrackerBlock = true
	})
	if len(ruleSet) != 2 {
		t.Fatalf("want 2 rule_set definitions, got %d", len(ruleSet))
	}
	ads := ruleBySet(t, rules, "geosite-ads")
	if ads["action"] != "reject" {
		t.Fatalf("ads rule must reject: %v", ads)
	}
	trk := ruleBySet(t, rules, "geosite-trackers")
	if trk["action"] != "reject" {
		t.Fatalf("trackers rule must reject: %v", trk)
	}
	// ads must come before trackers (first match wins; both reject — order stability)
	adsIdx, trkIdx := -1, -1
	for i, r := range rules {
		m := r.(map[string]any)
		if rs, ok := m["rule_set"].([]any); ok && len(rs) > 0 && rs[0] == "geosite-ads" {
			adsIdx = i
		}
		if rs, ok := m["rule_set"].([]any); ok && len(rs) > 0 && rs[0] == "geosite-trackers" {
			trkIdx = i
		}
	}
	if adsIdx > trkIdx {
		t.Fatalf("ads rule must precede trackers: %d > %d", adsIdx, trkIdx)
	}
}

func TestNoBlocklistsByDefault(t *testing.T) {
	rules, ruleSet, _ := buildRouted(t, nil)
	if len(ruleSet) != 0 {
		t.Fatalf("no rule_set expected, got %d", len(ruleSet))
	}
	for _, r := range rules {
		m := r.(map[string]any)
		if _, ok := m["rule_set"]; ok {
			t.Fatalf("no rule_set rules expected: %v", m)
		}
	}
}

func TestSplitExcludeModeWithApps(t *testing.T) {
	var b Builder
	opts := testOpts()
	opts.SplitMode = "exclude"
	opts.SplitExclude = []string{"bank.example"}
	opts.SplitExcludeApps = []string{"Госуслуги"}
	got, err := b.Build(testTrojan(), opts)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(got, &cfg)
	route := cfg["route"].(map[string]any)
	if route["final"] != "proxy" {
		t.Fatalf("final: %v", route["final"])
	}
	var appRule, domRule bool
	for _, r := range route["rules"].([]any) {
		m := r.(map[string]any)
		if procs, ok := m["process_name"].([]any); ok {
			if procs[0] == "Госуслуги" && m["outbound"] == "direct" && m["action"] == "route" {
				appRule = true
			}
		}
		if doms, ok := m["domain_suffix"].([]any); ok && len(doms) > 0 && doms[0] == "bank.example" && m["outbound"] == "direct" {
			domRule = true
		}
	}
	if !appRule || !domRule {
		t.Fatalf("app=%v dom=%v", appRule, domRule)
	}
}

func TestSplitIncludeModeWithApps(t *testing.T) {
	var b Builder
	opts := testOpts()
	opts.SplitMode = "include"
	opts.SplitInclude = []string{"work.example"}
	opts.SplitIncludeApps = []string{"Miro"}
	got, err := b.Build(testTrojan(), opts)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(got, &cfg)
	route := cfg["route"].(map[string]any)
	if route["final"] != "direct" {
		t.Fatalf("include mode must flip final: %v", route["final"])
	}
	var appRule, domRule bool
	for _, r := range route["rules"].([]any) {
		m := r.(map[string]any)
		if procs, ok := m["process_name"].([]any); ok && procs[0] == "Miro" && m["outbound"] == "proxy" {
			appRule = true
		}
		if doms, ok := m["domain_suffix"].([]any); ok && len(doms) > 0 && doms[0] == "work.example" && m["outbound"] == "proxy" {
			domRule = true
		}
	}
	if !appRule || !domRule {
		t.Fatalf("app=%v dom=%v", appRule, domRule)
	}
}

func TestSplitOffIgnoresLists(t *testing.T) {
	var b Builder
	opts := testOpts()
	opts.SplitMode = "off"
	opts.SplitExclude = []string{"bank.example"}
	opts.SplitExcludeApps = []string{"Госуслуги"}
	opts.SplitInclude = []string{"work.example"}
	opts.SplitIncludeApps = []string{"Miro"}
	got, err := b.Build(testTrojan(), opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"process_name", "bank.example", "work.example", "Miro"} {
		if strings.Contains(string(got), banned) {
			t.Fatalf("off mode must ignore lists, found %q:\n%s", banned, got)
		}
	}
}

func TestSplitExcludeRule(t *testing.T) {
	rules, _, final := buildRouted(t, func(o *core.Options) {
		o.SplitExclude = []string{"bank.example", "192.168.0.0/16"}
	})
	direct := ruleByOutbound(t, rules, "direct")
	if final != "proxy" {
		t.Fatalf("exclude-only must keep final proxy, got %v", final)
	}
	if ds, _ := direct["domain_suffix"].([]any); len(ds) != 1 || ds[0] != "bank.example" {
		t.Fatalf("domain_suffix: %v", ds)
	}
	if cidr, _ := direct["ip_cidr"].([]any); len(cidr) != 1 || cidr[0] != "192.168.0.0/16" {
		t.Fatalf("ip_cidr: %v", cidr)
	}
}

func TestSplitIncludeFlipsFinal(t *testing.T) {
	rules, _, final := buildRouted(t, func(o *core.Options) {
		o.SplitExclude = []string{"bank.example"}
		o.SplitInclude = []string{"10.0.0.0/8", "corp.example"}
	})
	if final != "direct" {
		t.Fatalf("include list must flip final to direct, got %v", final)
	}
	inc := ruleByOutbound(t, rules, "proxy")
	if cidr, _ := inc["ip_cidr"].([]any); len(cidr) != 1 || cidr[0] != "10.0.0.0/8" {
		t.Fatalf("include ip_cidr: %v", cidr)
	}
	// exclude must precede include so bypasses win
	exIdx, incIdx := -1, -1
	for i, r := range rules {
		m := r.(map[string]any)
		if m["outbound"] == "direct" && m["action"] == "route" {
			exIdx = i
		}
		if m["outbound"] == "proxy" && m["action"] == "route" {
			incIdx = i
		}
	}
	if exIdx == -1 || incIdx == -1 || exIdx > incIdx {
		t.Fatalf("exclude must precede include: %d, %d", exIdx, incIdx)
	}
}

func TestSocialBlockRule(t *testing.T) {
	rules, ruleSet, _ := buildRouted(t, func(o *core.Options) { o.SocialBlock = true })
	if len(ruleSet) != 1 {
		t.Fatalf("want 1 rule_set, got %d", len(ruleSet))
	}
	if ruleBySet(t, rules, "geosite-social")["action"] != "reject" {
		t.Fatal("social rule must reject")
	}
}

func TestAppFirewallRules(t *testing.T) {
	// a process_name reject rule must lead the rule list
	var b Builder
	p := domain.Profile{
		Protocol: domain.ProtocolTrojan,
		Endpoint: domain.Endpoint{Host: "192.0.2.5", Port: 443},
		Settings: domain.ProtocolSettings{Password: "x"},
	}
	opts := testOpts()
	opts.AppFirewall = []string{"torrent", "Steam Helper"}
	got, err := b.Build(p, opts)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := json.Unmarshal(got, &cfg); err != nil {
		t.Fatal(err)
	}
	route := cfg["route"].(map[string]any)
	if _, present := route["find_process_mode"]; present {
		t.Fatal("find_process_mode was removed in sing-box 1.14 and must not be emitted")
	}
	rules := route["rules"].([]any)
	rejIdx := -1
	for i, r := range rules {
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		if pn, ok := m["process_name"].([]any); ok && len(pn) == 2 && pn[0] == "torrent" && pn[1] == "Steam Helper" {
			if m["action"] != "reject" {
				t.Fatalf("app rule must reject: %v", m)
			}
			rejIdx = i
		}
	}
	if rejIdx == -1 {
		t.Fatal("process_name reject rule not found")
	}
}

func TestNoAppFirewallByDefault(t *testing.T) {
	var b Builder
	p := domain.Profile{
		Protocol: domain.ProtocolTrojan,
		Endpoint: domain.Endpoint{Host: "192.0.2.5", Port: 443},
		Settings: domain.ProtocolSettings{Password: "x"},
	}
	got, err := b.Build(p, testOpts())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "process_name") || strings.Contains(string(got), "find_process_mode") {
		t.Fatalf("no process rules expected:\n%s", got)
	}
}
