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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBypassHandlerMatchesAddresses(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	data := "# ranges\n192.0.2.0/24\n2001:db8::/32 # inline comment\n198.51.100.10\ninvalid\n"
	require.NoError(t, os.WriteFile(path, []byte(data), 0600))
	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path}})
	require.NoError(t, err)

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
		require.Equal(t, http.StatusOK, w.Code, "addr %q: %s", test.addr, w.Body.String())
		var response bypassResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, test.want, response.OK, "addr %q", test.addr)
	}
}

func TestBypassHandlerWhitelistInvertsMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	require.NoError(t, os.WriteFile(path, []byte("192.0.2.0/24\n"), 0600))
	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path, Whitelist: true}})
	require.NoError(t, err)

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
		require.Equal(t, http.StatusOK, w.Code, "addr %q: %s", test.addr, w.Body.String())
		var response bypassResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		assert.Equal(t, test.want, response.OK, "addr %q", test.addr)
	}
}

func TestBypassUsesFileWhenBothSourcesConfigured(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	require.NoError(t, os.WriteFile(path, []byte("192.0.2.0/24\n"), 0600))
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer server.Close()

	_, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path, URL: server.URL}})
	require.NoError(t, err)
	assert.False(t, called, "URL was requested despite file being configured")
}

func TestBypassLoadsURLSource(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		_, _ = w.Write([]byte("203.0.113.0/24\n"))
	}))
	defer server.Close()

	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", URL: server.URL}})
	require.NoError(t, err)
	assert.True(t, manager.items["geo"].trie.search(net.ParseIP("203.0.113.1")), "URL source was not loaded")
}

func TestBypassFailedReloadKeepsLastGoodList(t *testing.T) {
	path := filepath.Join(t.TempDir(), "geo.txt")
	require.NoError(t, os.WriteFile(path, []byte("192.0.2.0/24\n"), 0600))
	manager, err := newBypassManager(context.Background(), []BypassConfig{{Name: "geo", File: path}})
	require.NoError(t, err)
	require.NoError(t, os.Remove(path))
	assert.Error(t, manager.items["geo"].reload(context.Background()))
	assert.True(t, manager.items["geo"].trie.search(net.ParseIP("192.0.2.1")), "failed reload replaced the last good list")
}

func TestBypassHandlerRejectsUnknownName(t *testing.T) {
	manager := &bypassManager{items: map[string]*bypass{}}
	w := httptest.NewRecorder()
	manager.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/bypass/missing", strings.NewReader("{}")))
	assert.Equal(t, http.StatusNotFound, w.Code)
}
