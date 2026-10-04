package app

import (
	"slices"
	"testing"
)

func TestEffectiveApps(t *testing.T) {
	base := []string{"Miro"}
	got := EffectiveApps(base, true)
	if !slices.Contains(got, "Miro") || !slices.Contains(got, RFApps[0]) {
		t.Fatalf("preset must merge: %v", got)
	}
	seen := map[string]bool{}
	for _, a := range got {
		if seen[a] {
			t.Fatalf("duplicate %q in %v", a, got)
		}
		seen[a] = true
	}
	if !slices.Equal(EffectiveApps(base, false), base) {
		t.Fatal("preset off must return base as-is")
	}
}
