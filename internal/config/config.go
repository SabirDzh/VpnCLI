// Package config isolates viper: the rest of the code receives a typed
// Config struct and never touches viper directly.
package config

import (
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// Config — типизированная конфигурация приложения.
type Config struct {
	Core struct {
		Default string `mapstructure:"default"`
		SingBox struct {
			Path    string `mapstructure:"path"`
			MinVers string `mapstructure:"min_version"`
		} `mapstructure:"singbox"`
	} `mapstructure:"core"`
	Log struct {
		Level string `mapstructure:"level"`
	} `mapstructure:"log"`
	TUN struct {
		Enabled     bool   `mapstructure:"enabled"`
		Name        string `mapstructure:"name"`
		MTU         int    `mapstructure:"mtu"`
		Stack       string `mapstructure:"stack"`
		AutoRoute   bool   `mapstructure:"auto_route"`
		StrictRoute bool   `mapstructure:"strict_route"`
	} `mapstructure:"tun"`
	MixedPort int `mapstructure:"mixed_port"`
	// Features — TUI-managed routing extras: DNS-level blocklists and
	// split tunneling lists (domains or CIDRs), applied on next connect.
	Features struct {
		Adblock      bool     `mapstructure:"adblock" yaml:"adblock"`
		TrackerBlock bool     `mapstructure:"trackerblock" yaml:"trackerblock"`
		SplitExclude []string `mapstructure:"split_exclude" yaml:"split_exclude"`
		SplitInclude []string `mapstructure:"split_include" yaml:"split_include"`
	} `mapstructure:"features" yaml:"features"`
	// Update controls self-update behavior; auto applies on next launch.
	Update struct {
		Auto bool `mapstructure:"auto" yaml:"auto"`
	} `mapstructure:"update" yaml:"update"`
}

// Defaults returns the default configuration.
func Defaults() Config {
	var c Config
	c.Core.Default = "sing-box"
	c.Core.SingBox.MinVers = "1.14.0"
	c.Log.Level = "info"
	c.TUN.Enabled = true
	c.TUN.MTU = 9000
	c.TUN.Stack = "system"
	c.TUN.AutoRoute = true
	c.TUN.StrictRoute = true
	c.MixedPort = 10808
	return c
}

// Load builds Config with priority: flags map → env (VPN_*) → file → defaults.
// cfgFile may be empty (use default lookup). overrides carries bound flag values.
func Load(cfgFile string, overrides map[string]any) (Config, error) {
	v := viper.New()
	def := Defaults()

	v.SetDefault("core.default", def.Core.Default)
	v.SetDefault("core.singbox.path", def.Core.SingBox.Path)
	v.SetDefault("core.singbox.min_version", def.Core.SingBox.MinVers)
	v.SetDefault("log.level", def.Log.Level)
	v.SetDefault("tun.enabled", def.TUN.Enabled)
	v.SetDefault("tun.mtu", def.TUN.MTU)
	v.SetDefault("tun.stack", def.TUN.Stack)
	v.SetDefault("tun.auto_route", def.TUN.AutoRoute)
	v.SetDefault("tun.strict_route", def.TUN.StrictRoute)
	v.SetDefault("mixed_port", def.MixedPort)

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.AddConfigPath("$HOME/.config/vpn")
		v.AddConfigPath("/etc/vpn")
	}

	v.SetEnvPrefix("VPN")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	for k, val := range overrides {
		if val != nil {
			v.Set(k, val)
		}
	}

	_ = v.ReadInConfig() // missing file is fine, defaults apply

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := c.Validate(); err != nil {
		return Config{}, err
	}
	return c, nil
}

// Validate checks user-facing constraints.
func (c Config) Validate() error {
	switch strings.ToLower(c.Log.Level) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("%w: unknown log.level %q", domain.ErrInvalidConfig, c.Log.Level)
	}
	if c.Core.Default == "" {
		return fmt.Errorf("%w: core.default is empty", domain.ErrInvalidConfig)
	}
	if c.TUN.MTU < 1280 || c.TUN.MTU > 9000 {
		return fmt.Errorf("%w: tun.mtu out of range", domain.ErrInvalidConfig)
	}
	for _, list := range [][]string{c.Features.SplitExclude, c.Features.SplitInclude} {
		for _, tok := range list {
			if err := validateSplitToken(tok); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateSplitToken accepts a domain suffix or an IP CIDR prefix.
func validateSplitToken(tok string) error {
	t := strings.TrimSpace(tok)
	if t == "" || strings.ContainsAny(t, " \t") {
		return fmt.Errorf("%w: empty split token", domain.ErrInvalidConfig)
	}
	if strings.Contains(t, "/") {
		if _, err := netip.ParsePrefix(t); err != nil {
			return fmt.Errorf("%w: bad cidr %q", domain.ErrInvalidConfig, t)
		}
		return nil
	}
	if strings.HasPrefix(t, ".") || strings.Contains(t, "/") {
		return fmt.Errorf("%w: bad domain %q", domain.ErrInvalidConfig, t)
	}
	return nil
}

// Save writes the config back to path. Unknown keys already present in
// the file are preserved, and canonical viper key names are written so a
// later Load sees every value.
func Save(path string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("%w: config path is empty", domain.ErrInvalidConfig)
	}
	v := viper.New()
	v.SetConfigFile(path)
	_ = v.ReadInConfig() // missing file is fine: unknown keys survive below
	set := func(key string, val any) {
		if val != nil {
			v.Set(key, val)
		}
	}
	set("core.default", c.Core.Default)
	set("core.singbox.path", c.Core.SingBox.Path)
	set("core.singbox.min_version", c.Core.SingBox.MinVers)
	set("log.level", c.Log.Level)
	set("tun.enabled", c.TUN.Enabled)
	set("tun.name", c.TUN.Name)
	set("tun.mtu", c.TUN.MTU)
	set("tun.stack", c.TUN.Stack)
	set("tun.auto_route", c.TUN.AutoRoute)
	set("tun.strict_route", c.TUN.StrictRoute)
	set("mixed_port", c.MixedPort)
	set("features.adblock", c.Features.Adblock)
	set("features.trackerblock", c.Features.TrackerBlock)
	set("features.split_exclude", c.Features.SplitExclude)
	set("features.split_include", c.Features.SplitInclude)
	set("update.auto", c.Update.Auto)
	ext := filepath.Ext(path)
	tmp := strings.TrimSuffix(path, ext) + ".tmp" + ext
	if err := v.WriteConfigAs(tmp); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return os.Rename(tmp, path)
}
