package shared

import (
	"context"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/domain"
	"github.com/SabirDzh/VpnCLI/internal/tui/testutil"
)

func TestFetchStatus(t *testing.T) {
	conn := &testutil.FakeConn{St: app.StatusView{Running: true}}
	msg := FetchStatus(context.Background(), conn)()
	m, ok := msg.(StatusMsg)
	if !ok || !m.St.Running || m.Err != nil {
		t.Fatalf("got %#v", msg)
	}
	conn.StatusErr = domain.ErrNotRunning
	msg = FetchStatus(context.Background(), conn)()
	if m := msg.(StatusMsg); m.Err == nil {
		t.Fatal("must propagate error")
	}
}

func TestFetchProfiles(t *testing.T) {
	prof := &testutil.FakeProfiles{
		Items:    []domain.Profile{{ID: "a", Name: "a"}},
		ActiveID: "a",
	}
	m := FetchProfiles(prof)().(ProfilesMsg)
	if m.ActiveID != "a" || len(m.List) != 1 {
		t.Fatalf("got %#v", m)
	}
	prof.ListErr = domain.ErrProfileNotFound
	if m := FetchProfiles(prof)().(ProfilesMsg); m.Err == nil {
		t.Fatal("must propagate error")
	}
}

func TestDoUse(t *testing.T) {
	prof := &testutil.FakeProfiles{Items: []domain.Profile{{ID: "a"}}}
	if m := DoUse(prof, "a", "a")().(OpDoneMsg); m.Op != "use" || m.Err != nil {
		t.Fatalf("got %#v", m)
	}
}

func TestFetchSubsCounts(t *testing.T) {
	prof := &testutil.FakeProfiles{Items: []domain.Profile{
		{ID: "p1", Source: "subscription:s1"},
		{ID: "p2", Source: "subscription:s1"},
		{ID: "p3", Source: domain.ManualSource},
	}}
	subs := &testutil.FakeSubs{Items: []domain.Subscription{{ID: "s1"}}}
	m := FetchSubs(prof, subs)().(SubsMsg)
	if m.Counts["subscription:s1"] != 2 {
		t.Fatalf("counts = %v", m.Counts)
	}
	subs.ListErr = domain.ErrSubscriptionNotFound
	if m := FetchSubs(prof, subs)().(SubsMsg); m.Err == nil {
		t.Fatal("must propagate error")
	}
}

func TestDoSubUpdate(t *testing.T) {
	subs := &testutil.FakeSubs{UpdateN: 3}
	m := DoSubUpdate(subs, "s1", false)().(OpDoneMsg)
	if m.Op != "update" || m.N != 3 || m.Err != nil {
		t.Fatalf("got %#v", m)
	}
	subs.UpdateErr = domain.ErrSubscriptionNotFound
	if m := DoSubUpdate(subs, "s1", false)().(OpDoneMsg); m.Err == nil {
		t.Fatal("must propagate error")
	}
	all := DoSubUpdate(subs, "", true)().(OpDoneMsg)
	if all.Op != "update-all" {
		t.Fatalf("got %#v", all)
	}
}

func TestScheduleTick(t *testing.T) {
	if ScheduleTick(0) == nil {
		t.Fatal("must return a command")
	}
}

func TestDoErrors(t *testing.T) {
	conn := &testutil.FakeConn{UpErr: errX{}, DownErr: errX{}}
	if m := DoUp(ctx(), conn, "a", "a")().(OpDoneMsg); m.Err == nil {
		t.Fatal("must propagate up error")
	}
	if m := DoDown(ctx(), conn)().(OpDoneMsg); m.Err == nil {
		t.Fatal("must propagate down error")
	}
	prof := &testutil.FakeProfiles{UseErr: errX{}}
	if m := DoUse(prof, "a", "a")().(OpDoneMsg); m.Err == nil {
		t.Fatal("must propagate use error")
	}
}

func ctx() context.Context { return context.Background() }

type errX struct{}

func (errX) Error() string { return "x" }
