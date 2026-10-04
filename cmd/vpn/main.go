// Command vpn — composition root: flags, config, deps wiring, error mapping.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/cli"
	"github.com/SabirDzh/VpnCLI/internal/config"
	"github.com/SabirDzh/VpnCLI/internal/core"
	"github.com/SabirDzh/VpnCLI/internal/core/singbox"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/logging"
	"github.com/SabirDzh/VpnCLI/internal/platform"
	"github.com/SabirDzh/VpnCLI/internal/storage"
)

// version is overridden by goreleaser ldflags.
var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(exitCode(err))
	}
}

func run() error {
	var cfgFile, coreName, logLevel, singboxPath string

	// Pre-parse only global flags via a throwaway command so config is
	// available before building the real tree.
	pre := &cobra.Command{Use: "vpn"}
	pre.Flags().StringVar(&cfgFile, "config", "", "config file path")
	pre.Flags().StringVar(&coreName, "core", "", "core override (default from config)")
	pre.Flags().StringVar(&logLevel, "log-level", "", "log level override")
	pre.Flags().StringVar(&singboxPath, "singbox-path", "", "sing-box binary override")
	pre.FParseErrWhitelist = cobra.FParseErrWhitelist{UnknownFlags: true}
	_ = pre.ParseFlags(os.Args[1:])

	overrides := map[string]any{}
	if coreName != "" {
		overrides["core.default"] = coreName
	}
	if logLevel != "" {
		overrides["log.level"] = logLevel
	}
	if singboxPath != "" {
		overrides["core.singbox.path"] = singboxPath
	}

	cfg, err := config.Load(cfgFile, overrides)
	if err != nil {
		return err
	}
	log := logging.New(os.Stderr, cfg.Log.Level)

	paths, err := platform.Resolve()
	if err != nil {
		return err
	}
	if err := paths.Ensure(); err != nil {
		return err
	}
	store, err := storage.New(paths.DataDir, paths.StateFile)
	if err != nil {
		return err
	}

	reg := core.NewRegistry()
	reg.Register("sing-box", singbox.New, core.Settings{
		BinaryPath: cfg.Core.SingBox.Path,
		MinVersion: cfg.Core.SingBox.MinVers,
	})

	profileSvc := app.NewProfileService(store)
	subSvc := app.NewSubscriptionService(store, store)
	connSvc := app.NewConnectionService(store, reg, cfg, paths)
	updateSvc := app.NewUpdateService("SabirDzh/VpnCLI", version, paths.DataDir)
	configPath := cfgFile
	if configPath == "" {
		configPath = filepath.Join(paths.ConfigDir, "config.yaml")
	}
	connSvc.SetConfigPath(configPath)
	settingsSvc := app.NewSettingsService(configPath)
	statsSvc := app.NewStatsService("127.0.0.1:9090", paths.LogFile)

	root := cli.NewRootCmd(cli.Deps{
		Config:      cfg,
		ConfigPath:  configPath,
		Paths:       paths,
		Store:       store,
		Reg:         reg,
		Profile:     profileSvc,
		Sub:         subSvc,
		Conn:        connSvc,
		Update:      updateSvc,
		SettingsSvc: settingsSvc,
		StatsSvc:    statsSvc,
		Version:     version,
		Repo:        "SabirDzh/VpnCLI",
	})
	// Re-attach persistent flags to the real tree.
	root.PersistentFlags().StringVar(&cfgFile, "config", cfgFile, "config file path")
	root.PersistentFlags().StringVar(&coreName, "core", coreName, "core override")
	root.PersistentFlags().StringVar(&logLevel, "log-level", logLevel, "log level override")
	root.PersistentFlags().StringVar(&singboxPath, "singbox-path", singboxPath, "sing-box binary override")
	root.SetOut(os.Stdout)
	root.SetErr(os.Stderr)

	log.Debug("starting", "core", cfg.Core.Default)
	if cfg.Update.Auto {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			if v, err := updateSvc.AutoCheck(ctx); err == nil && v != "" {
				log.Info("vpn auto-updated, restart to apply", "version", v)
			}
		}()
	}
	return root.Execute()
}

func exitCode(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotPrivileged),
		errors.Is(err, domain.ErrAlreadyRunning):
		return 2
	case errors.Is(err, domain.ErrNotRunning),
		errors.Is(err, domain.ErrNoActiveProfile),
		errors.Is(err, domain.ErrProfileNotFound),
		errors.Is(err, domain.ErrSubscriptionNotFound),
		errors.Is(err, domain.ErrCoreNotFound),
		errors.Is(err, domain.ErrUnsupportedProtocol):
		return 3
	default:
		return 1
	}
}
