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
	// DNS — серверы и стратегия для builder'а (guard через туннель).
	DNS struct {
		Servers  []string `mapstructure:"servers" yaml:"servers"`
		Strategy string   `mapstructure:"strategy" yaml:"strategy"`
	} `mapstructure:"dns" yaml:"dns"`
	// Features — TUI-managed routing extras: DNS-level blocklists and
	// split tunneling lists (domains or CIDRs), applied on next connect.
	Features struct {
		Adblock      bool     `mapstructure:"adblock" yaml:"adblock"`
		TrackerBlock bool     `mapstructure:"trackerblock" yaml:"trackerblock"`
		SocialBlock  bool     `mapstructure:"socialblock" yaml:"socialblock"`
		KillSwitch   bool     `mapstructure:"kill_switch" yaml:"kill_switch"`
		SplitMode    string   `mapstructure:"split_mode" yaml:"split_mode"`
		SplitExclude []string `mapstructure:"split_exclude" yaml:"split_exclude"`
		SplitInclude []string `mapstructure:"split_include" yaml:"split_include"`
		// Списки приложений для режимов «кроме»/«только» (имена процессов).
		SplitExcludeApps []string `mapstructure:"split_exclude_apps" yaml:"split_exclude_apps"`
		SplitIncludeApps []string `mapstructure:"split_include_apps" yaml:"split_include_apps"`
		// PresetApps включает встроенный список РФ-приложений в «кроме».
		PresetApps  bool     `mapstructure:"preset_apps" yaml:"preset_apps"`
		AppFirewall []string `mapstructure:"appfirewall" yaml:"appfirewall"`
		// Multiplex: off|on|auto (auto = fallback-рестарт без mux).
		Multiplex string `mapstructure:"multiplex" yaml:"multiplex"`
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
	c.DNS.Servers = []string{"1.1.1.1", "8.8.8.8"}
	c.DNS.Strategy = "prefer_ipv4"
	c.Features.SplitMode = "exclude"
	c.Features.PresetApps = true
	c.Features.Multiplex = "auto"
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
	v.SetDefault("dns.servers", def.DNS.Servers)
	v.SetDefault("dns.strategy", def.DNS.Strategy)
	v.SetDefault("features.split_mode", def.Features.SplitMode)
	v.SetDefault("features.preset_apps", def.Features.PresetApps)
	v.SetDefault("features.multiplex", def.Features.Multiplex)

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
	// The pf kill switch passes only the utun interface; without TUN it
	// would sever sing-box's own uplink too.
	if c.Features.KillSwitch && !c.TUN.Enabled {
		return fmt.Errorf("%w: kill_switch requires tun.enabled", domain.ErrInvalidConfig)
	}
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
	for _, list := range [][]string{c.Features.SplitExclude, c.Features.SplitInclude} {
		for _, tok := range list {
			if err := validateSplitToken(tok); err != nil {
				return err
			}
		}
	}
	for _, app := range c.Features.AppFirewall {
		if app == "" || strings.ContainsAny(app, " \t/") {
			return fmt.Errorf("%w: bad app name %q", domain.ErrInvalidConfig, app)
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
	set("dns.servers", c.DNS.Servers)
	set("dns.strategy", c.DNS.Strategy)
	set("features.adblock", c.Features.Adblock)
	set("features.trackerblock", c.Features.TrackerBlock)
	set("features.socialblock", c.Features.SocialBlock)
	set("features.kill_switch", c.Features.KillSwitch)
	set("features.split_mode", c.Features.SplitMode)
	set("features.split_exclude_apps", c.Features.SplitExcludeApps)
	set("features.split_include_apps", c.Features.SplitIncludeApps)
	set("features.preset_apps", c.Features.PresetApps)
	set("features.appfirewall", c.Features.AppFirewall)
	set("features.multiplex", c.Features.Multiplex)
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
