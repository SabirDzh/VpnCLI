package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"

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
				Connection: connAdapter{svc: d.Conn},
				Profiles:   profileAdapter{svc: d.Profile, store: d.Store},
				Subs:       subAdapter{svc: d.Sub},
				Settings:   buildSettingsInfo(d),
				ReadOnly:   !platform.IsPrivileged(),
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

func (a profileAdapter) Use(id string) (domain.Profile, error) { return a.svc.Use(id) }

func (a profileAdapter) Remove(id string) error { return a.svc.Remove(id) }

func (a profileAdapter) Active() (domain.Profile, error) {
	return a.store.ActiveProfile()
}

// buildSettingsInfo snapshots the effective config and core state.
func buildSettingsInfo(d Deps) shared.SettingsInfo {
	info := shared.SettingsInfo{
		CoreDefault: d.Config.Core.Default,
		SingBoxPath: d.Config.Core.SingBox.Path,
		MinVersion:  d.Config.Core.SingBox.MinVers,
		LogLevel:    d.Config.Log.Level,
		TUNEnabled:  d.Config.TUN.Enabled,
		MTU:         d.Config.TUN.MTU,
		AutoRoute:   d.Config.TUN.AutoRoute,
		StrictRoute: d.Config.TUN.StrictRoute,
		MixedPort:   d.Config.MixedPort,
		Privileged:  platform.IsPrivileged(),
		ConfigDir:   d.Paths.ConfigDir,
		DataDir:     d.Paths.DataDir,
		StateFile:   d.Paths.StateFile,
		LogFile:     d.Paths.LogFile,
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

// subAdapter narrows SubscriptionService to shared.SubscriptionAPI.
type subAdapter struct{ svc *app.SubscriptionService }

func (a subAdapter) List() ([]domain.Subscription, error) { return a.svc.List() }

func (a subAdapter) Add(name, url string) (domain.Subscription, error) {
	return a.svc.Add(name, url)
}

func (a subAdapter) Update(id string) (int, error) { return a.svc.Update(id) }

func (a subAdapter) Remove(id string) error { return a.svc.Remove(id) }
