package config

import (
	"errors"
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
