// Package testutil provides in-memory fakes of the TUI service
// interfaces with injectable errors and delays.
package testutil

import (
	"context"
	"time"

	"github.com/SabirDzh/VpnCLI/internal/app"
	"github.com/SabirDzh/VpnCLI/internal/domain"
)

// FakeConn is an in-memory shared.ConnectionAPI.
type FakeConn struct {
	St        app.StatusView
	StatusErr error
	UpErr     error
	DownErr   error
	Delay     time.Duration

	UpCalls   []string
	DownCalls int
}

// Status implements shared.FakeConn.
func (f *FakeConn) Status(ctx context.Context) (app.StatusView, error) {
	if f.Delay > 0 {
		select {
		case <-ctx.Done():
			return app.StatusView{}, ctx.Err()
		case <-time.After(f.Delay):
		}
	}
	return f.St, f.StatusErr
}

// Up implements shared.FakeConn.
func (f *FakeConn) Up(_ context.Context, ref string) error {
	f.UpCalls = append(f.UpCalls, ref)
	return f.UpErr
}

// Down implements shared.FakeConn.
func (f *FakeConn) Down(_ context.Context) error {
	f.DownCalls++
	return f.DownErr
}

// FakeProfiles is an in-memory shared.ProfileAPI.
type FakeProfiles struct {
	Items     []domain.Profile
	ActiveID  string
	ListErr   error
	ActiveErr error
	AddErr    error
	UseErr    error
	RemoveErr error
	EditErr   error

	Used    []string
	Removed []string
	Added   []string
	RawAdds []RawAdd
	Edited  []EditCall
}

// EditCall records one Edit call; Value is the URI (profiles) or URL (subscriptions).
type EditCall struct {
	ID    string
	Name  string
	Value string
}

// List implements shared.FakeProfiles.
func (f *FakeProfiles) List() ([]domain.Profile, error) { return f.Items, f.ListErr }

// Add implements shared.ProfileAPI.
func (f *FakeProfiles) Add(uri string) (domain.Profile, error) {
	if f.AddErr != nil {
		return domain.Profile{}, f.AddErr
	}
	p := domain.Profile{ID: "new", Name: uri}
	f.Items = append(f.Items, p)
	f.Added = append(f.Added, uri)
	return p, nil
}

// RawAdded records AddRaw calls.
type RawAdd struct {
	Name string
	Proto domain.Protocol
	Data []byte
}

// AddRaw implements shared.ProfileAPI.
func (f *FakeProfiles) AddRaw(name string, proto domain.Protocol, data []byte) (domain.Profile, error) {
	if f.AddErr != nil {
		return domain.Profile{}, f.AddErr
	}
	f.RawAdds = append(f.RawAdds, RawAdd{Name: name, Proto: proto, Data: data})
	p := domain.Profile{ID: "new-raw", Name: name, Protocol: proto, Raw: data}
	f.Items = append(f.Items, p)
	return p, nil
}

// Active implements shared.FakeProfiles.
func (f *FakeProfiles) Active() (domain.Profile, error) {
	if f.ActiveErr != nil {
		return domain.Profile{}, f.ActiveErr
	}
	for _, p := range f.Items {
		if p.ID == f.ActiveID {
			return p, nil
		}
	}
	return domain.Profile{}, domain.ErrNoActiveProfile
}

// Use implements shared.FakeProfiles.
func (f *FakeProfiles) Use(id string) (domain.Profile, error) {
	if f.UseErr != nil {
		return domain.Profile{}, f.UseErr
	}
	f.Used = append(f.Used, id)
	for _, p := range f.Items {
		if p.ID == id || p.Name == id {
			return p, nil
		}
	}
	return domain.Profile{}, domain.ErrProfileNotFound
}

// Remove implements shared.FakeProfiles.
func (f *FakeProfiles) Remove(id string) error {
	f.Removed = append(f.Removed, id)
	return f.RemoveErr
}

// Edit implements shared.ProfileAPI.
func (f *FakeProfiles) Edit(idOrName, name, uri string) (domain.Profile, error) {
	if f.EditErr != nil {
		return domain.Profile{}, f.EditErr
	}
	f.Edited = append(f.Edited, EditCall{ID: idOrName, Name: name, Value: uri})
	return domain.Profile{ID: idOrName, Name: name}, nil
}

// FakeSubs is an in-memory shared.SubscriptionAPI.
type FakeSubs struct {
	Items     []domain.Subscription
	ListErr   error
	UpdateErr error
	UpdateN   int
	AddErr    error
	EditErr   error

	Updated []string
	Removed []string
	Added   [][2]string
	Edited  []EditCall
}

// List implements shared.FakeSubs.
func (f *FakeSubs) List() ([]domain.Subscription, error) { return f.Items, f.ListErr }

// Update implements shared.FakeSubs.
func (f *FakeSubs) Update(id string) (int, error) {
	f.Updated = append(f.Updated, id)
	return f.UpdateN, f.UpdateErr
}

// Remove implements shared.FakeSubs.
func (f *FakeSubs) Remove(id string) error {
	f.Removed = append(f.Removed, id)
	return nil
}

// Add implements shared.SubscriptionAPI.
func (f *FakeSubs) Add(name, url string) (domain.Subscription, error) {
	if f.AddErr != nil {
		return domain.Subscription{}, f.AddErr
	}
	sub := domain.Subscription{ID: "new-sub", Name: name, URL: url}
	f.Items = append(f.Items, sub)
	f.Added = append(f.Added, [2]string{name, url})
	return sub, nil
}

// Edit implements shared.SubscriptionAPI.
func (f *FakeSubs) Edit(idOrName, name, url string) (domain.Subscription, error) {
	if f.EditErr != nil {
		return domain.Subscription{}, f.EditErr
	}
	f.Edited = append(f.Edited, EditCall{ID: idOrName, Name: name, Value: url})
	return domain.Subscription{ID: idOrName, Name: name, URL: url}, nil
}

// FakeUpdate is an in-memory shared.UpdateAPI.
type FakeUpdate struct {
	Res         app.CheckResult
	CheckErr    error
	UpdateErr   error
	CheckCalls  int
	UpdateCalls []string
	CheckTime   time.Time
	UpdatedTime time.Time
}

// Check implements shared.UpdateAPI.
func (f *FakeUpdate) Check(context.Context) (app.CheckResult, error) {
	f.CheckCalls++
	f.CheckTime = time.Now()
	return f.Res, f.CheckErr
}

// Update implements shared.UpdateAPI.
func (f *FakeUpdate) Update(_ context.Context, tag string) error {
	if f.UpdateErr != nil {
		return f.UpdateErr
	}
	f.UpdateCalls = append(f.UpdateCalls, tag)
	f.UpdatedTime = time.Now()
	return nil
}

// LastCheck implements shared.UpdateAPI.
func (f *FakeUpdate) LastCheck() time.Time { return f.CheckTime }

// LastUpdated implements shared.UpdateAPI.
func (f *FakeUpdate) LastUpdated() time.Time { return f.UpdatedTime }
