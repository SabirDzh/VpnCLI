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
