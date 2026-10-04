// Package domain holds engine-neutral types: profiles, subscriptions
// and sentinel errors shared across layers.
package domain

import "errors"

// Sentinel errors returned by services and matched with errors.Is.
var (
	// ErrProfileNotFound is returned when no profile matches id or name.
	ErrProfileNotFound = errors.New("profile not found")
	// ErrProfileManaged is returned when editing a profile owned by a subscription.
	ErrProfileManaged = errors.New("profile is managed by a subscription")
	// ErrSubscriptionNotFound is returned when no subscription matches.
	ErrSubscriptionNotFound = errors.New("subscription not found")
	// ErrCoreNotFound is returned for unknown core names.
	ErrCoreNotFound = errors.New("core not found")
	// ErrUnsupportedProtocol is returned when no core supports the profile.
	ErrUnsupportedProtocol = errors.New("unsupported protocol")
	// ErrNotPrivileged is returned when TUN requires root/admin.
	ErrNotPrivileged = errors.New("root privileges required (run with sudo)")
	// ErrAlreadyRunning is returned when the VPN is already up.
	ErrAlreadyRunning = errors.New("vpn is already running")
	// ErrNotRunning is returned when the VPN is already down.
	ErrNotRunning = errors.New("vpn is not running")
	// ErrParse is returned for malformed URIs and configs.
	ErrParse = errors.New("parse error")
	// ErrNoActiveProfile is returned when no profile is selected.
	ErrNoActiveProfile = errors.New("no active profile selected")
	// ErrInvalidConfig is returned for invalid app configuration.
	ErrInvalidConfig = errors.New("invalid config")
)
