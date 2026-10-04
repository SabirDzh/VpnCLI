// Package fakes holds TUI service fakes that reference shared types.
// (testutil cannot import shared: shared's own tests import testutil.)
package fakes

import (
	"github.com/SabirDzh/VpnCLI/internal/tui/shared"
)

// FakeSettings is an in-memory shared.SettingsAPI.
type FakeSettings struct {
	Info         shared.SettingsInfo
	SetErr       error
	AdblockCalls []bool
	TrackerCalls []bool
	SplitCalls   [][2][]string
	AutoCalls    []bool
	Snapshots    int
}

// Snapshot implements shared.SettingsAPI.
func (f *FakeSettings) Snapshot() shared.SettingsInfo {
	f.Snapshots++
	return f.Info
}

// SetAdblock implements shared.SettingsAPI.
func (f *FakeSettings) SetAdblock(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.AdblockCalls = append(f.AdblockCalls, on)
	f.Info.Adblock = on
	return nil
}

// SetTrackerBlock implements shared.SettingsAPI.
func (f *FakeSettings) SetTrackerBlock(on bool) error {
	if f.SetErr != nil {
		return f.SetErr
	}
	f.TrackerCalls = append(f.TrackerCalls, on)
	f.Info.TrackerBlock = on
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
