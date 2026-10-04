// Package cli builds cobra commands. Commands parse flags and call app
// services; there is no business logic here.
package cli

import (
	"github.com/spf13/cobra"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/config"
	"github.com/SabirDzh/VpnCLI/internal/core"
	"github.com/SabirDzh/VpnCLI/internal/platform"
	"github.com/SabirDzh/VpnCLI/internal/storage"
)

// Deps are wired in main.go (composition root).
type Deps struct {
	Config      config.Config
	ConfigPath  string
	Paths       platform.Paths
	Store       *storage.Store
	Reg         *core.Registry
	Profile     *app.ProfileService
	Sub         *app.SubscriptionService
	Conn        *app.ConnectionService
	Update      *app.UpdateService
	SettingsSvc *app.SettingsService
	StatsSvc    *app.StatsService
	Version     string
	Repo        string
}

// NewRootCmd assembles all commands.
func NewRootCmd(d Deps) *cobra.Command {
	root := &cobra.Command{
		Use:   "vpn",
		Short: "CLI VPN client (sing-box core)",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true, // main() prints errors with exit codes
	}
	root.AddCommand(
		NewUpCmd(d),
		NewDownCmd(d),
		NewStatusCmd(d),
		NewProfileCmd(d),
		NewSubCmd(d),
		NewUpdateCmd(d),
		NewTUICmd(d),
		NewVersionCmd(d),
	)
	return root
}
