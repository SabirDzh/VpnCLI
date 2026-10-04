//go:build !darwin

package platform

import "errors"

// ErrKillSwitchUnsupported is returned where no kill switch backend exists.
var ErrKillSwitchUnsupported = errors.New("kill switch is not supported on this platform yet")

// KillSwitchRules is not implemented off darwin.
func KillSwitchRules() string { return "" }

// KillSwitchRulesFor is not implemented off darwin.
func KillSwitchRulesFor(_ string, _ int) (string, error) { return "", ErrKillSwitchUnsupported }

// EnableKillSwitch is not implemented off darwin.
func EnableKillSwitch(_ string, _ int) error { return ErrKillSwitchUnsupported }

// DisableKillSwitch is not implemented off darwin.
func DisableKillSwitch() error { return ErrKillSwitchUnsupported }

// KillSwitchActive is not implemented off darwin.
func KillSwitchActive() bool { return false }
