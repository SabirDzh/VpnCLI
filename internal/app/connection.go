package app

import (
	"context"
	"fmt"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/config"
	"github.com/SabirDzh/VpnCLI/internal/core"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/platform"
	"github.com/SabirDzh/VpnCLI/internal/storage"
)

// ConnectionService owns engine lifecycle: up/down/status.
type ConnectionService struct {
	store    *storage.Store
	registry *core.Registry
	cfg      config.Config
	paths    platform.Paths
}

// NewConnectionService creates the service.
func NewConnectionService(store *storage.Store, reg *core.Registry, cfg config.Config, paths platform.Paths) *ConnectionService {
	return &ConnectionService{store: store, registry: reg, cfg: cfg, paths: paths}
}

func (s *ConnectionService) options() core.Options {
	return core.Options{
		TUNEnabled:       s.cfg.TUN.Enabled,
		MTU:              s.cfg.TUN.MTU,
		Stack:            s.cfg.TUN.Stack,
		AutoRoute:        s.cfg.TUN.AutoRoute,
		StrictRoute:      s.cfg.TUN.StrictRoute,
		MixedEnabled:     true,
		MixedPort:        s.cfg.MixedPort,
		LogLevel:         s.cfg.Log.Level,
		Adblock:          s.cfg.Features.Adblock,
		TrackerBlock:     s.cfg.Features.TrackerBlock,
		SocialBlock:      s.cfg.Features.SocialBlock,
		AppFirewall:      s.cfg.Features.AppFirewall,
		SplitMode:        s.cfg.Features.SplitMode,
		SplitExclude:     s.cfg.Features.SplitExclude,
		SplitInclude:     s.cfg.Features.SplitInclude,
		SplitExcludeApps: EffectiveApps(s.cfg.Features.SplitExcludeApps, s.cfg.Features.PresetApps),
		SplitIncludeApps: s.cfg.Features.SplitIncludeApps,
		Multiplex:        s.cfg.Features.Multiplex,
		DNSServers:       s.cfg.DNS.Servers,
		DNSStrategy:      s.cfg.DNS.Strategy,
	}
}

// Up starts the VPN for the given profile (or the active one).
func (s *ConnectionService) Up(ctx context.Context, profileRef string) (core.RunInfo, error) {
	if !platform.IsPrivileged() {
		return core.RunInfo{}, domain.ErrNotPrivileged
	}
	if st, _ := s.store.LoadState(); st != nil && platform.Alive(st.PID) {
		return core.RunInfo{}, fmt.Errorf("%w (pid %d)", domain.ErrAlreadyRunning, st.PID)
	}
	var (
		p   domain.Profile
		err error
	)
	if profileRef != "" {
		p, err = s.store.GetProfile(profileRef)
	} else {
		p, err = s.store.ActiveProfile()
	}
	if err != nil {
		return core.RunInfo{}, err
	}

	engine, err := s.registry.Select(p, s.cfg.Core.Default)
	if err != nil {
		return core.RunInfo{}, err
	}
	info, err := engine.Start(ctx, core.StartRequest{
		Profile:    p,
		Options:    s.options(),
		RuntimeDir: s.paths.RuntimeDir,
		LogFile:    s.paths.LogFile,
	})
	if err != nil {
		return core.RunInfo{}, err
	}
	_ = s.store.SaveState(storage.State{
		Core:       engine.Name(),
		ProfileID:  p.ID,
		PID:        info.PID,
		StartedAt:  time.Now(),
		ConfigPath: info.ConfigPath,
	})
	// Remember selection so `status` and subsequent `up` resolve it.
	_, _ = s.store.SetActiveProfile(p.ID)
	if s.cfg.Features.KillSwitch {
		// Fail closed: without the pf anchor the tunnel would run
		// unprotected, so a failed enable aborts the whole Up.
		if err := platform.EnableKillSwitch(p.Endpoint.Host, int(p.Endpoint.Port)); err != nil {
			_ = engine.Stop(ctx, core.RunInfo{PID: info.PID, ConfigPath: info.ConfigPath, Core: engine.Name()})
			_ = s.store.ClearState()
			return core.RunInfo{}, fmt.Errorf("kill switch: %w", err)
		}
	}
	return info, nil
}

// Down stops the running VPN and clears state.
func (s *ConnectionService) Down(ctx context.Context) error {
	st, err := s.store.LoadState()
	if err != nil {
		return err
	}
	if st == nil {
		return domain.ErrNotRunning
	}
	alive, permitted := platform.Signallable(st.PID)
	if !alive {
		_ = s.store.ClearState()
		return domain.ErrNotRunning
	}
	if !permitted && !platform.IsPrivileged() {
		// The daemon belongs to root (started via sudo); a user-space
		// down can neither signal it nor report honestly otherwise.
		return domain.ErrNotPrivileged
	}
	engine, err := s.registry.Get(st.Core)
	if err != nil {
		// Unknown core: still clear stale state.
		_ = s.store.ClearState()
		return err
	}
	if err := engine.Stop(ctx, core.RunInfo{PID: st.PID, ConfigPath: st.ConfigPath, Core: st.Core}); err != nil {
		return err
	}
	if s.cfg.Features.KillSwitch {
		// Best effort: state is already cleared, and a leftover anchor
		// can be flushed by the next Down or manually.
		_ = platform.DisableKillSwitch()
	}
	return s.store.ClearState()
}

// Status describes the current connection. The active profile is always
// resolved, so callers know what Up would dial even while disconnected.
func (s *ConnectionService) Status(ctx context.Context) (StatusView, error) {
	st, err := s.store.LoadState()
	if err != nil {
		return StatusView{}, err
	}
	view := StatusView{}
	if active, err := s.store.ActiveProfile(); err == nil {
		view.ProfileID = active.ID
		view.ProfileName = active.Name
		view.Endpoint = fmt.Sprintf("%s:%d", active.Endpoint.Host, active.Endpoint.Port)
	}
	if st == nil {
		return view, nil
	}
	if !platform.Alive(st.PID) {
		// Stale pid file (e.g. reboot or kill): clean up and report stopped.
		_ = s.store.ClearState()
		view.StalePID = st.PID
		return view, nil
	}
	view.Running = true
	view.Core = st.Core
	view.PID = st.PID
	view.Since = st.StartedAt
	if st.ProfileID != "" {
		view.ProfileID = st.ProfileID
		if p, err := s.store.GetProfile(st.ProfileID); err == nil {
			view.ProfileName = p.Name
			view.Endpoint = fmt.Sprintf("%s:%d", p.Endpoint.Host, p.Endpoint.Port)
		}
	}
	return view, nil
}

// StatusView is a CLI-friendly status snapshot.
type StatusView struct {
	Running     bool
	Core        string
	ProfileID   string
	ProfileName string
	Endpoint    string
	PID         int
	Since       time.Time
	StalePID    int
}
