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
	Stack          string
	AutoRoute      bool
	StrictRoute    bool
	MixedPort      int
	Privileged     bool
	ConfigDir      string
	DataDir        string
	StateFile      string
	LogFile        string
	// Features (TUI-managed, applied on next connect).
	Adblock          bool
	TrackerBlock     bool
	SocialBlock      bool
	KillSwitch       bool
	SplitMode        string // exclude|include|off
	SplitExclude     []string
	SplitInclude     []string
	SplitExcludeApps []string
	SplitIncludeApps []string
	PresetApps       bool
	AppFirewall      []string
	Multiplex        string // off|on|auto
	// DNS settings.
	DNSServers  []string
	DNSStrategy string
	// Update settings.
	AutoUpdate bool
}

// SettingsAPI reads and changes persisted settings. Snapshot reflects the
// file state after the last Set* call.
type SettingsAPI interface {
	Snapshot() SettingsInfo
	// SetBlocklist toggles a DNS blocklist kind: ads|trackers|social.
	SetBlocklist(kind string, on bool) error
	SetAppFirewall(apps []string) error
	SetKillSwitch(on bool) error
	// SetSplitMode switches the split tunneling mode: exclude|include|off.
	SetSplitMode(mode string) error
	// SetSplitApps edits an app list: which is exclude|include.
	SetSplitApps(which string, apps []string) error
	SetPresetApps(on bool) error
	SetMultiplex(mode string) error
	SetDNSServers(servers []string) error
	SetDNSstrategy(strategy string) error
	SetTUNEnabled(on bool) error
	SetMTU(mtu int) error
	SetStack(stack string) error
	SetAutoRoute(on bool) error
	SetStrictRoute(on bool) error
	SetMixedPort(port int) error
	SetLogLevel(level string) error
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
	// AddRaw imports a native core config blob as-is.
	AddRaw(name string, proto domain.Protocol, data []byte) (domain.Profile, error)
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
