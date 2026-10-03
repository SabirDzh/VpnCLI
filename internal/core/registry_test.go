package core

import (
	"context"
	"testing"

	"github.com/SabirDzh/VpnCLI/internal/domain"
)

type fakeCore struct{ name string }

func (f *fakeCore) Name() string { return f.name }
func (f *fakeCore) Supports(p domain.Profile) bool {
	return p.Protocol == domain.ProtocolTrojan
}

func (f *fakeCore) Start(_ context.Context, _ StartRequest) (RunInfo, error) {
	return RunInfo{PID: 42, Core: f.name}, nil
}
func (f *fakeCore) Stop(_ context.Context, _ RunInfo) error { return nil }
func (f *fakeCore) Status(_ context.Context, info RunInfo) (Status, error) {
	return Status{Running: true, PID: info.PID, Core: f.name}, nil
}

func testRegistry() *Registry {
	r := NewRegistry()
	r.Register("fake", func(Settings) (Core, error) { return &fakeCore{"fake"}, nil }, Settings{})
	return r
}

func TestSelectPreferred(t *testing.T) {
	r := testRegistry()
	p := domain.Profile{Protocol: domain.ProtocolTrojan}
	c, err := r.Select(p, "fake")
	if err != nil || c.Name() != "fake" {
		t.Fatalf("select: %v %v", c, err)
	}
}

func TestSelectUnsupported(t *testing.T) {
	r := testRegistry()
	if _, err := r.Select(domain.Profile{Protocol: domain.ProtocolVLESS}, ""); err == nil {
		t.Fatal("expected unsupported error")
	}
}

func TestGetUnknown(t *testing.T) {
	if _, err := testRegistry().Get("nope"); err == nil {
		t.Fatal("expected core-not-found")
	}
}
