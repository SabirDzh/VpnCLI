package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/SabirDzh/VpnCLI/internal/config"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/core/singbox"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/platform"
	"github.com/SabirDzh/VpnCLI/internal/tui"
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
)

// NewTUICmd opens the interactive terminal UI.
func NewTUICmd(d Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Open the interactive terminal UI",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("TUI требует интерактивный терминал")
			}
			ctx, stop := signal.NotifyContext(cmd.Context(),
				syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			return tui.Run(ctx, shared.Deps{
				Connection:  connAdapter{svc: d.Conn},
				Profiles:    profileAdapter{svc: d.Profile, store: d.Store},
				Subs:        subAdapter{svc: d.Sub},
				Settings:    buildSettingsInfo(d.Config, d.Paths),
				SettingsAPI: settingsAPI{svc: d.SettingsSvc, cfgPath: d.ConfigPath, paths: d.Paths},
				Update:      updateAPI{svc: d.Update},
				Stats:       d.StatsSvc,
				Version:     d.Version,
				Repo:        d.Repo,
				ReadOnly:    !platform.IsPrivileged(),
			})
		},
	}
}

// connAdapter narrows ConnectionService to shared.ConnectionAPI.
type connAdapter struct{ svc *app.ConnectionService }

func (a connAdapter) Status(ctx context.Context) (app.StatusView, error) {
	return a.svc.Status(ctx)
}

func (a connAdapter) Up(ctx context.Context, ref string) error {
	_, err := a.svc.Up(ctx, ref)
	return err
}

func (a connAdapter) Down(ctx context.Context) error { return a.svc.Down(ctx) }

// profileAdapter narrows ProfileService (+store for Active) to shared.ProfileAPI.
type profileAdapter struct {
	svc   *app.ProfileService
	store profileActiveStore
}

type profileActiveStore interface {
	ActiveProfile() (domain.Profile, error)
}

func (a profileAdapter) List() ([]domain.Profile, error) { return a.svc.List() }

func (a profileAdapter) Add(uri string) (domain.Profile, error) {
	return a.svc.AddFromURI(uri)
}

func (a profileAdapter) AddRaw(name string, proto domain.Protocol, data []byte) (domain.Profile, error) {
	return a.svc.AddRaw(name, proto, data)
}

func (a profileAdapter) Use(id string) (domain.Profile, error) { return a.svc.Use(id) }

func (a profileAdapter) Remove(id string) error { return a.svc.Remove(id) }

func (a profileAdapter) Edit(idOrName, name, uri string) (domain.Profile, error) {
	return a.svc.Edit(idOrName, name, uri)
}

func (a profileAdapter) Active() (domain.Profile, error) {
	return a.store.ActiveProfile()
}

// buildSettingsInfo snapshots the effective config and core state.
func buildSettingsInfo(cfg config.Config, paths platform.Paths) shared.SettingsInfo {
	info := shared.SettingsInfo{
		CoreDefault:      cfg.Core.Default,
		SingBoxPath:      cfg.Core.SingBox.Path,
		MinVersion:       cfg.Core.SingBox.MinVers,
		LogLevel:         cfg.Log.Level,
		TUNEnabled:       cfg.TUN.Enabled,
		MTU:              cfg.TUN.MTU,
		AutoRoute:        cfg.TUN.AutoRoute,
		StrictRoute:      cfg.TUN.StrictRoute,
		MixedPort:        cfg.MixedPort,
		Privileged:       platform.IsPrivileged(),
		ConfigDir:        paths.ConfigDir,
		DataDir:          paths.DataDir,
		StateFile:        paths.StateFile,
		LogFile:          paths.LogFile,
		Adblock:          cfg.Features.Adblock,
		TrackerBlock:     cfg.Features.TrackerBlock,
		SocialBlock:      cfg.Features.SocialBlock,
		KillSwitch:       cfg.Features.KillSwitch,
		SplitMode:        cfg.Features.SplitMode,
		SplitExclude:     cfg.Features.SplitExclude,
		SplitInclude:     cfg.Features.SplitInclude,
		SplitExcludeApps: cfg.Features.SplitExcludeApps,
		SplitIncludeApps: cfg.Features.SplitIncludeApps,
		PresetApps:       cfg.Features.PresetApps,
		AppFirewall:      cfg.Features.AppFirewall,
		Multiplex:        cfg.Features.Multiplex,
		DNSServers:       cfg.DNS.Servers,
		DNSStrategy:      cfg.DNS.Strategy,
		Stack:            cfg.TUN.Stack,
		AutoUpdate:       cfg.Update.Auto,
	}
	if bin, err := singbox.FindBinary(info.SingBoxPath); err != nil {
		info.SingBoxErr = "not found in PATH"
	} else if ver, err := singbox.BinaryVersion(bin); err != nil {
		info.SingBoxErr = "version check failed"
	} else {
		info.SingBoxVersion = ver
	}
	return info
}

// settingsAPI narrows the config-file service to shared.SettingsAPI.
type settingsAPI struct {
	svc     *app.SettingsService
	cfgPath string
	paths   platform.Paths
}

func (a settingsAPI) Snapshot() shared.SettingsInfo {
	cfg, err := config.Load(a.cfgPath, nil)
	if err != nil {
		return shared.SettingsInfo{}
	}
	return buildSettingsInfo(cfg, a.paths)
}

func (a settingsAPI) SetBlocklist(kind string, on bool) error {
	switch kind {
	case "ads", "trackers", "social":
	default:
		return fmt.Errorf("unknown blocklist kind %q", kind)
	}
	_, err := a.svc.Update(func(c *config.Config) {
		switch kind {
		case "ads":
			c.Features.Adblock = on
		case "trackers":
			c.Features.TrackerBlock = on
		case "social":
			c.Features.SocialBlock = on
		}
	})
	return err
}

func (a settingsAPI) SetAppFirewall(apps []string) error {
	_, err := a.svc.Update(func(c *config.Config) { c.Features.AppFirewall = apps })
	return err
}

func (a settingsAPI) SetKillSwitch(on bool) error {
	_, err := a.svc.Update(func(c *config.Config) { c.Features.KillSwitch = on })
	return err
}

func (a settingsAPI) SetSplitMode(mode string) error {
	_, err := a.svc.Update(func(c *config.Config) { c.Features.SplitMode = mode })
	return err
}

func (a settingsAPI) SetSplitApps(which string, apps []string) error {
	_, err := a.svc.Update(func(c *config.Config) {
		switch which {
		case "exclude":
			c.Features.SplitExcludeApps = apps
		case "include":
			c.Features.SplitIncludeApps = apps
		}
	})
	return err
}

func (a settingsAPI) SetPresetApps(on bool) error {
	_, err := a.svc.Update(func(c *config.Config) { c.Features.PresetApps = on })
	return err
}

func (a settingsAPI) SetMultiplex(mode string) error {
	_, err := a.svc.Update(func(c *config.Config) { c.Features.Multiplex = mode })
	return err
}

func (a settingsAPI) SetDNSServers(servers []string) error {
	_, err := a.svc.Update(func(c *config.Config) { c.DNS.Servers = servers })
	return err
}

func (a settingsAPI) SetDNSstrategy(strategy string) error {
	_, err := a.svc.Update(func(c *config.Config) { c.DNS.Strategy = strategy })
	return err
}

func (a settingsAPI) SetTUNEnabled(on bool) error {
	_, err := a.svc.Update(func(c *config.Config) { c.TUN.Enabled = on })
	return err
}

func (a settingsAPI) SetMTU(mtu int) error {
	_, err := a.svc.Update(func(c *config.Config) { c.TUN.MTU = mtu })
	return err
}

func (a settingsAPI) SetStack(stack string) error {
	_, err := a.svc.Update(func(c *config.Config) { c.TUN.Stack = stack })
	return err
}

func (a settingsAPI) SetAutoRoute(on bool) error {
	_, err := a.svc.Update(func(c *config.Config) { c.TUN.AutoRoute = on })
	return err
}

func (a settingsAPI) SetStrictRoute(on bool) error {
	_, err := a.svc.Update(func(c *config.Config) { c.TUN.StrictRoute = on })
	return err
}

func (a settingsAPI) SetMixedPort(port int) error {
	_, err := a.svc.Update(func(c *config.Config) { c.MixedPort = port })
	return err
}

func (a settingsAPI) SetLogLevel(level string) error {
	_, err := a.svc.Update(func(c *config.Config) { c.Log.Level = level })
	return err
}

func (a settingsAPI) SetSplit(exclude, include []string) error {
	_, err := a.svc.Update(func(c *config.Config) {
		c.Features.SplitExclude = exclude
		c.Features.SplitInclude = include
	})
	return err
}

func (a settingsAPI) SetAutoUpdate(on bool) error {
	_, err := a.svc.Update(func(c *config.Config) { c.Update.Auto = on })
	return err
}

// updateAPI narrows UpdateService to shared.UpdateAPI.
type updateAPI struct{ svc *app.UpdateService }

func (a updateAPI) Check(ctx context.Context) (app.CheckResult, error) {
	return a.svc.Check(ctx)
}

func (a updateAPI) Update(ctx context.Context, tag string) error {
	return a.svc.Update(ctx, tag)
}

func (a updateAPI) LastCheck() time.Time   { return a.svc.LastCheck() }
func (a updateAPI) LastUpdated() time.Time { return a.svc.LastUpdated() }

// subAdapter narrows SubscriptionService to shared.SubscriptionAPI.
type subAdapter struct{ svc *app.SubscriptionService }

func (a subAdapter) List() ([]domain.Subscription, error) { return a.svc.List() }

func (a subAdapter) Add(name, url string) (domain.Subscription, error) {
	return a.svc.Add(name, url)
}

func (a subAdapter) Update(id string) (int, error) { return a.svc.Update(id) }

func (a subAdapter) Remove(id string) error { return a.svc.Remove(id) }

func (a subAdapter) Edit(idOrName, name, url string) (domain.Subscription, error) {
	return a.svc.Edit(idOrName, name, url)
}
