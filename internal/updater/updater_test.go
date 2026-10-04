package updater

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.1.0", "v0.1.1", -1},
		{"v0.1.1", "0.1.1", 0},
		{"1.14.2", "1.14.0", 1},
		{"dev", "v9.9.9", -1}, // non-numeric parses as 0
	}
	for _, c := range cases {
		if got := Compare(c.a, c.b); got != c.want {
			t.Errorf("Compare(%q,%q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestAssetName(t *testing.T) {
	if got := AssetName("v0.1.1", "darwin", "arm64"); got != "vpn_0.1.1_darwin_arm64.tar.gz" {
		t.Error(got)
	}
	if got := AssetName("v0.1.1", "windows", "amd64"); got != "vpn_0.1.1_windows_amd64.zip" {
		t.Error(got)
	}
}

func TestLatestTag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.2.0"}`))
	}))
	defer srv.Close()
	old := apiBase
	apiBase = srv.URL
	defer func() { apiBase = old }()
	got, err := LatestTag(context.Background(), srv.Client(), "x/y")
	if err != nil {
		t.Fatal(err)
	}
	if got != "v0.2.0" {
		t.Fatal(got)
	}
}
