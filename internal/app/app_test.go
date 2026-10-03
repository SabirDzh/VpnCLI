package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/config"
	"github.com/SabirDzh/VpnCLI/internal/core"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/platform"
	"github.com/SabirDzh/VpnCLI/internal/storage"
)

func testStore(t *testing.T) *storage.Store {
	t.Helper()
	dir := t.TempDir()
	s, err := storage.New(filepath.Join(dir, "data"), filepath.Join(dir, "run", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestSubscriptionSync(t *testing.T) {
	st := testStore(t)
	svc := NewSubscriptionService(st, st)
	sub, err := svc.Add("s", "https://example.com/sub")
	if err != nil {
		t.Fatal(err)
	}
	// fake remote: two profiles
	svc.fetch = func(url string) ([]domain.Profile, error) {
		a := domain.Profile{
			ID: "id-a", Name: "a", Protocol: domain.ProtocolTrojan,
			Endpoint: domain.Endpoint{Host: "h1", Port: 443},
		}
		b := domain.Profile{
			ID: "id-b", Name: "b", Protocol: domain.ProtocolVLESS,
			Endpoint: domain.Endpoint{Host: "h2", Port: 443},
		}
		return []domain.Profile{a, b}, nil
	}
	if n, err := svc.Update(sub.ID); err != nil || n != 2 {
		t.Fatalf("update: %d %v", n, err)
	}
	list, _ := st.ListProfiles()
	if len(list) != 2 || list[0].Source != domain.SubscriptionSource(sub.ID) {
		t.Fatalf("profiles: %+v", list)
	}
	// second sync drops id-b
	svc.fetch = func(url string) ([]domain.Profile, error) {
		return []domain.Profile{{
			ID: "id-a", Name: "a", Protocol: domain.ProtocolTrojan,
			Endpoint: domain.Endpoint{Host: "h1", Port: 443},
		}}, nil
	}
	if n, err := svc.Update(sub.ID); err != nil || n != 1 {
		t.Fatalf("update2: %d %v", n, err)
	}
	list, _ = st.ListProfiles()
	if len(list) != 1 || list[0].ID != "id-a" {
		t.Fatalf("prune: %+v", list)
	}
}

func TestProfileAddUse(t *testing.T) {
	st := testStore(t)
	svc := NewProfileService(st)
	p, err := svc.AddFromURI("trojan://pw@h.example:443#one")
	if err != nil {
		t.Fatal(err)
	}
	if p.Source != domain.ManualSource {
		t.Fatalf("source: %s", p.Source)
	}
	if _, err := svc.Use(p.ID); err != nil {
		t.Fatal(err)
	}
	if err := svc.Remove(p.ID); err != nil {
		t.Fatal(err)
	}
}

func TestConnectionGuards(t *testing.T) {
	st := testStore(t)
	cfg, _ := config.Load("", nil)
	paths := platform.Paths{DataDir: t.TempDir(), RuntimeDir: t.TempDir(), StateFile: filepath.Join(t.TempDir(), "state.json")}
	reg := core.NewRegistry()
	svc := NewConnectionService(st, reg, cfg, paths)
	ctx := context.Background()

	if _, err := svc.Up(ctx, ""); !errors.Is(err, domain.ErrNotPrivileged) && !isRoot() {
		t.Fatalf("expected not-privileged, got %v", err)
	}
	if _, err := svc.Status(ctx); err != nil {
		t.Fatal(err)
	}
	stt, _ := svc.Status(ctx)
	if stt.Running {
		t.Fatal("expected stopped")
	}
	if err := svc.Down(ctx); !errors.Is(err, domain.ErrNotRunning) {
		t.Fatalf("expected not-running, got %v", err)
	}
}

func isRoot() bool { return platform.IsPrivileged() }

func TestProfileErrors(t *testing.T) {
	st := testStore(t)
	svc := NewProfileService(st)
	if _, err := svc.AddFromURI("wireguard://x@y:1"); err == nil {
		t.Fatal("expected parse error")
	}
	if _, err := svc.AddRaw("n", domain.ProtocolVLESS, nil); err == nil {
		t.Fatal("expected empty raw error")
	}
	if _, err := svc.AddRaw("n", domain.ProtocolVLESS, []byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err := svc.Remove("missing"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestSubCRUD(t *testing.T) {
	st := testStore(t)
	svc := NewSubscriptionService(st, st)
	sub, _ := svc.Add("s", "https://example.com/1")
	if list, _ := svc.List(); len(list) != 1 {
		t.Fatalf("list: %+v", list)
	}
	if _, err := svc.Update("missing"); err == nil {
		t.Fatal("expected not found")
	}
	svc.fetch = func(url string) ([]domain.Profile, error) { return nil, errors.New("net down") }
	if _, err := svc.Update(sub.ID); err == nil {
		t.Fatal("expected fetch error")
	}
	if err := svc.Remove(sub.ID); err != nil {
		t.Fatal(err)
	}
}
