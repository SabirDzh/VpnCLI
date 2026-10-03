package domain

import "errors"

var (
	ErrProfileNotFound      = errors.New("profile not found")
	ErrSubscriptionNotFound = errors.New("subscription not found")
	ErrCoreNotFound         = errors.New("core not found")
	ErrUnsupportedProtocol  = errors.New("unsupported protocol")
	ErrNotPrivileged        = errors.New("root privileges required (run with sudo)")
	ErrAlreadyRunning       = errors.New("vpn is already running")
	ErrNotRunning           = errors.New("vpn is not running")
	ErrParse                = errors.New("parse error")
	ErrNoActiveProfile      = errors.New("no active profile selected")
	ErrInvalidConfig        = errors.New("invalid config")
)
