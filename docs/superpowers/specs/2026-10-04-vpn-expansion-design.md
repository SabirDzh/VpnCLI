# VpnCLI expansion: routing v2, protocols, mux, resilience, stats, export

Date: 2026-10-04
Status: approved (user «согласен, приступай»)

## Overview

Ten workstreams extending VpnCLI. Approved decisions:
- «Кроме/только» are split-tunneling modes over **apps and domains**; firewall
  stays a separate full-network-block (reject) list.
- RF-apps preset: wide list, enabled by default, lives in the «кроме» list.
- Stats source: clash_api + log parsing.

## 1. Split tunneling v2 (modes + apps)

Config (`features`):
- `split_mode: exclude|include|off` — default `exclude` («кроме»).
- `split_exclude`, `split_include` — existing domain/CIDR lists (unchanged).
- New `split_exclude_apps`, `split_include_apps` — process names, validated like
  appfirewall entries (no spaces, no `/`).
- New `preset_apps: true` (default) — merges the built-in RF list into the
  effective exclude-apps list. Preset is a code constant, toggle only.

Builder (exclude mode): `{process_name: effApps, action: route, outbound: direct}`
prepended; domain rules unchanged. Include mode: `{process_name: incApps,
action: route, outbound: proxy}` + existing domain rule; `final: direct` when
include is active. `split_mode` gates which list applies (off = neither).

Preset (built-in constant, process names as macOS shows them, refined during
implementation against live process lists): Госуслуги, СберБанк/СберБанк Онлайн,
Т-Банк/Тинькофф, ВТБ, Альфа-Банк, Райффайзен, Яндекс (browser/music/tracker),
VK (messenger/vk), 1С, Делимобиль, Ситидрайв.

TUI Settings: rows `split mode` (enter cycles кроме→только→выкл),
`apps кроме`, `apps только` (comma editors), `preset РФ` toggle.

CLI: profile export covers sharing (§10); no new split CLI in this scope.

## 2. DNS settings + guard

Config: new top-level `dns` section: `servers: [1.1.1.1, 8.8.8.8]` (max 3,
host or doh URL), `strategy: prefer_ipv4|prefer_ipv6|ipv4_only|ipv6_only`
(default `prefer_ipv4`). Editable in TUI (DNS section rows).

Builder: remote DNS server = first entry via `https` type with `detour: proxy`
(a plain-IP entry stays `udp` type? No — always doh to protect inside tunnel;
if the entry is a URL, use its type). Guard rules:
- `{port: 53, action: hijack-dns}` — catches plaintext DNS to any resolver
  routed into the tunnel (replaces the `protocol: dns` matcher, which misses
  fixed-resolver clients).
- `{protocol: dns, action: hijack-dns}` kept for non-port-53 DNS (DoT/DoQ
  still goes to final).
- `dns.strategy` set from config.

Verification: real sing-box, mixed inbound, dig via socks5h against an
external resolver → response must come from the remote resolver through the
proxy (log shows hijack-dns).

## 3. IPv6

- Kill switch rules gain `pass to { ..., fc00::/7, fe80::/7 }` (v6 ULA +
  link-local alongside the v4 private ranges).
- `dns.strategy` covers v4/v6 preference.
- TUN already carries `fdfe:dcba:9876::1/126`; route `final` handles v6 as-is.
- Honest limit: v6 egress works only where the server/route provides it.

## 4. Protocols: tuic, anytls, ssh

New parsers registered in `subscription/uri` registry:
- `tuic://uuid:password@host:port?congestion_control=bbr&sni=&alpn=` → ProtocolTUIC.
- `anytls://password@host:port?sni=&insecure=` → ProtocolAnyTLS.
- `ssh://user@host:port?password=` → ProtocolSSH (password auth).

Domain: `ProtocolTUIC/AnyTLS/SSH` + validation; ProtocolSettings gains
`Congestion`, and reuses UUID/Password/SNI/Insecure/ALPN.

Builder outbounds: `tuic` (v5: uuid+password, congestion_control, tls with
alpn/sni), `anytls` (password, tls), `ssh` (user, password; private-key
auth deferred — no URI standard). Tests: parse round-trip + `sing-box check`
against the real binary for each built config.

## 5. Multiplex: off|on|auto

Config: `features.multiplex: off|on|auto` (default `auto`). Applies to
vless/vmess/trojan (sing-box mux is TCP-transport only; hysteria2 excluded).

- on/auto: outbound gains `multiplex: {enabled, protocol: h2mux, max_streams: 8,
  padding: true}`.
- auto fallback: supervisor marks a fresh run as "probing" when mux is on;
  a log watcher looks for mux handshake failures (`dial.*multiplex|mux.*closed|unknown
  transport` patterns — pinned against a real failure log during implementation)
  within the first 10 s. On hit: build config without mux, **overlapped restart**
  (start new process, wait for its startup log line, then stop the old one).
  Downtime window < 1 s; kill switch endpoint pass rule stays loaded, so
  the tunnel is the only thing reconnecting.

Verification: unit test for the watcher (injected log lines) + overlapped
restart unit test with a fake engine; live run against a mux-less server is
manual (no such server in test env) — watcher proven by feeding the exact
failure text captured from a real sing-box mux failure.

## 6. Network watchdog

`platform` (darwin): subscribe to PF_ROUTE (RTM_IFINFO/RTM_NEWADDR) via
syscall; debounce 2 s; on default-route/interface change and running core →
`supervisor.Restart` (same overlapped restart as §5) + re-resolve endpoint and
refresh the kill switch anchor (server IPs may differ per network).

Non-darwin: polling `route get default` fallback every 10 s (same interface,
best-effort).

Tests: interface of the watcher is injectable (fake event source); restart
path covered by §5 tests. Live proof: toggle Wi-Fi manually — documented as a
manual check (cannot automate network state changes safely).

## 7. Full settings editing + import

SettingsAPI gains: SetTUNEnabled, SetMTU, SetStack, SetAutoRoute,
SetStrictRoute, SetMixedPort, SetLogLevel, SetDNSServers, SetDNSstrategy,
SetSplitMode, SetSplitApps(exclude|include, apps), SetPresetApps.
Validation lives in config.Validate (MTU range, stack enum, port range,
kill-switch↔TUN coupling already enforced; enabling TUN off with kill switch
on must fail with a clear error).

TUI Settings sections rebuilt: Core / Network (tun, mtu, stack, auto/strict
route) / DNS (servers, strategy) / Split (mode, 4 lists, preset) / Firewall
(appfirewall) / App (log level, privileged, paths).

Import:
- CLI: `vpn profile add --file path` (reads first URI/line), `--clipboard`.
- TUI Profiles: `i` → path input; `p` → clipboard paste (platform.ReadClipboard:
  pbpaste / xclip / wl-paste / powershell).

## 8. Other: stats + log viewer

Builder adds `experimental: {clash_api: {external_controller:
127.0.0.1:9090, default_mode: rule}, cache_file: {enabled: true}}` when not
present in Raw configs (Raw passes through untouched, honest limit for
imported configs).

New `StatsService`: GET /connections → uploadTotal, downloadTotal, active
count; GET /traffic sampled 1/s for live rates while the Other page is open.
Blocked counter: count reject log lines (`router: match.*reject|rejected`) in
the log file since process start.

Other page: new `Stats` section (totals, live speed, connections, blocked);
key `l` toggles an in-page log tail (last N lines, refresh 1/s, esc closes).

## 9. Parser benchmark

- `subscription/uri/bench_test.go`: `BenchmarkParseURI` per scheme +
  `BenchmarkParseBatch10k` (b.ReportAllocs, ns/op → derives conf/s).
- `vpn bench`: builds a synthetic 10k URI batch (round-robin over the 8
  schemes with varying params), times full parse, prints conf/s and allocs.
  Reported in the final summary with real numbers.

## 10. Export / share

- TUI Profiles: `y` → copy selected profile URI to clipboard; `Y` → all URIs.
  Subscriptions: `y` → copy sub URL.
- CLI: `vpn profile export <id|name>` prints URI; flags `--clipboard`,
  `--all`, `--out file` (one URI per line). `vpn sub export [--all] [--out file]`
  prints `name url` lines.
- File format = one URI per line, directly re-importable via
  `vpn profile add --file`.
- Toast feedback in TUI; clipboard write via platform.WriteClipboard (pbcopy /
  xclip / wl-copy / powershell clip).

## Honest limits

- Mux auto = fallback restart, not zero-loss handoff (sing-box has no
  hot-reload); window < 1 s under kill switch protection.
- In-video (YouTube) ads remain unblockable at DNS/route level.
- ShadowTLS needs an outbound chain — deferred.
- Raw imported configs bypass builder-managed features (they are passed
  through verbatim).

## Testing strategy

TDD per phase; real sing-box `check` + targeted live probes (DNS hijack,
mux failure text, process rules); expect-pty TUI checks for new screens;
gofmt/vet/full suite before every commit; worktree build-verify per commit.
