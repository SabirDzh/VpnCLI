package storage

import (
	"path/filepath"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	dir := t.TempDir()
	s, err := New(filepath.Join(dir, "data"), filepath.Join(dir, "run", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestProfileCRUD(t *testing.T) {
	s := newTestStore(t)
	p := domain.Profile{
		ID: "abc123", Name: "p1", Protocol: domain.ProtocolTrojan,
		Source:   domain.ManualSource,
		Endpoint: domain.Endpoint{Host: "h.example", Port: 443},
		Settings: domain.ProtocolSettings{Password: "pw"},
	}
	if err := s.SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetProfile("abc123")
	if err != nil || got.Name != "p1" {
		t.Fatalf("get: %+v %v", got, err)
	}
	if _, err := s.GetProfile("nope"); err == nil {
		t.Fatal("expected not found")
	}
	if _, err := s.ActiveProfile(); err == nil {
		t.Fatal("expected no-active error")
	}
	sel, err := s.SetActiveProfile("p1") // by name
	if err != nil || sel.ID != "abc123" {
		t.Fatalf("use: %+v %v", sel, err)
	}
	list, err := s.ListProfiles()
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	// upsert
	p.Name = "p1-renamed"
	if err := s.SaveProfile(p); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteProfile("abc123"); err != nil {
		t.Fatal(err)
	}
	if list, _ := s.ListProfiles(); len(list) != 0 {
		t.Fatalf("expected empty, got %+v", list)
	}
}

func TestSubscriptions(t *testing.T) {
	s := newTestStore(t)
	sub := domain.Subscription{ID: "s1", Name: "sub", URL: "https://example.com/sub"}
	if err := s.SaveSubscription(sub); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListSubscriptions()
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %+v %v", list, err)
	}
	if err := s.DeleteSubscription("sub"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSubscription("s1"); err == nil {
		t.Fatal("expected not found")
	}
}

func TestStateRoundtrip(t *testing.T) {
	s := newTestStore(t)
	if st, _ := s.LoadState(); st != nil {
		t.Fatal("expected nil state")
	}
	if err := s.SaveState(State{Core: "sing-box", ProfileID: "p1", PID: 1234}); err != nil {
		t.Fatal(err)
	}
	st, err := s.LoadState()
	if err != nil || st.PID != 1234 || st.Core != "sing-box" {
		t.Fatalf("state: %+v %v", st, err)
	}
	if err := s.ClearState(); err != nil {
		t.Fatal(err)
	}
	if st, _ := s.LoadState(); st != nil {
		t.Fatal("expected cleared state")
	}
}
