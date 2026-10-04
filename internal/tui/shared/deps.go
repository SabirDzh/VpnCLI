// Package shared holds the TUI kernel: service interfaces, messages,
// command constructors and error texts shared by the root model, screens
// and components. It lives in a leaf package so screens can use it
// without an import cycle with the root tui package.
package shared

import (
	"context"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// Deps wires services into the TUI. ReadOnly is set when the process
// lacks TUN privileges: lists and status work, up/down refuse cleanly.
type Deps struct {
	Connection  ConnectionAPI
	Profiles    ProfileAPI
	Subs        SubscriptionAPI
	Settings    SettingsInfo
	SettingsAPI SettingsAPI
	Update      UpdateAPI
	Version     string
	Repo        string
	ReadOnly    bool
}

// SettingsInfo is a snapshot of the effective configuration and
// environment, assembled once by the composition root.
type SettingsInfo struct {
	CoreDefault    string
	SingBoxPath    string
	SingBoxVersion string
	SingBoxErr     string
	MinVersion     string
	LogLevel       string
	TUNEnabled     bool
	MTU            int
	AutoRoute      bool
	StrictRoute    bool
	MixedPort      int
	Privileged     bool
	ConfigDir      string
	DataDir        string
	StateFile      string
	LogFile        string
	// Features (TUI-managed, applied on next connect).
	Adblock      bool
	TrackerBlock bool
	SplitExclude []string
	SplitInclude []string
	// Update settings.
	AutoUpdate bool
}

// SettingsAPI reads and changes persisted settings. Snapshot reflects the
// file state after the last Set* call.
type SettingsAPI interface {
	Snapshot() SettingsInfo
	SetAdblock(on bool) error
	SetTrackerBlock(on bool) error
	SetSplit(exclude, include []string) error
	SetAutoUpdate(on bool) error
}

// UpdateAPI checks and installs CLI updates and reports last-run dates.
type UpdateAPI interface {
	Check(ctx context.Context) (app.CheckResult, error)
	Update(ctx context.Context, tag string) error
	LastCheck() time.Time
	LastUpdated() time.Time
}

// ConnectionAPI is the subset of ConnectionService the TUI needs.
type ConnectionAPI interface {
	Status(ctx context.Context) (app.StatusView, error)
	Up(ctx context.Context, profileRef string) error
	Down(ctx context.Context) error
}

// ProfileAPI is the subset of ProfileService the TUI needs.
type ProfileAPI interface {
	List() ([]domain.Profile, error)
	Add(uri string) (domain.Profile, error)
	Use(idOrName string) (domain.Profile, error)
	Remove(idOrName string) error
	Edit(idOrName, name, uri string) (domain.Profile, error)
	Active() (domain.Profile, error)
}

// SubscriptionAPI is the subset of SubscriptionService the TUI needs.
type SubscriptionAPI interface {
	List() ([]domain.Subscription, error)
	Add(name, url string) (domain.Subscription, error)
	Update(idOrName string) (int, error)
	Remove(idOrName string) error
	Edit(idOrName, name, url string) (domain.Subscription, error)
}
