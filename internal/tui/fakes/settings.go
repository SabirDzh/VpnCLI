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

// FakeSettings is an in-memory shared.SettingsAPI.
type FakeSettings struct {
	Info             shared.SettingsInfo
	SetErr           error
	BlockCalls       []BlockCall
	AppFirewallCalls [][]string
	KillSwitchCalls  []bool
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
