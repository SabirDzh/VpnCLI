// Package logging configures slog with secret redaction.
package logging

import (
	"context"
	"io"
	"log/slog"
	"strings"
)

// secrets replaced in log lines (query params like uuid/password stay out of logs).
var secretKeys = []string{"uuid", "password", "psk", "publickey", "public-key", "shortid", "token"}

// New creates a text slog logger at level.
func New(out io.Writer, level string) *slog.Logger {
	lvl := slog.LevelInfo
	switch strings.ToLower(level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	}
	h := slog.NewTextHandler(out, &slog.HandlerOptions{Level: lvl})
	return slog.New(redactor{h})
}

type redactor struct{ slog.Handler }

func (r redactor) Handle(ctx context.Context, e slog.Record) error {
	e.Message = Redact(e.Message)
	return r.Handler.Handle(ctx, e)
}

// Redact replaces obvious secret fragments in a string.
func Redact(s string) string {
	low := strings.ToLower(s)
	for _, k := range secretKeys {
		if strings.Contains(low, k) {
			return "[redacted]"
		}
	}
	// vless://uuid@ or trojan://pass@ style URIs
	if strings.Contains(s, "://") && strings.Contains(s, "@") {
		return "[redacted-uri]"
	}
	return s
}
