package shared

import (
	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// Shared message types exchanged between the root model and screens.
// Exported so screen subpackages can produce and consume them.

// StatusMsg carries a connection status snapshot.
type StatusMsg struct {
	St  app.StatusView
	Err error
}

// ProfilesMsg carries the profile list with the active id.
type ProfilesMsg struct {
	List     []domain.Profile
	ActiveID string
	Err      error
}

// SubsMsg carries subscriptions with per-source profile counts.
type SubsMsg struct {
	Subs   []domain.Subscription
	Counts map[string]int
	Err    error
}

// OpDoneMsg reports completion of an async operation (up/down/use/update).
type OpDoneMsg struct {
	Op    string // "up" | "down" | "use" | "update" | "update-all"
	Label string // human context, e.g. profile name
	N     int    // affected profiles (update ops)
	Err   error
}

// TickMsg triggers the next status poll. It is scheduled only after the
// previous StatusMsg arrived, so slow responses never pile up.
type TickMsg struct{}

// ToastExpiredMsg hides the toast with the matching id.
type ToastExpiredMsg struct{ ID int }

// ConfirmResultMsg carries a modal dialog verdict.
type ConfirmResultMsg struct {
	Tag string
	OK  bool
}
