package singbox

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/core"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/platform"
	"github.com/SabirDzh/VpnCLI/internal/process"
)

// Core runs sing-box as an external process via process.Supervisor.
type Core struct {
	settings core.Settings
	builder  Builder
}

// New creates a sing-box Core from settings.
func New(s core.Settings) (core.Core, error) {
	if s.MinVersion == "" {
		s.MinVersion = MinVersion
	}
	return &Core{settings: s}, nil
}

// Name implements core.Core.
func (c *Core) Name() string { return "sing-box" }

// Supports implements core.Core (MVP protocols; Raw passes through).
func (c *Core) Supports(p domain.Profile) bool {
	if len(p.Raw) > 0 {
		return true
	}
	return p.Protocol.Valid()
}

// Start implements core.Core.
func (c *Core) Start(ctx context.Context, req core.StartRequest) (core.RunInfo, error) {
	cfg, err := c.builder.Build(req.Profile, req.Options)
	if err != nil {
		return core.RunInfo{}, err
	}
	bin, err := FindBinary(c.settings.BinaryPath)
	if err != nil {
		return core.RunInfo{}, err
	}
	ver, err := BinaryVersion(bin)
	if err != nil {
		return core.RunInfo{}, err
	}
	if err := CheckMinVersion(ver, c.settings.MinVersion); err != nil {
		return core.RunInfo{}, err
	}

	if err := os.MkdirAll(req.RuntimeDir, 0o700); err != nil {
		return core.RunInfo{}, err
	}
	cfgPath := filepath.Join(req.RuntimeDir, "sing-box.json")
	if err := os.WriteFile(cfgPath, cfg, 0o600); err != nil {
		return core.RunInfo{}, err
	}
	// Validate before detaching: `sing-box check -c`.
	if out, err := exec.CommandContext(ctx, bin, "check", "-c", cfgPath).CombinedOutput(); err != nil {
		return core.RunInfo{}, fmt.Errorf("sing-box check failed: %w\n%s", err, out)
	}

	logFile := req.LogFile
	if logFile == "" {
		logFile = filepath.Join(req.RuntimeDir, "sing-box.log")
	}
	pid, err := process.Start(ctx, bin, []string{"run", "-c", cfgPath}, logFile)
	if err != nil {
		return core.RunInfo{}, err
	}
	// Give the child a moment; if it died instantly, surface the log tail.
	time.Sleep(500 * time.Millisecond)
	if !platform.Alive(pid) {
		tail, _ := os.ReadFile(logFile)
		if len(tail) > 2000 {
			tail = tail[len(tail)-2000:]
		}
		return core.RunInfo{}, fmt.Errorf("sing-box exited immediately:\n%s", tail)
	}
	return core.RunInfo{PID: pid, ConfigPath: cfgPath, Core: c.Name()}, nil
}

// Stop implements core.Core.
func (c *Core) Stop(_ context.Context, info core.RunInfo) error {
	return process.Stop(info.PID, 10*time.Second)
}

// Status implements core.Core.
func (c *Core) Status(_ context.Context, info core.RunInfo) (core.Status, error) {
	return core.Status{
		Running:   platform.Alive(info.PID),
		PID:       info.PID,
		Core:      info.Core,
		ProfileID: "",
	}, nil
}
