// Package fakes holds TUI service fakes that reference shared types.
// (testutil cannot import shared: shared's own tests import testutil.)
package fakes

import (
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
)

// BlockCall records one SetBlocklist invocation.
type BlockCall struct {
	Kind string
	On   bool
}

// SplitAppsCall records one SetSplitApps invocation.
type SplitAppsCall struct {
	Which string
	Apps  []string
}

// FakeSettings is an in-memory shared.SettingsAPI.
type FakeSettings struct {
	Info             shared.SettingsInfo
	SetErr           error
	BlockCalls       []BlockCall
	AppFirewallCalls [][]string
	KillSwitchCalls  []bool
	SplitModeCalls   []string
	SplitAppsCalls   []SplitAppsCall
	PresetCalls      []bool
	MuxCalls         []string
	DNSServersCalls  [][]string
	DNSstrategyCalls []string
	TUNCalls         []bool
	MTUCalls         []int
	StackCalls       []string
	AutoRouteCalls   []bool
	StrictRouteCalls []bool
	MixedPortCalls   []int
	LogLevelCalls    []string
	SplitCalls       [][2][]string
	AutoCalls        []bool
	Snapshots        int
}

// Snapshot implements shared.SettingsAPI.
func (f *FakeSettings) Snapshot() shared.SettingsInfo {
	f.Snapshots++
	return f.Info
}

// SetBlocklist implements shared.SettingsAPI.
func (f *FakeSettings) SetBlocklist(kind string, on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.BlockCalls = append(f.BlockCalls, BlockCall{Kind: kind, On: on})
	switch kind {
	case "ads":
		f.Info.Adblock = on
	case "trackers":
		f.Info.TrackerBlock = on
	case "social":
		f.Info.SocialBlock = on
	}
	return nil
}

// SetAppFirewall implements shared.SettingsAPI.
func (f *FakeSettings) SetAppFirewall(apps []string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.AppFirewallCalls = append(f.AppFirewallCalls, apps)
	f.Info.AppFirewall = apps
	return nil
}

// SetKillSwitch implements shared.SettingsAPI.
func (f *FakeSettings) SetKillSwitch(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.KillSwitchCalls = append(f.KillSwitchCalls, on)
	f.Info.KillSwitch = on
	return nil
}

// SetSplitMode implements shared.SettingsAPI.
func (f *FakeSettings) SetSplitMode(mode string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.SplitModeCalls = append(f.SplitModeCalls, mode)
	f.Info.SplitMode = mode
	return nil
}

// SetSplitApps implements shared.SettingsAPI.
func (f *FakeSettings) SetSplitApps(which string, apps []string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.SplitAppsCalls = append(f.SplitAppsCalls, SplitAppsCall{Which: which, Apps: apps})
	switch which {
	case "exclude":
		f.Info.SplitExcludeApps = apps
	case "include":
		f.Info.SplitIncludeApps = apps
	}
	return nil
}

// SetPresetApps implements shared.SettingsAPI.
func (f *FakeSettings) SetPresetApps(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.PresetCalls = append(f.PresetCalls, on)
	f.Info.PresetApps = on
	return nil
}

// SetMultiplex implements shared.SettingsAPI.
func (f *FakeSettings) SetMultiplex(mode string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.MuxCalls = append(f.MuxCalls, mode)
	f.Info.Multiplex = mode
	return nil
}

// SetDNSServers implements shared.SettingsAPI.
func (f *FakeSettings) SetDNSServers(servers []string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.DNSServersCalls = append(f.DNSServersCalls, servers)
	f.Info.DNSServers = servers
	return nil
}

// SetDNSstrategy implements shared.SettingsAPI.
func (f *FakeSettings) SetDNSstrategy(strategy string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.DNSstrategyCalls = append(f.DNSstrategyCalls, strategy)
	f.Info.DNSStrategy = strategy
	return nil
}

// SetTUNEnabled implements shared.SettingsAPI.
func (f *FakeSettings) SetTUNEnabled(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.TUNCalls = append(f.TUNCalls, on)
	f.Info.TUNEnabled = on
	return nil
}

// SetMTU implements shared.SettingsAPI.
func (f *FakeSettings) SetMTU(mtu int) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.MTUCalls = append(f.MTUCalls, mtu)
	f.Info.MTU = mtu
	return nil
}

// SetStack implements shared.SettingsAPI.
func (f *FakeSettings) SetStack(stack string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.StackCalls = append(f.StackCalls, stack)
	f.Info.Stack = stack
	return nil
}

// SetAutoRoute implements shared.SettingsAPI.
func (f *FakeSettings) SetAutoRoute(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.AutoRouteCalls = append(f.AutoRouteCalls, on)
	f.Info.AutoRoute = on
	return nil
}

// SetStrictRoute implements shared.SettingsAPI.
func (f *FakeSettings) SetStrictRoute(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.StrictRouteCalls = append(f.StrictRouteCalls, on)
	f.Info.StrictRoute = on
	return nil
}

// SetMixedPort implements shared.SettingsAPI.
func (f *FakeSettings) SetMixedPort(port int) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.MixedPortCalls = append(f.MixedPortCalls, port)
	f.Info.MixedPort = port
	return nil
}

// SetLogLevel implements shared.SettingsAPI.
func (f *FakeSettings) SetLogLevel(level string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.LogLevelCalls = append(f.LogLevelCalls, level)
	f.Info.LogLevel = level
	return nil
}

// SetSplit implements shared.SettingsAPI.
func (f *FakeSettings) SetSplit(exclude, include []string) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.SplitCalls = append(f.SplitCalls, [2][]string{exclude, include})
	f.Info.SplitExclude = exclude
	f.Info.SplitInclude = include
	return nil
}

// SetAutoUpdate implements shared.SettingsAPI.
func (f *FakeSettings) SetAutoUpdate(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.AutoCalls = append(f.AutoCalls, on)
	f.Info.AutoUpdate = on
	return nil
}
