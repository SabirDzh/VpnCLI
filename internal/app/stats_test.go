package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStatsTotals(t *testing.T) {
	var hitBase string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hitBase = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"uploadTotal":1024,"downloadTotal":2097152,"connections":[{},{},{}]}`))
	}))
	defer srv.Close()

	log := filepath.Join(t.TempDir(), "sb.log")
	os.WriteFile(log, []byte("INFO router: rejected connection\nINFO other line\nINFO router: rejected again\n"), 0o600)

	s := NewStatsService(strings.TrimPrefix(srv.URL, "http://"), log)
	got, err := s.Totals(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if hitBase != "/connections" {
		t.Fatalf("endpoint: %s", hitBase)
	}
	if got.Up != 1024 || got.Down != 2097152 || got.Active != 3 {
		t.Fatalf("totals: %+v", got)
	}
	if got.Blocked != 2 {
		t.Fatalf("blocked: %d", got.Blocked)
	}
}

func TestStatsTailLog(t *testing.T) {
	log := filepath.Join(t.TempDir(), "sb.log")
	var b strings.Builder
	for i := 0; i < 300; i++ {
		b.WriteString("line\n")
	}
	os.WriteFile(log, []byte(b.String()), 0o600)
	s := NewStatsService("127.0.0.1:1", log)
	tail := s.TailLog(10)
	if len(tail) != 10 || tail[9] != "line" {
		t.Fatalf("tail: %d lines", len(tail))
	}
}

func TestStatsMissingLog(t *testing.T) {
	s := NewStatsService("127.0.0.1:1", filepath.Join(t.TempDir(), "absent.log"))
	n, err := s.Blocked()
	if err != nil || n != 0 {
		t.Fatalf("missing log must be zero blocked, got %d %v", n, err)
	}
	if tail := s.TailLog(10); tail != nil {
		t.Fatalf("missing log must give no tail: %v", tail)
	}
}
