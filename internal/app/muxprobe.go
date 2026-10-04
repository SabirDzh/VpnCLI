package app

import (
	"context"
	"io"
	"os"
	"regexp"
	"time"
)

// muxWindow is the probing window after a run with mux enabled: a server
// that does not support multiplex fails its handshake within seconds.
const muxWindow = 10 * time.Second

var muxFailRe = regexp.MustCompile(`(?i)(multiplex|mux).*(failed|error|closed|refused)|unknown transport`)

// WatchMux tails logFile during the probe window and calls onFallback at
// the first mux handshake failure, so the caller can restart without mux.
// Only lines appended after the call are inspected.
func WatchMux(ctx context.Context, logFile string, onFallback func()) {
	f, err := os.Open(logFile)
	if err != nil {
		return
	}
	defer f.Close()
	if _, err := f.Seek(0, io.SeekEnd); err != nil {
		return
	}
	deadline := time.Now().Add(muxWindow)
	buf := make([]byte, 0, 8192)
	chunk := make([]byte, 4096)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, rerr := f.Read(chunk)
		if n > 0 {
			buf = append(buf, chunk[:n]...)
			if len(buf) > 64*1024 {
				buf = buf[len(buf)-32*1024:]
			}
			if muxFailRe.Match(buf) {
				onFallback()
				return
			}
		}
		if rerr != nil {
			time.Sleep(200 * time.Millisecond)
		}
	}
}
