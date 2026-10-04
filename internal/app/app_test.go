package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	cfg := config.Defaults()
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

func TestProfileEdit(t *testing.T) {
	st := testStore(t)
	svc := NewProfileService(st)
	p, err := svc.AddFromURI("trojan://pw@h:1#home")
	if err != nil {
		t.Fatal(err)
	}
	// rename keeps the id
	renamed, err := svc.Edit(p.ID, "office", "")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.ID != p.ID || renamed.Name != "office" {
		t.Fatalf("renamed = %+v", renamed)
	}
	list, _ := svc.List()
	if len(list) != 1 || list[0].Name != "office" {
		t.Fatalf("list = %+v", list)
	}
	// uri replace swaps the identity, keeps the name and the profile count
	replaced, err := svc.Edit(p.ID, "", "trojan://pw2@h2:2#ignored")
	if err != nil {
		t.Fatal(err)
	}
	if replaced.ID == p.ID || replaced.Name != "office" {
		t.Fatalf("replaced = %+v", replaced)
	}
	list, _ = svc.List()
	if len(list) != 1 || list[0].ID != replaced.ID {
		t.Fatalf("list = %+v", list)
	}
	// active selection follows the replaced id
	if _, err := svc.Use(replaced.ID); err != nil {
		t.Fatal(err)
	}
	after, err := svc.Edit(replaced.ID, "", "trojan://pw3@h3:3#x")
	if err != nil {
		t.Fatal(err)
	}
	if a, err := st.ActiveProfile(); err != nil || a.ID != after.ID {
		t.Fatalf("active = %+v, %v", a, err)
	}
	// subscription-owned profiles are refused
	owned := domain.Profile{ID: "sub1", Name: "remote", Protocol: domain.ProtocolTrojan, Source: "subscription:x"}
	if err := st.SaveProfile(owned); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Edit("sub1", "new", ""); !errors.Is(err, domain.ErrProfileManaged) {
		t.Fatalf("expected managed error, got %v", err)
	}
	if _, err := svc.Edit("missing", "n", ""); !errors.Is(err, domain.ErrProfileNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestSubEdit(t *testing.T) {
	st := testStore(t)
	svc := NewSubscriptionService(st, st)
	sub, err := svc.Add("s", "https://example.com/1")
	if err != nil {
		t.Fatal(err)
	}
	edited, err := svc.Edit(sub.ID, "renamed", "https://example.com/2")
	if err != nil {
		t.Fatal(err)
	}
	if edited.ID != sub.ID || edited.Name != "renamed" || edited.URL != "https://example.com/2" {
		t.Fatalf("edited = %+v", edited)
	}
	list, _ := svc.List()
	if len(list) != 1 || list[0].Name != "renamed" || list[0].URL != "https://example.com/2" {
		t.Fatalf("list = %+v", list)
	}
	if _, err := svc.Edit("missing", "n", ""); !errors.Is(err, domain.ErrSubscriptionNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestDownForeignProcessNeedsSudo(t *testing.T) {
	if isRoot() {
		t.Skip("requires unprivileged user")
	}
	st := testStore(t)
	cfg := config.Defaults()
	paths := platform.Paths{DataDir: t.TempDir(), RuntimeDir: t.TempDir(), StateFile: filepath.Join(t.TempDir(), "state.json")}
	reg := core.NewRegistry()
	svc := NewConnectionService(st, reg, cfg, paths)
	// PID 1 exists but belongs to root: down must ask for sudo, not kill.
	if err := st.SaveState(storage.State{Core: "sing-box", ProfileID: "p", PID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := svc.Down(context.Background()); !errors.Is(err, domain.ErrNotPrivileged) {
		t.Fatalf("expected not-privileged, got %v", err)
	}
}

func TestStatusShowsActiveWhileStopped(t *testing.T) {
	st := testStore(t)
	cfg := config.Defaults()
	paths := platform.Paths{DataDir: t.TempDir(), RuntimeDir: t.TempDir(), StateFile: filepath.Join(t.TempDir(), "state.json")}
	svc := NewConnectionService(st, core.NewRegistry(), cfg, paths)
	psvc := NewProfileService(st)
	p, err := psvc.AddFromURI("trojan://pw@h.example:443#one")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := psvc.Use(p.ID); err != nil {
		t.Fatal(err)
	}
	view, err := svc.Status(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if view.Running || view.ProfileName != "one" || view.ProfileID == "" {
		t.Fatalf("must expose active profile while stopped: %+v", view)
	}
}

func TestSettingsServiceUpdate(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config.yaml"
	def := config.Defaults()
	if err := config.Save(path, def); err != nil {
		t.Fatal(err)
	}
	svc := NewSettingsService(path)
	got, err := svc.Update(func(c *config.Config) { c.Features.Adblock = true })
	if err != nil {
		t.Fatal(err)
	}
	if !got.Features.Adblock {
		t.Fatalf("returned config: %+v", got)
	}
	reread, err := config.Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reread.Features.Adblock || reread.Features.TrackerBlock {
		t.Fatalf("not persisted: %+v", reread.Features)
	}
	// invalid mutation errors and leaves the file untouched
	if _, err := svc.Update(func(c *config.Config) {
		c.Features.TrackerBlock = true
		c.Features.SplitInclude = []string{"bad/999"}
	}); err == nil {
		t.Fatal("expected validation error")
	}
	reread, err = config.Load(path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reread.Features.Adblock || reread.Features.TrackerBlock || len(reread.Features.SplitInclude) != 0 {
		t.Fatalf("file must be untouched after failed update: %+v", reread.Features)
	}
}

type fakeRT struct {
	status int
	body   string
}

func (f fakeRT) RoundTrip(*http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: f.status, Body: io.NopCloser(strings.NewReader(f.body))}, nil
}

func TestUpdateServiceAutoCheckAndRecord(t *testing.T) {
	dir := t.TempDir()
	svc := NewUpdateService("x/y", "v0.0.1", dir)
	svc.Client = &http.Client{Transport: fakeRT{200, `{"tag_name":"v0.0.1"}`}}
	// up to date: AutoCheck is a no-op and check date is recorded
	v, err := svc.AutoCheck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if v != "" {
		t.Fatalf("no update expected, got %q", v)
	}
	if svc.LastCheck().IsZero() {
		t.Fatal("last check must be recorded")
	}
	data, err := os.ReadFile(dir + "/updates.json")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "lastCheck") {
		t.Fatalf("record file: %s", data)
	}
	// network failure surfaces as error
	svc.Client = &http.Client{Transport: fakeRT{500, "boom"}}
	if _, err := svc.AutoCheck(context.Background()); err == nil {
		t.Fatal("expected check error")
	}
}
