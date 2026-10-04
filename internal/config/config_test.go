package config

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

func TestDefaults(t *testing.T) {
	c, err := Load("", nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Core.Default != "sing-box" || c.Core.SingBox.MinVers != "1.14.0" {
		t.Fatalf("defaults: %+v", c.Core)
	}
	if !c.TUN.Enabled || c.MixedPort != 10808 {
		t.Fatalf("defaults: %+v", c)
	}
}

func TestOverrides(t *testing.T) {
	c, err := Load("", map[string]any{"core.default": "xray", "log.level": "debug"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Core.Default != "xray" || c.Log.Level != "debug" {
		t.Fatalf("overrides: %+v", c)
	}
}

func TestInvalidLevel(t *testing.T) {
	if _, err := Load("", map[string]any{"log.level": "verbose"}); !errors.Is(err, domain.ErrInvalidConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}

func TestFeaturesRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	src := `
features:
  adblock: true
  trackerblock: false
  split_exclude:
    - bank.example
    - 192.168.0.0/16
  split_include:
    - "10.0.0.0/8"
update:
  auto: true
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Features.Adblock || c.Features.TrackerBlock {
		t.Fatalf("features flags: %+v", c.Features)
	}
	if len(c.Features.SplitExclude) != 2 || c.Features.SplitExclude[0] != "bank.example" {
		t.Fatalf("split_exclude: %+v", c.Features.SplitExclude)
	}
	if !c.Update.Auto {
		t.Fatalf("update: %+v", c.Update)
	}
	// mutate and save; reload must match
	c.Features.TrackerBlock = true
	c.Update.Auto = false
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	c2, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c2.Features.TrackerBlock || c2.Update.Auto {
		t.Fatalf("roundtrip: %+v", c2)
	}
}

func TestFeaturesValidation(t *testing.T) {
	dir := t.TempDir()
	bad := dir + "/bad.yaml"
	src := "features:\n  split_include:\n    - \"not a cidr/99999\"\n"
	if err := os.WriteFile(bad, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad, nil); !errors.Is(err, domain.ErrInvalidConfig) {
		t.Fatalf("expected invalid config for bad cidr, got %v", err)
	}
	empty := dir + "/empty.yaml"
	if err := os.WriteFile(empty, []byte("features:\n  split_include:\n    - \"  \"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(empty, nil); !errors.Is(err, domain.ErrInvalidConfig) {
		t.Fatalf("expected invalid config for empty token, got %v", err)
	}
}

func TestSavePreservesKeysAndUnknowns(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	src := `
mixed_port: 9999
custom_unknown_key: hello
tun:
  auto_route: false
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.MixedPort != 9999 || c.TUN.AutoRoute {
		t.Fatalf("pre-save load: %+v", c)
	}
	c.Features.Adblock = true
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"mixed_port: 9999", "custom_unknown_key: hello", "auto_route: false", "adblock: true"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("saved config missing %q:\n%s", want, raw)
		}
	}
	// and viper still reads everything back correctly
	c2, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if c2.MixedPort != 9999 || c2.TUN.AutoRoute || !c2.Features.Adblock {
		t.Fatalf("post-save load: %+v", c2)
	}
}

func TestNewFeaturesRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	src := `
features:
  socialblock: true
  kill_switch: true
  appfirewall:
    - torrent-client
    - Steam
`
	if err := os.WriteFile(path, []byte(src), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Features.SocialBlock || !c.Features.KillSwitch {
		t.Fatalf("flags: %+v", c.Features)
	}
	if len(c.Features.AppFirewall) != 2 || c.Features.AppFirewall[0] != "torrent-client" {
		t.Fatalf("appfirewall: %+v", c.Features.AppFirewall)
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{"socialblock: true", "kill_switch: true", "torrent-client"} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("saved config missing %q:\n%s", want, raw)
		}
	}
}

func TestKillSwitchNeedsTUN(t *testing.T) {
	c := Defaults()
	c.Features.KillSwitch = true
	c.TUN.Enabled = false
	if err := c.Validate(); !errors.Is(err, domain.ErrInvalidConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}
	c.TUN.Enabled = true
	if err := c.Validate(); err != nil {
		t.Fatalf("tun on must satisfy kill switch: %v", err)
	}
}

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
  split_exclude_apps:
    - Госуслуги
  split_include_apps:
    - Miro
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
		c.Features.SplitMode != "include" || c.Features.PresetApps ||
		len(c.Features.SplitExcludeApps) != 1 || c.Features.Multiplex != "off" {
		t.Fatalf("load: dns=%+v features=%+v", c.DNS, c.Features)
	}
	if err := Save(path, c); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(path)
	for _, want := range []string{"split_mode: include", "split_exclude_apps:", "Госуслуги", "preset_apps: false", `multiplex: "off"`, "https://dns.quad9.net/dns-query"} {
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

func TestAppFirewallValidation(t *testing.T) {
	dir := t.TempDir()
	bad := dir + "/bad.yaml"
	if err := os.WriteFile(bad, []byte("features:\n  appfirewall:\n    - \"bad name\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(bad, nil); !errors.Is(err, domain.ErrInvalidConfig) {
		t.Fatalf("expected invalid config, got %v", err)
	}
}
