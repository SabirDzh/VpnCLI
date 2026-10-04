// Package singbox adapts the sing-box core to the core.Core interface:
// neutral profiles are translated to native JSON and run as an
// external process via process.Supervisor.
package singbox

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SabirDzh/VpnCLI/internal/core"
	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// Builder translates a neutral Profile into a sing-box 1.14 JSON config.
// TUN is the primary mode; mixed proxy is an optional second inbound.
type Builder struct{}

// Rule-set sources (verified remote .srs; sing-box downloads and caches them).
const (
	adsRuleSetURL      = "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs"
	trackersRuleSetURL = "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-public-tracker.srs"
	socialRuleSetURL   = "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-social-media-%21cn.srs"
)

// blocklistSets maps feature toggles to verified remote rule-sets.
var blocklistSets = []struct {
	flag bool
	tag  string
	url  string
}{}

// splitRule splits tokens into domain_suffix (hosts) and ip_cidr (prefixes).
// appRule routes traffic of the given process names to the outbound.
func appRule(apps []string, outbound string) map[string]any {
	procs := make([]any, len(apps))
	for i, name := range apps {
		procs[i] = name
	}
	return map[string]any{"process_name": procs, "action": "route", "outbound": outbound}
}

func splitRule(tokens []string, outbound string) map[string]any {
	var domains, cidrs []any
	for _, tok := range tokens {
		if strings.Contains(tok, "/") {
			cidrs = append(cidrs, tok)
		} else {
			domains = append(domains, tok)
		}
	}
	rule := map[string]any{"action": "route", "outbound": outbound}
	if len(domains) > 0 {
		rule["domain_suffix"] = domains
	}
	if len(cidrs) > 0 {
		rule["ip_cidr"] = cidrs
	}
	return rule
}

// Build implements core.ConfigBuilder.
func (Builder) Build(p domain.Profile, opts core.Options) ([]byte, error) {
	if len(p.Raw) > 0 {
		return p.Raw, nil // imported native config passes through
	}
	proxy, err := outbound(p)
	if err != nil {
		return nil, err
	}

	baseRules := []any{
		// DNS guard first: catch plaintext queries to ANY resolver
		// routed into the tunnel, not just protocol-detected DNS.
		map[string]any{"port": 53, "action": "hijack-dns"},
		map[string]any{"protocol": "dns", "action": "hijack-dns"},
		map[string]any{"ip_cidr": []string{"224.0.0.0/3", "ff00::/8"}, "action": "reject"},
		map[string]any{"ip_is_private": true, "action": "route", "outbound": "direct"},
	}
	rules := baseRules
	if len(opts.AppFirewall) > 0 {
		processes := make([]any, len(opts.AppFirewall))
		for i, name := range opts.AppFirewall {
			processes[i] = name
		}
		rules = append([]any{map[string]any{"process_name": processes, "action": "reject"}}, rules...)
	}
	var ruleSet []any
	sets := []struct {
		on  bool
		tag string
		url string
	}{
		{opts.Adblock, "geosite-ads", adsRuleSetURL},
		{opts.TrackerBlock, "geosite-trackers", trackersRuleSetURL},
		{opts.SocialBlock, "geosite-social", socialRuleSetURL},
	}
	for _, set := range sets {
		if !set.on {
			continue
		}
		ruleSet = append(ruleSet, map[string]any{
			"tag": set.tag, "type": "remote", "url": set.url, "download_detour": "direct",
		})
		rules = append(rules, map[string]any{"rule_set": []any{set.tag}, "action": "reject"})
	}
	// Split mode: "" = legacy dual-list behavior (exclude→direct,
	// include→proxy with final flip) kept for backward compatibility.
	mode := opts.SplitMode
	if mode == "" {
		mode = "legacy"
	}
	switch mode {
	case "exclude":
		if len(opts.SplitExcludeApps) > 0 {
			rules = append(rules, appRule(opts.SplitExcludeApps, "direct"))
		}
		if len(opts.SplitExclude) > 0 {
			rules = append(rules, splitRule(opts.SplitExclude, "direct"))
		}
	case "include":
		if len(opts.SplitIncludeApps) > 0 {
			rules = append(rules, appRule(opts.SplitIncludeApps, "proxy"))
		}
		if len(opts.SplitInclude) > 0 {
			rules = append(rules, splitRule(opts.SplitInclude, "proxy"))
		}
	case "legacy":
		if len(opts.SplitExclude) > 0 {
			rules = append(rules, splitRule(opts.SplitExclude, "direct"))
		}
		if len(opts.SplitInclude) > 0 {
			rules = append(rules, splitRule(opts.SplitInclude, "proxy"))
		}
	}
	final := "proxy"
	if (mode == "include" || mode == "legacy") &&
		(len(opts.SplitInclude) > 0 || len(opts.SplitIncludeApps) > 0) {
		final = "direct"
	}

	route := map[string]any{
		"rules":                   rules,
		"auto_detect_interface":   true,
		"final":                   final,
		"default_domain_resolver": map[string]any{"server": "local"},
	}
	// No find_process_mode: removed in sing-box 1.14, which enables
	// process search automatically when a process rule exists.
	if len(ruleSet) > 0 {
		route["rule_set"] = ruleSet
	}
	cfg := map[string]any{
		"log": map[string]any{"level": logLevel(opts.LogLevel)},
		"dns": map[string]any{
			"servers":  dnsServers(opts),
			"strategy": dnsStrategy(opts),
			"final":    "remote",
		},
		"inbounds":  inbounds(opts),
		"outbounds": []any{proxy, map[string]any{"type": "direct", "tag": "direct"}, map[string]any{"type": "block", "tag": "block"}},
		"route":     route,
		"experimental": map[string]any{
			"clash_api":  map[string]any{"external_controller": "127.0.0.1:9090", "default_mode": "rule"},
			"cache_file": map[string]any{"enabled": true},
		},
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, err
	}
	return out, nil
}

// dnsServers renders configured servers; every entry resolves through the
// proxy (doh) so even plain-IP resolvers stay protected inside the tunnel.
// The local server stays last: default_domain_resolver points to it.
func dnsServers(opts core.Options) []any {
	servers := opts.DNSServers
	if len(servers) == 0 {
		servers = []string{"1.1.1.1", "8.8.8.8"}
	}
	out := make([]any, 0, len(servers)+1)
	for i, s := range servers {
		tag := "remote"
		if i > 0 {
			tag = fmt.Sprintf("remote-%d", i)
		}
		out = append(out, map[string]any{
			"type": "https", "tag": tag, "server": s, "detour": "proxy",
		})
	}
	out = append(out, map[string]any{"type": "local", "tag": "local"})
	return out
}

func dnsStrategy(opts core.Options) string {
	switch opts.DNSStrategy {
	case "prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only":
		return opts.DNSStrategy
	}
	return "prefer_ipv4"
}

func logLevel(l string) string {
	switch l {
	case "debug", "info", "warn", "error":
		return l
	default:
		return "info"
	}
}

func inbounds(opts core.Options) []any {
	var in []any
	if opts.TUNEnabled {
		mtu := opts.MTU
		if mtu == 0 {
			mtu = 9000
		}
		tun := map[string]any{
			"type":         "tun",
			"tag":          "tun-in",
			"address":      []string{"172.18.0.1/30", "fdfe:dcba:9876::1/126"},
			"mtu":          mtu,
			"stack":        "system",
			"auto_route":   opts.AutoRoute,
			"strict_route": opts.StrictRoute,
		}
		if opts.TUNName != "" {
			// Explicit name (e.g. tun0 on Linux). Empty = auto:
			// sing-box picks tun0 on Linux, utunX on macOS.
			tun["interface_name"] = opts.TUNName
		}
		in = append(in, tun)
	}
	if opts.MixedEnabled {
		port := opts.MixedPort
		if port == 0 {
			port = 10808
		}
		in = append(in, map[string]any{
			"type":        "mixed",
			"tag":         "mixed-in",
			"listen":      "127.0.0.1",
			"listen_port": port,
		})
	}
	return in
}

func outbound(p domain.Profile) (map[string]any, error) {
	base := map[string]any{
		"tag":         "proxy",
		"server":      p.Endpoint.Host,
		"server_port": p.Endpoint.Port,
	}
	s := p.Settings
	switch p.Protocol {
	case domain.ProtocolVLESS:
		if s.UUID == "" {
			return nil, fmt.Errorf("vless: missing uuid")
		}
		base["type"] = "vless"
		base["uuid"] = s.UUID
		if s.Flow != "" {
			base["flow"] = s.Flow
		}
		if tlsCfg := tlsFor(s); tlsCfg != nil {
			base["tls"] = tlsCfg
		}
		if tr := transportFor(s); tr != nil {
			base["transport"] = tr
		}
	case domain.ProtocolVMess:
		if s.UUID == "" {
			return nil, fmt.Errorf("vmess: missing uuid")
		}
		base["type"] = "vmess"
		base["uuid"] = s.UUID
		if s.AlterID > 0 {
			base["alter_id"] = s.AlterID
		}
		if s.Cipher != "" {
			base["security"] = s.Cipher
		}
		if tlsCfg := tlsFor(s); tlsCfg != nil {
			base["tls"] = tlsCfg
		}
		if tr := transportFor(s); tr != nil {
			base["transport"] = tr
		}
	case domain.ProtocolTrojan:
		if s.Password == "" {
			return nil, fmt.Errorf("trojan: missing password")
		}
		base["type"] = "trojan"
		base["password"] = s.Password
		base["tls"] = tlsForcing(s)
		if tr := transportFor(s); tr != nil {
			base["transport"] = tr
		}
	case domain.ProtocolShadowsocks:
		if s.Password == "" || s.Method == "" {
			return nil, fmt.Errorf("shadowsocks: missing method/password")
		}
		base["type"] = "shadowsocks"
		base["method"] = s.Method
		base["password"] = s.Password
	case domain.ProtocolHysteria2:
		if s.Password == "" {
			return nil, fmt.Errorf("hysteria2: missing password")
		}
		base["type"] = "hysteria2"
		base["password"] = s.Password
		base["tls"] = hy2TLS(p)
		if s.ObfsType != "" {
			obfs := map[string]any{"type": s.ObfsType}
			if s.ObfsPassword != "" {
				obfs["password"] = s.ObfsPassword
			}
			base["obfs"] = obfs
		}
		if s.UpMbps > 0 {
			base["up_mbps"] = s.UpMbps
		}
		if s.DownMbps > 0 {
			base["down_mbps"] = s.DownMbps
		}
	default:
		return nil, fmt.Errorf("%w: %s", domain.ErrUnsupportedProtocol, p.Protocol)
	}
	return base, nil
}

// tlsFor returns nil when security is none/empty (plain connection).
func tlsFor(s domain.ProtocolSettings) map[string]any {
	switch s.Security {
	case "tls", "reality":
	default:
		return nil
	}
	return tlsForcing(s)
}

func tlsForcing(s domain.ProtocolSettings) map[string]any {
	t := map[string]any{"enabled": true}
	if s.SNI != "" {
		t["server_name"] = s.SNI
	}
	if s.Security == "reality" {
		r := map[string]any{"enabled": true}
		if s.RealityPublicKey != "" {
			r["public_key"] = s.RealityPublicKey
		}
		if s.RealityShortID != "" {
			r["short_id"] = s.RealityShortID
		}
		t["reality"] = r
	}
	if s.FP != "" {
		t["utls"] = map[string]any{"enabled": true, "fingerprint": s.FP}
	}
	if s.ALPN != "" {
		t["alpn"] = []string{s.ALPN}
	}
	if s.Insecure {
		t["insecure"] = true
	}
	return t
}

// hy2TLS builds mandatory TLS for Hysteria2: server_name falls back to the
// endpoint host, ALPN defaults to h3.
func hy2TLS(p domain.Profile) map[string]any {
	s := p.Settings
	t := map[string]any{"enabled": true}
	name := s.SNI
	if name == "" {
		name = p.Endpoint.Host
	}
	t["server_name"] = name
	alpn := s.ALPN
	if alpn == "" {
		alpn = "h3"
	}
	t["alpn"] = []string{alpn}
	if s.Insecure {
		t["insecure"] = true
	}
	return t
}

func transportFor(s domain.ProtocolSettings) map[string]any {
	switch s.Transport {
	case "ws":
		t := map[string]any{"type": "ws"}
		if s.Path != "" {
			t["path"] = s.Path
		}
		if s.Host != "" {
			t["headers"] = map[string]any{"Host": s.Host}
		}
		return t
	case "grpc":
		t := map[string]any{"type": "grpc"}
		if s.Path != "" {
			t["service_name"] = s.Path
		}
		return t
	default:
		return nil
	}
}
