package subscription

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchPlainList(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == "" {
			t.Error("missing User-Agent")
		}
		w.Write([]byte("trojan://pw@h.example:443#one\n\n# comment\nvless://11111111-2222-4333-8444-555555555555@h2.example:443#two\n"))
	}))
	defer srv.Close()
	got, err := Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d profiles", len(got))
	}
}

func TestFetchBase64List(t *testing.T) {
	body := "ss://YWVzLTI1Ni1nY206cHcx@10.0.0.1:8388#s1\ntrojan://pw@h.example:443#t1\n"
	enc := base64.StdEncoding.EncodeToString([]byte(body))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(enc))
	}))
	defer srv.Close()
	got, err := Fetch(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d profiles: %+v", len(got), got)
	}
}

func TestFetchHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()
	if _, err := Fetch(srv.URL); err == nil {
		t.Fatal("expected error")
	}
}

func TestParseContentSkipsGarbage(t *testing.T) {
	got, err := ParseContent("not-a-uri\ntrojan://pw@h.example:443#ok\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("got %+v", got)
	}
}
