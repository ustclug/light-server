package main

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBypassHandlerMatchesAddresses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	data := "# ranges\n192.0.2.0/24\n2001:db8::/32 # inline comment\n198.51.100.10\ninvalid\n"
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path}})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		addr string
		want bool
	}{
		{addr: "192.0.2.20:443", want: true},
		{addr: "[2001:db8::1]:443", want: true},
		{addr: "198.51.100.10", want: true},
		{addr: "203.0.113.1:80", want: false},
	} {
		body := "{\"network\":\"tcp\",\"addr\":\"" + test.addr + "\",\"path\":\"/\",\"client\":\"gost\"}"
		r := httptest.NewRequest(http.MethodPost, "/bypass/geo", strings.NewReader(body))
		w := httptest.NewRecorder()
		manager.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("addr %q: status = %d, body = %s", test.addr, w.Code, w.Body.String())
		}
		var response bypassResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.OK != test.want {
			t.Errorf("addr %q: ok = %v, want %v", test.addr, response.OK, test.want)
		}
	}
}

func TestBypassHandlerWhitelistInvertsMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	if err := os.WriteFile(path, []byte("192.0.2.0/24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path, Whitelist: true}})
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		addr string
		want bool
	}{
		{addr: "192.0.2.20:443", want: false},
		{addr: "203.0.113.1:443", want: true},
	} {
		body := "{\"network\":\"tcp\",\"addr\":\"" + test.addr + "\"}"
		r := httptest.NewRequest(http.MethodPost, "/bypass/geo", strings.NewReader(body))
		w := httptest.NewRecorder()
		manager.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("addr %q: status = %d, body = %s", test.addr, w.Code, w.Body.String())
		}
		var response bypassResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if response.OK != test.want {
			t.Errorf("addr %q: ok = %v, want %v", test.addr, response.OK, test.want)
		}
	}
}

func TestBypassUsesFileWhenBothSourcesConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	if err := os.WriteFile(path, []byte("192.0.2.0/24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()

	_, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path, URL: server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	if called {
		t.Fatal("URL was requested despite file being configured")
	}
}

func TestBypassLoadsURLSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("source method = %s", r.Method)
		}
		_, _ = w.Write([]byte("203.0.113.0/24\n"))
	}))
	defer server.Close()

	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", URL: server.URL}})
	if err != nil {
		t.Fatal(err)
	}
	if !manager.items["geo"].trie.search(net.ParseIP("203.0.113.1")) {
		t.Fatal("URL source was not loaded")
	}
}

func TestBypassFailedReloadKeepsLastGoodList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	if err := os.WriteFile(path, []byte("192.0.2.0/24\n"), 0600); err != nil {
		t.Fatal(err)
	}
	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path}})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if err := manager.items["geo"].reload(context.Background()); err == nil {
		t.Fatal("reload succeeded after source removal")
	}
	if !manager.items["geo"].trie.search(net.ParseIP("192.0.2.1")) {
		t.Fatal("failed reload replaced the last good list")
	}
}

func TestBypassHandlerRejectsUnknownName(t *testing.T) {
	manager := &bypassManager{items: map[string]*bypass{}}
	w := httptest.NewRecorder()
	manager.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/bypass/missing", strings.NewReader("{}")))
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown bypass status = %d", w.Code)
	}
}
