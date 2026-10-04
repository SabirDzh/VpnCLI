# VpnCLI Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the ten approved workstreams: split-tunneling v2 (modes + apps + RF preset), DNS settings + guard, IPv6 kill-switch fix, tuic/anytls/ssh protocols, multiplex auto with fallback, network watchdog, full TUI settings editing + import, stats/log viewer, parser benchmark, export/share.

**Architecture:** Config schema grows (`dns`, `features.*`), the sing-box builder translates new Options into native config, `ConnectionService` gains Restart + watchdog orchestration, `SettingsAPI`/Settings screen expose everything, `Other` gains stats/log via sing-box clash_api. Export adds a URI serializer per protocol.

**Tech Stack:** Go 1.26, sing-box 1.14 (external binary, `charm.land` bubbletea/lipgloss v2 for TUI), viper config, pf (macOS kill switch).

**Spec:** `docs/superpowers/specs/2026-10-04-vpn-expansion-design.md`

## Global Constraints

- Deviation from spec §5/§6 (deliberate, recorded): **overlapped restart is unsafe with TUN** — two sing-box processes fight over the utun device/routes. Restart is fast sequential (`Stop` → `Start`, window ≈1–2 s); the kill switch anchor stays loaded so DNS/endpoint keep flowing rules-wise. Honest wording everywhere: «минимальный таймаут», not zero-loss.
- Every config key must round-trip through `config.Save` (viper `set()` list) — one missed key silently drops user settings.
- TDD: failing test before implementation for every task; full `go test ./...`, `gofmt -l .`, `go vet ./...` clean before each commit.
- Commits: `feat(scope): lowercase imperative`, one concern per commit, each verified with `go build ./...` in a throwaway worktree before moving on. Never push.
- Real sing-box (`/opt/homebrew/bin/sing-box`) is the oracle: `check -c` on every generated config shape added in this plan.
- TUI must not overflow 60×20 (`TestFitsMinimalTerminal` exists; extend it when adding rows).
- Russian UI copy, lowercase keys; settings changes persist and apply on next connect.

---

### Task 1: Config schema v2

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Produces: `Config.DNS{Servers []string, Strategy string}`, `Config.Features.SplitMode string`, `SplitExcludeApps []string`, `SplitIncludeApps []string`, `PresetApps bool`, `Multiplex string`. Defaults: `dns.servers=["1.1.1.1","8.8.8.8"]`, `dns.strategy="prefer_ipv4"`, `features.split_mode="exclude"`, `features.preset_apps=true`, `features.multiplex="auto"`.

- [ ] **Step 1: Write failing tests**

Add to `config_test.go`:

```go
func TestExpansionDefaults(t *testing.T) {
	c := Defaults()
	if len(c.DNS.Servers) != 2 || c.DNS.Servers[0] != "1.1.1.1" {
		t.Fatalf("dns servers: %+v", c.DNS.Servers)
	}
	if c.DNS.Strategy != "prefer_ipv4" || c.Features.SplitMode != "exclude" ||
		!c.Features.PresetApps || c.Features.Multiplex != "auto" {
		t.Fatalf("defaults: dns=%+v mode=%q preset=%v mux=%q",
			c.DNS, c.Features.SplitMode, c.Features.PresetApps, c.Features.Multiplex)
	}
}

func TestExpansionRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	src := `
dns:
  servers: ["9.9.9.9", "https://dns.quad9.net/dns-query"]
  strategy: ipv6_only
features:
  split_mode: include
  split_exclude_apps: [Госуслуги]
  split_include_apps: [Miro]
  preset_apps: false
  multiplex: off
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.DNS.Servers[0] != "9.9.9.9" || c.DNS.Strategy != "ipv6_only" ||
		c.Features.SplitMode != "include" || !c.Features.PresetApps == false ||
		len(c.Features.SplitExcludeApps) != 1 || c.Features.Multiplex != "off" {
		t.Fatalf("load: %+v %+v", c.DNS, c.Features)
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{"split_mode: include", "split_exclude_apps:", "Госуслуги", "preset_apps: false", "multiplex: off", "https://dns.quad9.net/dns-query"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("saved missing %q:\n%s", want, raw)
		}
	}
}

func TestExpansionValidation(t *testing.T) {
	mk := func(mut func(*Config)) Config {
		c := Defaults()
		mut(&c)
		return c
	}
	cases := []struct {
		name string
		cfg  Config
	}{
		{"bad mode", mk(func(c *Config) { c.Features.SplitMode = "both" })},
		{"bad mux", mk(func(c *Config) { c.Features.Multiplex = "maybe" })},
		{"bad strategy", mk(func(c *Config) { c.DNS.Strategy = "auto" })},
		{"too many dns", mk(func(c *Config) { c.DNS.Servers = []string{"1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4"} })},
		{"bad dns", mk(func(c *Config) { c.DNS.Servers = []string{"a b"} })},
		{"bad split app", mk(func(c *Config) { c.Features.SplitExcludeApps = []string{"bad name"} })},
	}
	for _, tc := range cases {
		if err := tc.cfg.Validate(); !errors.Is(err, domain.ErrInvalidConfig) {
			t.Fatalf("%s: expected invalid, got %v", tc.name, err)
		}
	}
}
```

- [ ] **Step 2: Run to verify fail**

Run: `go test ./internal/config/ 2>&1 | tail -5`
Expected: FAIL (fields do not exist — build error counts as red).

- [ ] **Step 3: Implement**

In `config.go` add to struct + defaults + validate + Save:

```go
// struct additions
DNS struct {
	Servers  []string `mapstructure:"servers" yaml:"servers"`
	Strategy string   `mapstructure:"strategy" yaml:"strategy"`
} `mapstructure:"dns" yaml:"dns"`
// inside Features:
SplitMode        string   `mapstructure:"split_mode" yaml:"split_mode"`
SplitExcludeApps []string `mapstructure:"split_exclude_apps" yaml:"split_exclude_apps"`
SplitIncludeApps []string `mapstructure:"split_include_apps" yaml:"split_include_apps"`
PresetApps       bool     `mapstructure:"preset_apps" yaml:"preset_apps"`
Multiplex        string   `mapstructure:"multiplex" yaml:"multiplex"`
```

Defaults: `c.DNS.Servers = []string{"1.1.1.1", "8.8.8.8"}`, `c.DNS.Strategy = "prefer_ipv4"`, `c.Features.SplitMode = "exclude"`, `c.Features.PresetApps = true`, `c.Features.Multiplex = "auto"`.
Load: `v.SetDefault("dns.servers", def.DNS.Servers)`, `v.SetDefault("dns.strategy", def.DNS.Strategy)`, `v.SetDefault("features.split_mode", def.Features.SplitMode)`, `v.SetDefault("features.preset_apps", def.Features.PresetApps)`, `v.SetDefault("features.multiplex", def.Features.Multiplex)`.
Validate additions:

```go
switch c.Features.SplitMode {
case "exclude", "include", "off":
default:
	return fmt.Errorf("%w: unknown features.split_mode %q", domain.ErrInvalidConfig, c.Features.SplitMode)
}
switch c.Features.Multiplex {
case "off", "on", "auto":
default:
	return fmt.Errorf("%w: unknown features.multiplex %q", domain.ErrInvalidConfig, c.Features.Multiplex)
}
switch c.DNS.Strategy {
case "prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only":
default:
	return fmt.Errorf("%w: unknown dns.strategy %q", domain.ErrInvalidConfig, c.DNS.Strategy)
}
if len(c.DNS.Servers) == 0 || len(c.DNS.Servers) > 3 {
	return fmt.Errorf("%w: dns.servers must hold 1..3 entries", domain.ErrInvalidConfig)
}
for _, s := range c.DNS.Servers {
	if s == "" || strings.ContainsAny(s, " \t") {
		return fmt.Errorf("%w: bad dns server %q", domain.ErrInvalidConfig, s)
	}
}
for _, list := range [][]string{c.Features.SplitExcludeApps, c.Features.SplitIncludeApps} {
	for _, app := range list {
		if app == "" || strings.ContainsAny(app, " \t/") {
			return fmt.Errorf("%w: bad split app name %q", domain.ErrInvalidConfig, app)
		}
	}
}
```

Save additions: `set("dns.servers", c.DNS.Servers)`, `set("dns.strategy", c.DNS.Strategy)`, `set("features.split_mode", c.Features.SplitMode)`, `set("features.split_exclude_apps", c.Features.SplitExcludeApps)`, `set("features.split_include_apps", c.Features.SplitIncludeApps)`, `set("features.preset_apps", c.Features.PresetApps)`, `set("features.multiplex", c.Features.Multiplex)`.

- [ ] **Step 4: Run to verify pass**

Run: `go test ./internal/config/`
Expected: PASS (all existing tests too).

- [ ] **Step 5: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): split modes with app lists, preset, multiplex and dns section"
```

---

### Task 2: RF preset and effective app lists

**Files:**
- Create: `internal/app/preset.go`
- Test: `internal/app/preset_test.go`

**Interfaces:**
- Produces: `var RFApps = []string{...}` (non-empty), `func EffectiveApps(base []string, preset bool) []string` — returns base + RFApps (dedup, base order first) when preset, else base.

- [ ] **Step 1: Write failing test**

```go
package app

import (
	"slices"
	"testing"
)

func TestEffectiveApps(t *testing.T) {
	base := []string{"Miro"}
	got := EffectiveApps(base, true)
	if !slices.Contains(got, "Miro") || !slices.Contains(got, RFApps[0]) {
		t.Fatalf("preset must merge: %v", got)
	}
	// no duplicates
	seen := map[string]bool{}
	for _, a := range got {
		if seen[a] {
			t.Fatalf("duplicate %q in %v", a, got)
		}
		seen[a] = true
	}
	if !slices.Equal(EffectiveApps(base, false), base) {
		t.Fatal("preset off must return base as-is")
	}
}
```

- [ ] **Step 2: Verify fail** — `go test ./internal/app/ -run TestEffectiveApps` → FAIL (undefined).

- [ ] **Step 3: Implement `internal/app/preset.go`**

```go
package app

// RFApps lists process names of apps that must bypass the VPN in Russia
// (banks, government, ride-hailing). macOS process names verified with
// `ps -axo comm=` during this task; keep both latin and cyrillic variants.
var RFApps = []string{
	"Госуслуги", "gosuslugi",
	"СберБанк", "СБОЛ", "SberBank", "СберБанк Онлайн",
	"Т-Банк", "Тинькофф", "Tinkoff", "TBank",
	"ВТБ", "VTB", "VTB24",
	"Альфа-Банк", "AlfaBank", "Альфа Онлайн",
	"Райффайзен", "Raiffeisen",
	"Яндекс", "Yandex", "Яндекс Музыка", "Yandex Music", "ЯндексБраузер",
	"VK", "VK Messenger", "ВКонтакте",
	"1С", "1cv8",
	"Делимобиль", "Delimobil", "Ситидрайв", "Citydrive",
}

// EffectiveApps merges base with the preset, preserving order and
// deduplicating (first occurrence wins).
func EffectiveApps(base []string, preset bool) []string {
	if !preset {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(RFApps))
	out := make([]string, 0, len(base)+len(RFApps))
	add := func(list []string) {
		for _, a := range list {
			if _, ok := seen[a]; ok {
				continue
			}
			seen[a] = struct{}{}
			out = append(out, a)
		}
	}
	add(base)
	add(RFApps)
	return out
}
```

- [ ] **Step 4: Verify pass** — `go test ./internal/app/` → PASS.
- [ ] **Step 5: Verify live process names** — run `ps -axo comm= | sort -u | grep -iE 'сбер|банк|госусл|яндекс|vk|вк|тинькофф|т-банк|втб|альфа|райф|1с|делимоб|ситидрайв'`; adjust `RFApps` entries to names actually present on this machine (keep absent ones — they match on other machines).

- [ ] **Step 6: Commit**

```bash
git add internal/app/preset.go internal/app/preset_test.go
git commit -m "feat(app): rf-apps preset and effective split-app merging"
```

---

### Task 3: Builder — DNS section, guard, clash_api/cache_file

**Files:**
- Modify: `internal/core/core.go` (Options), `internal/core/singbox/builder.go`
- Test: `internal/core/singbox/builder_test.go`

**Interfaces:**
- Consumes: nothing new.
- Produces: `core.Options.DNSServers []string`, `DNSStrategy string`. Builder renders `dns` from these (fallback: `["1.1.1.1","8.8.8.8"]` / `prefer_ipv4`), adds route rule `{port: 53, action: hijack-dns}` first, and `experimental` block with `clash_api` (127.0.0.1:9090) + `cache_file`.

- [ ] **Step 1: Failing tests**

```go
func TestDNSSectionAndGuard(t *testing.T) {
	var b Builder
	p := testProfile()
	opts := testOpts()
	opts.DNSServers = []string{"9.9.9.9", "https://dns.quad9.net/dns-query"}
	opts.DNSStrategy = "ipv6_only"
	got, err := b.Build(p, opts)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(got, &cfg)
	dns := cfg["dns"].(map[string]any)
	if dns["strategy"] != "ipv6_only" {
		t.Fatalf("strategy: %v", dns["strategy"])
	}
	servers := dns["servers"].([]any)
	first := servers[0].(map[string]any)
	if first["server"] != "9.9.9.9" || first["type"] != "https" || first["detour"] != "proxy" {
		t.Fatalf("first server: %v", first)
	}
	// guard: port-53 hijack is the first route rule
	rules := cfg["route"].(map[string]any)["rules"].([]any)
	r0 := rules[0].(map[string]any)
	if r0["port"] != float64(53) || r0["action"] != "hijack-dns" {
		t.Fatalf("guard rule must be first: %v", r0)
	}
	// second entry is https with URL server
	second := servers[1].(map[string]any)
	if second["server"] != "https://dns.quad9.net/dns-query" {
		t.Fatalf("doh url server: %v", second)
	}
}

func TestExperimentalClashAPI(t *testing.T) {
	var b Builder
	got, err := b.Build(testProfile(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(got, &cfg)
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
	got, err := b.Build(testProfile(), testOpts())
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	json.Unmarshal(got, &cfg)
	servers := cfg["dns"].(map[string]any)["servers"].([]any)
	if len(servers) != 2 {
		t.Fatalf("fallback dns servers: %v", servers)
	}
}
```

(If `testProfile`/`testOpts` helpers have other names in `builder_test.go`, reuse the existing ones — the file defines one profile fixture and one opts fixture; keep names consistent with what exists.)

- [ ] **Step 2: Verify fail** — build error (fields undefined).
- [ ] **Step 3: Implement in builder.go**

Replace the `dns` literal in `Build`:

```go
"dns": map[string]any{
	"servers":  dnsServers(opts),
	"strategy": dnsStrategy(opts),
	"final":    "remote",
},
```

helpers:

```go
// dnsServers renders configured servers; every entry is resolved through
// the proxy (doh) so plain-IP resolvers stay protected inside the tunnel.
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
	return out
}

func dnsStrategy(opts core.Options) string {
	switch opts.DNSStrategy {
	case "prefer_ipv4", "prefer_ipv6", "ipv4_only", "ipv6_only":
		return opts.DNSStrategy
	}
	return "prefer_ipv4"
}
```

Route rules: prepend `map[string]any{"port": 53, "action": "hijack-dns"}` before the existing `protocol: dns` rule (keep both). Experimental block (after `route` in `cfg`):

```go
"experimental": map[string]any{
	"clash_api":  map[string]any{"external_controller": "127.0.0.1:9090", "default_mode": "rule"},
	"cache_file": map[string]any{"enabled": true},
},
```

Add `DNSServers []string` and `DNSStrategy string` to `core.Options`.

- [ ] **Step 4: Verify pass + real sing-box**

Run: `go test ./internal/core/singbox/` → PASS.
Then probe: regenerate the task-3 config via the probe pattern used previously (temp main inside repo, `go run`), run `/opt/homebrew/bin/sing-box check -c /tmp/vpn-dns.json` → exit 0.

- [ ] **Step 5: Commit**

```bash
git add internal/core/core.go internal/core/singbox/builder.go internal/core/singbox/builder_test.go
git commit -m "feat(core): configurable dns with port-53 guard, clash api and cache"
```

---

### Task 4: Builder — split modes with apps

**Files:**
- Modify: `internal/core/core.go`, `internal/core/singbox/builder.go`
- Test: `internal/core/singbox/builder_test.go`

**Interfaces:**
- Consumes: `Options.SplitMode`, new `Options.SplitExcludeApps`, `Options.SplitIncludeApps`.
- Produces: rule layout — mode `exclude`: `{process_name: excludeApps, action: route, outbound: direct}` + `{domain_suffix/ip_cidr: excludeDomains, ...direct}`; mode `include`: `{process_name: includeApps, ...proxy}` + domains→proxy + `final: direct`; mode `off`/empty: neither, `final: proxy`.

- [ ] **Step 1: Failing tests**

```go
func TestSplitExcludeModeWithApps(t *testing.T) {
	var b Builder
	opts := testOpts()
	opts.SplitMode = "exclude"
	opts.SplitExclude = []string{"bank.example"}
	opts.SplitExcludeApps = []string{"Госуслуги"}
	opts.SplitInclude = nil
	got, _ := b.Build(testProfile(), opts)
	var cfg map[string]any
	json.Unmarshal(got, &cfg)
	route := cfg["route"].(map[string]any)
	rules := route["rules"].([]any)
	if route["final"] != "proxy" {
		t.Fatalf("final: %v", route["final"])
	}
	var appRule, domRule bool
	for _, r := range rules {
		m := r.(map[string]any)
		if procs, ok := m["process_name"].([]any); ok {
			if procs[0] == "Госуслуги" && m["outbound"] == "direct" && m["action"] == "route" {
				appRule = true
			}
		}
		if doms, ok := m["domain_suffix"].([]any); ok && doms[0] == "bank.example" && m["outbound"] == "direct" {
			domRule = true
		}
		if m["ip_cidr"] != nil && m["outbound"] == "direct" {
			t.Fatalf("no ip_cidr tokens given but found %v", m)
		}
	}
	if !appRule || !domRule {
		t.Fatalf("app=%v dom=%v rules=%v", appRule, domRule, rules)
	}
}

func TestSplitIncludeModeWithApps(t *testing.T) {
	var b Builder
	opts := testOpts()
	opts.SplitMode = "include"
	opts.SplitInclude = []string{"work.example"}
	opts.SplitIncludeApps = []string{"Miro"}
	got, _ := b.Build(testProfile(), opts)
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
		if doms, ok := m["domain_suffix"].([]any); ok && doms[0] == "work.example" && m["outbound"] == "proxy" {
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
	got, _ := b.Build(testProfile(), opts)
	if strings.Contains(string(got), "process_name") || strings.Contains(string(got), "bank.example") {
		t.Fatalf("off mode must ignore lists:\n%s", got)
	}
}
```

- [ ] **Step 2: Verify fail** (fields undefined).
- [ ] **Step 3: Implement** — rework `splitRule` usage in `Build`:

```go
// inside Build, replacing existing split handling:
switch opts.SplitMode {
case "exclude":
	if len(opts.SplitExcludeApps) > 0 {
		rules = append(rules, map[string]any{
			"process_name": opts.SplitExcludeApps, "action": "route", "outbound": "direct",
		})
	}
	if len(opts.SplitExclude) > 0 {
		rules = append(rules, splitRule(opts.SplitExclude, "direct"))
	}
case "include":
	if len(opts.SplitIncludeApps) > 0 {
		rules = append(rules, map[string]any{
			"process_name": opts.SplitIncludeApps, "action": "route", "outbound": "proxy",
		})
	}
	if len(opts.SplitInclude) > 0 {
		rules = append(rules, splitRule(opts.SplitInclude, "proxy"))
	}
}
```

and `final`: `final := "proxy"; if opts.SplitMode == "include" && (len(opts.SplitInclude) > 0 || len(opts.SplitIncludeApps) > 0) { final = "direct" }`. Empty `SplitMode` keeps legacy behavior (exclude list → direct; include → final flip) — map `""` to the exclude/include dual-list legacy path? No: wiring (Task 8) always passes an explicit mode from defaults; builder treats `""` as `off` is wrong for legacy tests — existing tests `TestSplitTunneling...` pass only lists with empty mode. Decision: treat `""` as the **legacy dual-list behavior** (exclude→direct, include→proxy+final flip) so existing tests stay green; wire always sends explicit mode going forward. Encode that: `mode := opts.SplitMode; if mode == "" { mode = "legacy" }` and add `case "legacy":` doing exactly what the current code does today (copy the existing `splitRule`/final logic unchanged).
- [ ] **Step 4: Verify pass** — `go test ./internal/core/singbox/` → PASS (old split tests green via legacy branch, new tests green).
- [ ] **Step 5: Commit**

```bash
git add internal/core/core.go internal/core/singbox/builder.go internal/core/singbox/builder_test.go
git commit -m "feat(core): split tunneling modes with per-app lists"
```

---

### Task 5: Builder — multiplex + tuic/anytls/ssh outbounds

**Files:**
- Modify: `internal/core/core.go`, `internal/core/singbox/builder.go`
- Test: `internal/core/singbox/builder_test.go`

**Interfaces:**
- Consumes: `Options.Multiplex` (`off|on|auto|""`), new domain protocols from Task 6 (`domain.ProtocolTUIC/AnyTLS/SSH` — defined in Task 6; this task only adds the builder `case` branches, compile lands after Task 6; therefore **Task 5 runs after Task 6** or is compiled together — order: do 6 before 5).
- Produces: mux block on vless/vmess/trojan when `Multiplex != "off"`; outbounds `tuic`, `anytls`, `ssh`.

- [ ] **Step 1: Failing tests**

```go
func TestMultiplexBlock(t *testing.T) {
	var b Builder
	p := testProfile() // vless
	opts := testOpts()
	opts.Multiplex = "on"
	got, _ := b.Build(p, opts)
	var cfg map[string]any
	json.Unmarshal(got, &cfg)
	out := cfg["outbounds"].([]any)
	proxy := out[0].(map[string]any)
	mux, ok := proxy["multiplex"].(map[string]any)
	if !ok || mux["enabled"] != true || mux["protocol"] != "h2mux" {
		t.Fatalf("mux: %v", proxy["multiplex"])
	}
	// off → no mux
	opts.Multiplex = "off"
	got, _ = b.Build(p, opts)
	json.Unmarshal(got, &cfg)
	if cfg["outbounds"].([]any)[0].(map[string]any)["multiplex"] != nil {
		t.Fatal("mux off must not emit multiplex")
	}
}

func TestTUICAnyTLSSSHOutbounds(t *testing.T) {
	cases := []struct {
		proto domain.Protocol
		set   domain.ProtocolSettings
		check func(map[string]any) bool
	}{
		{domain.ProtocolTUIC, domain.ProtocolSettings{UUID: "u", Password: "p", Congestion: "bbr", SNI: "s.example"},
			func(o map[string]any) bool {
				return o["type"] == "tuic" && o["congestion_control"] == "bbr" &&
					o["uuid"] == "u" && o["password"] == "p" &&
					o["tls"].(map[string]any)["server_name"] == "s.example"
			}},
		{domain.ProtocolAnyTLS, domain.ProtocolSettings{Password: "p", SNI: "s.example", Insecure: true},
			func(o map[string]any) bool {
				return o["type"] == "anytls" && o["password"] == "p" &&
					o["tls"].(map[string]any)["insecure"] == true
			}},
		{domain.ProtocolSSH, domain.ProtocolSettings{User: "root", Password: "p"},
			func(o map[string]any) bool {
				return o["type"] == "ssh" && o["user"] == "root" && o["password"] == "p"
			}},
	}
	var b Builder
	for _, tc := range cases {
		p := domain.Profile{Protocol: tc.proto, Endpoint: domain.Endpoint{Host: "h.example", Port: 443}, Settings: tc.set}
		got, err := b.Build(p, testOpts())
		if err != nil {
			t.Fatalf("%s: %v", tc.proto, err)
		}
		var cfg map[string]any
		json.Unmarshal(got, &cfg)
		if !tc.check(cfg["outbounds"].([]any)[0].(map[string]any)) {
			t.Fatalf("%s outbound mismatch:\n%s", tc.proto, got)
		}
	}
}
```

- [ ] **Step 2: Verify fail.**
- [ ] **Step 3: Implement**

`core.Options.Multiplex string`. In `outbound()`, after the base map is populated for vless/vmess/trojan:

```go
// muxFor is applied by callers to vless/vmess/trojan.
func muxFor(opts core.Options) map[string]any {
	if opts.Multiplex == "off" {
		return nil
	}
	return map[string]any{
		"enabled": true, "protocol": "h2mux", "max_streams": 8, "padding": true,
	}
}
```

Wire it: change `outbound(p domain.Profile)` to `outbound(p domain.Profile, opts core.Options)` and inside vless/vmess/trojan branches `if m := muxFor(opts); m != nil { base["multiplex"] = m }`. Update all `outbound(p)` call sites + tests.
New cases:

```go
case domain.ProtocolTUIC:
	if s.UUID == "" || s.Password == "" {
		return nil, fmt.Errorf("tuic: missing uuid/password")
	}
	base["type"] = "tuic"
	base["uuid"] = s.UUID
	base["password"] = s.Password
	if s.Congestion != "" {
		base["congestion_control"] = s.Congestion
	}
	base["tls"] = tlsForcing(s)
case domain.ProtocolAnyTLS:
	if s.Password == "" {
		return nil, fmt.Errorf("anytls: missing password")
	}
	base["type"] = "anytls"
	base["password"] = s.Password
	base["tls"] = tlsForcing(s)
case domain.ProtocolSSH:
	if s.User == "" {
		return nil, fmt.Errorf("ssh: missing user")
	}
	base["type"] = "ssh"
	base["user"] = s.User
	if s.Password != "" {
		base["password"] = s.Password
	}
```

- [ ] **Step 4: Verify** — `go test ./internal/core/singbox/` → PASS. Real sing-box probe: build one config per new protocol (temp probe main like Task 3), `sing-box check -c` each → exit 0. If `check` rejects a field for 1.14, fix the emitted shape and re-check (record final shape in the test).
- [ ] **Step 5: Commit**

```bash
git add internal/core/core.go internal/core/singbox/builder.go internal/core/singbox/builder_test.go
git commit -m "feat(core): multiplex block and tuic/anytls/ssh outbounds"
```

---

### Task 6: Parsers — tuic, anytls, ssh (+ domain enums)

**Files:**
- Modify: `internal/domain/profile.go` (Protocol enum + Valid + `ProtocolSettings.User/Congestion` fields)
- Create: `internal/subscription/uri/tuic.go`, `internal/subscription/uri/anytls.go`, `internal/subscription/uri/ssh.go`
- Modify: `internal/subscription/uri/uri.go` (registry)
- Test: `internal/subscription/uri/uri_test.go` (+ per-file tests)

**Interfaces:**
- Produces: `domain.ProtocolTUIC = "tuic"`, `ProtocolAnyTLS = "anytls"`, `ProtocolSSH = "ssh"`; `ProtocolSettings.User string`, `ProtocolSettings.Congestion string`; `uri.Parse` accepts `tuic://`, `anytls://`, `ssh://`.

- [ ] **Step 1: Failing tests**

```go
func TestParseTUIC(t *testing.T) {
	p, err := Parse("tuic://uuid-x:pass-w@vpn.example:8443?congestion_control=bbr&sni=s.example&alpn=h3#TUIC%20node")
	if err != nil {
		t.Fatal(err)
	}
	if p.Protocol != domain.ProtocolTUIC || p.Settings.UUID != "uuid-x" || p.Settings.Password != "pass-w" ||
		p.Settings.Congestion != "bbr" || p.Settings.SNI != "s.example" ||
		p.Endpoint.Host != "vpn.example" || p.Endpoint.Port != 8443 || p.Name != "TUIC node" {
		t.Fatalf("parsed: %+v", p)
	}
}

func TestParseAnyTLSSSH(t *testing.T) {
	p, err := Parse("anytls://pass-w@vpn.example:443?sni=s.example&insecure=1#any")
	if err != nil || p.Protocol != domain.ProtocolAnyTLS || p.Settings.Password != "pass-w" || !p.Settings.Insecure {
		t.Fatalf("anytls: %+v err=%v", p, err)
	}
	s, err := Parse("ssh://root@vpn.example:22?password=secret#box")
	if err != nil || s.Protocol != domain.ProtocolSSH || s.Settings.User != "root" || s.Settings.Password != "secret" || s.Endpoint.Port != 22 {
		t.Fatalf("ssh: %+v err=%v", s, err)
	}
}
```

- [ ] **Step 2: Verify fail.**
- [ ] **Step 3: Implement** — follow the existing per-scheme file pattern (`trojan.go` is the closest template: `url.Parse`, query params into `ProtocolSettings`, `#fragment` name). `tuic.go`:

```go
package uri

import (
	"fmt"
	"net/url"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

func ParseTUIC(raw string) (domain.Profile, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return domain.Profile{}, fmt.Errorf("%w: %v", domain.ErrParse, err)
	}
	if u.User == nil {
		return domain.Profile{}, fmt.Errorf("%w: tuic: missing user:password", domain.ErrParse)
	}
	host, port, err := hostPort(u)
	if err != nil {
		return domain.Profile{}, err
	}
	p := domain.Profile{
		Protocol: domain.ProtocolTUIC,
		Name:     fragmentName(u),
		Endpoint: domain.Endpoint{Host: host, Port: port},
		Settings: domain.ProtocolSettings{
			UUID:       u.User.Username(),
			Password:   passOf(u.User),
			Congestion: u.Query().Get("congestion_control"),
			SNI:        u.Query().Get("sni"),
			ALPN:       u.Query().Get("alpn"),
		},
	}
	if err := p.Validate(); err != nil {
		return domain.Profile{}, err
	}
	return p, nil
}
```

(`hostPort`, `fragmentName`, `passOf` — reuse the helpers already present in `uri.go`/`trojan.go`; check their exact names before writing and reuse them.) `anytls.go` mirrors trojan (password, `sni`, `insecure`); `ssh.go` reads `u.User.Username()` into `Settings.User` and `password` query into `Settings.Password`.
Registry: add entries to the `parsers` map + `Schemes()` list. Domain: add constants + `Valid()` cases + fields `User`, `Congestion` in `ProtocolSettings` (json tags `user`, `congestion`).
- [ ] **Step 4: Verify pass** — `go test ./internal/subscription/... ./internal/domain/...` → PASS.
- [ ] **Step 5: Commit**

```bash
git add internal/domain/profile.go internal/subscription/uri/
git commit -m "feat(uri): tuic, anytls and ssh parsers"
```

---

### Task 7: IPv6 kill-switch ranges

**Files:**
- Modify: `internal/platform/killswitch_darwin.go`
- Test: `internal/platform/killswitch_test.go`

- [ ] **Step 1: Failing test** — extend `TestKillSwitchRulesContent` wants with `"fc00::/7", "fe80::/7"`.
- [ ] **Step 2: Verify fail.**
- [ ] **Step 3: Implement** — add both to the `pass to { ... }` set in `KillSwitchRules()`.
- [ ] **Step 4: Verify pass** — `go test ./internal/platform/`.
- [ ] **Step 5: Commit**

```bash
git add internal/platform/killswitch_darwin.go internal/platform/killswitch_test.go
git commit -m "fix(platform): pass ipv6 ula and link-local through kill switch"
```

---

### Task 8: Wiring — config → options (split mode, apps, dns, mux, preset)

**Files:**
- Modify: `internal/app/connection.go` (`options()`), `internal/cli/tui.go` (`buildSettingsInfo`)
- Test: existing suites keep passing.

**Interfaces:**
- Consumes: `app.EffectiveApps` (Task 2), config fields (Task 1).
- Produces: `core.Options` fully populated from config:

```go
func (s *ConnectionService) options() core.Options {
	return core.Options{
		// ...existing fields...
		SocialBlock:  s.cfg.Features.SocialBlock,
		AppFirewall:  s.cfg.Features.AppFirewall,
		SplitMode:    s.cfg.Features.SplitMode,
		SplitExclude: s.cfg.Features.SplitExclude,
		SplitInclude: s.cfg.Features.SplitInclude,
		SplitExcludeApps: app.EffectiveApps(s.cfg.Features.SplitExcludeApps, s.cfg.Features.PresetApps),
		SplitIncludeApps: s.cfg.Features.SplitIncludeApps,
		Multiplex:    s.cfg.Features.Multiplex,
		DNSServers:   s.cfg.DNS.Servers,
		DNSStrategy:  s.cfg.DNS.Strategy,
	}
}
```

- [ ] **Step 1:** implement; [ ] **Step 2:** `go build ./... && go test ./...` → all green; [ ] **Step 3: Commit**

```bash
git add internal/app/connection.go internal/cli/tui.go
git commit -m "feat(app): wire split modes, preset, dns and multiplex into core options"
```

(`buildSettingsInfo` gains `SplitMode`, `PresetApps`, `Multiplex`, DNS fields into `shared.SettingsInfo` — needed by Task 11; add fields here, screen uses them later.)

---

### Task 9: Restart + mux fallback probe

**Files:**
- Modify: `internal/app/connection.go` (add `Restart`)
- Create: `internal/app/muxprobe.go`
- Test: `internal/app/muxprobe_test.go`

**Interfaces:**
- Produces: `func (s *ConnectionService) Restart(ctx context.Context) (core.RunInfo, error)` — resolves active profile, `engine.Stop(old)` → `engine.Start(new)` → replace state file (kill switch: disable+enable around the restart to refresh endpoint IPs). `func WatchMux(ctx context.Context, logFile string, onFallback func()) ` — goroutine: tail the log for 10 s, regex `(?i)(multiplex|mux).*(failed|error|closed|refused)|unknown transport`; one hit → call `onFallback` once.

- [ ] **Step 1: Failing test**

```go
func TestWatchMuxFallback(t *testing.T) {
	dir := t.TempDir()
	log := dir + "/sb.log"
	os.WriteFile(log, []byte(""), 0o600)
	hit := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go WatchMux(ctx, log, func() { hit <- struct{}{} })
	time.Sleep(100 * time.Millisecond)
	f, _ := os.OpenFile(log, os.O_APPEND|os.O_WRONLY, 0o600)
	f.WriteString("ERROR router: multiplex: handshake failed: connection refused\n")
	f.Close()
	select {
	case <-hit:
	case <-time.After(3 * time.Second):
		t.Fatal("fallback not triggered")
	}
	// benign line must not trigger
	os.WriteFile(log, []byte(""), 0o600)
	go WatchMux(ctx, log, func() { t.Fatal("false positive") })
	time.Sleep(100 * time.Millisecond)
	f2, _ := os.OpenFile(log, os.O_APPEND|os.O_WRONLY, 0o600)
	f2.WriteString("INFO inbound/mixed: connection from 127.0.0.1\n")
	f2.Close()
	time.Sleep(500 * time.Millisecond)
}
```

- [ ] **Step 2: Verify fail** → [ ] **Step 3: Implement `muxprobe.go`** — read from file offset 0, poll 200 ms until ctx done or `time.Now().Sub(start) > 10*time.Second`, `regexp.MustCompile` on each appended chunk, `sync.Once`-style trigger. `Restart` in connection.go:

```go
// Restart stops the running core and starts it again with the current
// options (mux fallback, network change). Sequential by design: two
// sing-box processes cannot share the TUN device.
func (s *ConnectionService) Restart(ctx context.Context) (core.RunInfo, error) {
	st, err := s.store.LoadState()
	if err != nil || st == nil {
		return core.RunInfo{}, domain.ErrNotRunning
	}
	if err := s.Down(ctx); err != nil {
		return core.RunInfo{}, err
	}
	info, err := s.Up(ctx, st.ProfileID)
	if err != nil {
		return core.RunInfo{}, err
	}
	return info, nil
}
```

(`Up` re-reads config? No — `s.cfg` is stale after settings edits; Restart must reload: add `s.cfg, _ = config.Load(s.cfgPath, nil)` — ConnectionService needs cfgPath; it has `paths`; add field via constructor if absent, else reload in `cmd/vpn` by rebuilding the service — check constructor wiring and pick the minimal honest option; document choice in the commit.)
- [ ] **Step 4: Verify pass** — `go test ./internal/app/`.
- [ ] **Step 5: Commit**

```bash
git add internal/app/connection.go internal/app/muxprobe.go internal/app/muxprobe_test.go
git commit -m "feat(app): core restart and mux fallback log probe"
```

---

### Task 10: Network watchdog

**Files:**
- Create: `internal/platform/netwatch_darwin.go`, `internal/platform/netwatch_other.go`
- Create: `internal/app/netwatch.go`
- Test: `internal/app/netwatch_test.go`

**Interfaces:**
- Produces: `func DefaultIface() (string, error)` — parses `route -n get default` output (`interface: en0` line); `func WatchNetwork(ctx context.Context, pollEvery time.Duration, onChange func())` (app layer, injectable poller: `type IfacePoller func() (string, error)`).
- [ ] **Step 1: Failing test** — fake poller returning `en0` then `en11`; `WatchNetwork` with 10 ms interval calls `onChange` exactly once (debounce).
- [ ] **Step 2: Verify fail** → [ ] **Step 3: Implement** — app/netwatch.go:

```go
package app

import (
	"context"
	"time"
)

// IfacePoller returns the current default-route interface name.
type IfacePoller func() (string, error)

// WatchNetwork polls the default interface and fires onChange (at most
// once per change) until ctx is done.
func WatchNetwork(ctx context.Context, poll IfacePoller, every time.Duration, onChange func()) {
	t := time.NewTicker(every)
	defer t.Stop()
	cur, err := poll()
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			next, err := poll()
			if err != nil {
				continue
			}
			if next != cur {
				cur = next
				onChange()
			}
		}
	}
}
```

platform/netwatch_darwin.go: `DefaultIface()` via `exec.Command("route", "-n", "get", "default")`, regex `interface:\s+(\S+)`. `netwatch_other.go`: parse `/proc/net/route` (linux) or return error otherwise.
- [ ] **Step 4: Verify pass** → [ ] **Step 5: Commit**

```bash
git add internal/platform/netwatch_darwin.go internal/platform/netwatch_other.go internal/app/netwatch.go internal/app/netwatch_test.go
git commit -m "feat(app): network watchdog for default-route changes"
```

(Watchdog-to-Restart hookup is wired in Task 11's connection flow: `Up` starts `WatchNetwork` + `WatchMux` goroutines with `context.WithoutCancel(ctx)`; `Down` cancels them — include this in Task 11 Step 3 if not landed here.)

---

### Task 11: SettingsAPI v2 + full TUI settings editing (+ mux/watchdog hookup)

**Files:**
- Modify: `internal/tui/shared/deps.go` (SettingsInfo fields: SplitMode, SplitExcludeApps, SplitIncludeApps, PresetApps, Multiplex, DNSServers, DNSStrategy; SettingsAPI: `SetSplitMode(string) error`, `SetSplitApps(which string, apps []string) error`, `SetPresetApps(bool) error`, `SetMultiplex(string) error`, `SetDNSServers([]string) error`, `SetDNSstrategy(string) error`, `SetTUNEnabled(bool) error`, `SetMTU(int) error`, `SetStack(string) error`, `SetAutoRoute(bool) error`, `SetStrictRoute(bool) error`, `SetMixedPort(int) error`, `SetLogLevel(string) error`)
- Modify: `internal/cli/tui.go` (adapter), `internal/tui/fakes/settings.go`, `internal/tui/screen/settings/settings.go`
- Modify: `internal/app/connection.go` — `Up` starts `WatchNetwork` + (if mux auto) `WatchMux` with `context.WithoutCancel`; `Down` cancels; mux fallback callback = `s.Restart` with mux forced off (reload config, set `Multiplex=off` for this run only, in-memory override + toast/log line).
- Test: `internal/tui/screen/settings/settings_test.go`, `internal/tui/fakes` compile.

**Interfaces:**
- Consumes: everything from Tasks 1–10.
- Produces: Settings screen sections Core/Network/DNS/Split/Firewall/App, all editable; mux auto fallback wired.

- [ ] **Step 1: Failing screen tests**

```go
func TestSplitModeCycles(t *testing.T) {
	m, api := newModel(nil)
	m = downToLabel(t, m, func(r row) bool { return r.kind == kindSplitMode })
	ns, _ := m.Update(tea.KeyPressMsg{Code: ' '})
	_ = ns
	if len(api.SplitModeCalls) != 1 || api.SplitModeCalls[0] != "include" {
		t.Fatalf("mode calls: %v", api.SplitModeCalls)
	}
}

func TestEditSplitApps(t *testing.T) {
	m, api := newModel(nil)
	m = downToLabel(t, m, func(r row) bool { return r.kind == kindAppExclude })
	ns, _ := m.Update(kf('\r'))
	mm := ns.(*Model)
	if !mm.input.Showing() {
		t.Fatal("must open input")
	}
	ns, _ = mm.Update(kf('X'))
	ns.Update(kf('\r'))
	if len(api.SplitAppsCalls) != 1 || api.SplitAppsCalls[0].Which != "exclude" {
		t.Fatalf("calls: %v", api.SplitAppsCalls)
	}
}
```

(fakes gain `SplitModeCalls []string`, `SplitAppsCalls []struct{ Which string; Apps []string }`, etc. — every new SettingsAPI method gets a recorded call; extend `testInfo()` with the new fields.)
- [ ] **Step 2: Verify fail.**
- [ ] **Step 3: Implement** — screen `rowKind` gains `kindSplitMode, kindAppExclude, kindAppInclude, kindPreset, kindMux, kindDNSServers, kindDNSstrategy, kindTUN, kindMTU, kindStack, kindAutoRoute, kindStrictRoute, kindMixedPort, kindLogLevel`; `activate()` maps each to the right Set* (cycles: mode кроме→только→выкл, stack system→gvisor→mixed, log level debug→info→warn→error, strategy prefer_ipv4→prefer_ipv6→ipv4_only→ipv6_only, mux auto→on→off); numeric editors (MTU, mixed port) validate with `strconv.Atoi` and stay open on error with a toast. Sections reorganized; keep labels ≤ 14 chars; run `TestFitsMinimalTerminal` (it opens every page at 60×20).
- [ ] **Step 4: Verify pass** — `go test ./internal/tui/...` all green.
- [ ] **Step 5: Real pty smoke** — expect rig (isolated HOME, 60×20): open Settings, cycle split mode, confirm config file on disk changed (`features.split_mode: include`), esc, quit.
- [ ] **Step 6: Commit**

```bash
git add -A internal/tui internal/cli/tui.go internal/app/connection.go
git commit -m "feat(tui): full settings editing, split modes, preset, dns, mux fallback wiring"
```

---

### Task 12: Clipboard + import (file / clipboard)

**Files:**
- Create: `internal/platform/clipboard.go` (`ReadClipboard() (string, error)`, `WriteClipboard(string) error` — pbpaste/pbcopy, xclip/wl-paste/wl-copy fallbacks, `clipboard_other.go` for unsupported → error)
- Modify: `internal/cli/profile.go` (`vpn profile add --file/--clipboard` flags)
- Modify: `internal/tui/screen/profiles/profiles.go` (`i`, `p` keys → input opens, value from clipboard for `p`)
- Test: `internal/platform/clipboard_test.go` (WriteClipboard→ReadClipboard round-trip skipped when no binary: skip-if-missing pattern `exec.LookPath("pbpaste")`), profiles screen test with injectable `readClipboard func() (string, error)`.

- [ ] **Step 1: Failing tests** (round-trip on macOS where pbcopy exists; TUI: `p` fills input with fake clipboard value, `enter` calls Profiles.Add).
- [ ] **Step 2: Verify fail** → [ ] **Step 3: Implement** → [ ] **Step 4: Verify pass** (`go test ./internal/platform/ ./internal/tui/...`) → [ ] **Step 5: Commit**

```bash
git add internal/platform/clipboard*.go internal/cli/profile.go internal/tui/screen/profiles/
git commit -m "feat(profile): import from file and clipboard, platform clipboard access"
```

---

### Task 13: URI serializer (export core)

**Files:**
- Create: `internal/subscription/uri/export.go` (`func ToURI(p domain.Profile) (string, error)` — all 8 protocols)
- Test: `internal/subscription/uri/export_test.go`

**Interfaces:**
- Produces: `ToURI` — round-trip guarantee `Parse(ToURI(p))` equals input on Endpoint/Settings/Protocol/Name.

- [ ] **Step 1: Failing test** — table over 8 protocol fixtures: build profile, `ToURI`, `Parse`, compare fields (`Endpoint`, `Protocol`, `Name`, key `Settings` fields per protocol).
- [ ] **Step 2: Verify fail** → [ ] **Step 3: Implement** (vless/trojan/hy2/tuic/anytls/ssh: `url.URL{Scheme, User, Host, RawQuery, Fragment}` with `url.Values`; ss: `ss://` + base64(`method:password`) + `@host:port#name`; vmess: `vmess://` + base64 JSON `{"v":"2","ps":name,"add":host,"port":port,"id":uuid,"aid":0,"scy":cipher,"net":transport,"tls":security,"sni":sni,"path":path}`) → [ ] **Step 4: Verify pass** → [ ] **Step 5: Commit**

```bash
git add internal/subscription/uri/export.go internal/subscription/uri/export_test.go
git commit -m "feat(uri): profile to share-uri serializer for all protocols"
```

---

### Task 14: Export commands + TUI copy keys

**Files:**
- Modify: `internal/cli/profile.go`, `internal/cli/sub.go` (`vpn profile export <id|name> [--clipboard] [--all] [--out f]`, `vpn sub export [--all] [--out f]`)
- Modify: `internal/app/profile_service.go` (`ExportURI(idOrName) (string, error)` via `uri.ToURI`), `internal/tui/screen/profiles/profiles.go` (`y`, `Y` keys + toasts), `internal/tui/screen/subscriptions/subscriptions.go` (`y` key)
- Test: CLI export to temp file (`--out`) asserts one URI per line and parses back; TUI `y` calls injectable `writeClipboard` and toasts.

- [ ] **Step 1: Failing tests** → [ ] **Step 2: fail** → [ ] **Step 3: implement** → [ ] **Step 4: pass** → [ ] **Step 5: Commit**

```bash
git add internal/cli/profile.go internal/cli/sub.go internal/app/profile_service.go internal/tui/screen/profiles/ internal/tui/screen/subscriptions/
git commit -m "feat(export): share profiles via clipboard, file and cli commands"
```

---

### Task 15: Stats service + Other stats/log viewer

**Files:**
- Create: `internal/app/stats.go` (`StatsService{base string, client}`, `func (s *StatsService) Totals(ctx) (Traffic, error)` — GET `/connections`, fields `uploadTotal/downloadTotal/connections count`; `Rates(ctx) (up, down int64, err error)` — GET `/traffic` one-shot)
- Create: `internal/app/logtail.go` (`func Tail(path string, n int) []string`, `func CountSince(path, pattern string, since time.Time) int`)
- Modify: `internal/core/singbox/builder.go` — nothing (Task 3 added experimental)
- Modify: `internal/tui/screen/other/other.go` (Stats section rows + `l` toggles log tail view, refresh via `shared.ScheduleTick`)
- Test: `internal/app/stats_test.go` (`httptest.Server` serving fixture JSON), `internal/app/logtail_test.go` (temp file, injected lines), other screen test asserting stats rows render from a fake.

- [ ] **Step 1: Failing tests** → [ ] **Step 2: fail** → [ ] **Step 3: implement** (Other page polls stats only while open; log view caps at 200 lines, ANSI-stripped) → [ ] **Step 4: pass + real smoke: `go run ./cmd/vpn`-built binary connects (probe profile) and `/connections` answers** → [ ] **Step 5: Commit**

```bash
git add internal/app/stats.go internal/app/stats_test.go internal/app/logtail.go internal/app/logtail_test.go internal/tui/screen/other/
git commit -m "feat(other): traffic stats, blocked counter and log viewer"
```

---

### Task 16: Parser benchmark

**Files:**
- Create: `internal/subscription/uri/bench_test.go`
- Modify: `internal/cli/bench.go` (new `vpn bench` command), `internal/cli/root.go` (register)

- [ ] **Step 1: bench tests**

```go
package uri

import (
	"fmt"
	"strings"
	"testing"
)

func benchURIs(n int) []string {
	tmpls := []string{
		"vless://%s@h%d.example:443?security=tls&sni=s.example&type=ws&path=/w#n%d",
		"vmess://%s", // filled below with base64 json
		"trojan://%s@h%d.example:443?sni=s.example#n%d",
		"ss://YWVzLTI1Ni1nY206cGFzcw==@h%d.example:8388#n%d",
		"hy2://%s@h%d.example:443?sni=s.example#n%d",
		"tuic://%s:pw@h%d.example:8443?congestion_control=bbr#n%d",
		"anytls://%s@h%d.example:443#n%d",
		"ssh://root@h%d.example:22?password=pw#n%d",
	}
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		id := fmt.Sprintf("11111111-2222-3333-4444-55555555%04x", i)
		switch {
		case i%8 == 1:
			out = append(out, vmessBench(id, i))
		default:
			out = append(out, fmt.Sprintf(tmpls[i%8], id, i, i))
		}
	}
	return out
}

func BenchmarkParseBatch10k(b *testing.B) {
	uris := benchURIs(10000)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, u := range uris {
			if _, err := Parse(u); err != nil {
				b.Fatal(err)
			}
		}
	}
}
```

(`vmessBench` builds the base64 JSON fixture; keep it in the file.)
- [ ] **Step 2: run** `go test -bench BenchmarkParseBatch10k -benchmem ./internal/subscription/uri/` → record ns/op, allocs.
- [ ] **Step 3: `vpn bench` command** — same synthetic batch, `time.Since` around the loop, prints `parsed 10000 configs in Xs (N conf/s, Y ns/conf)`. [ ] **Step 4: run `./vpn bench`, record output.** [ ] **Step 5: Commit**

```bash
git add internal/subscription/uri/bench_test.go internal/cli/bench.go internal/cli/root.go
git commit -m "feat(bench): uri parser benchmark and vpn bench command"
```

---

### Task 17: README, final verification, worktree checks

- [ ] **Step 1:** README: new config keys sample (dns, split_mode, split apps, preset_apps, multiplex), TUI key rows (`i`, `p`, `y`, `Y`, `l`, split-mode cycling), honest limits section unchanged + mux/window note.
- [ ] **Step 2:** `gofmt -l .`, `go vet ./...`, `go test ./...`, `go build -o ./vpn ./cmd/vpn`, `./vpn bench`.
- [ ] **Step 3:** pty smoke at 60×20: menu → Settings (edit MTU → disk check) → Other (stats rows render) → quit; `TestFitsMinimalTerminal` green.
- [ ] **Step 4:** `git status` clean; every commit worktree-verified; final report with benchmark numbers and honest limits.
- [ ] **Step 5: Commit**

```bash
git add README.md
git commit -m "docs(readme): split modes, dns, multiplex, import/export, stats"
```
