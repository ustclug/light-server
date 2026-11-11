package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeAuthenticator struct {
	calls int
	id    string
	ok    bool
	err   error
}

func (f *fakeAuthenticator) Authenticate(context.Context, string, string) (string, bool, error) {
	f.calls++
	return f.id, f.ok, f.err
}

func TestAuthenticationResponseAndCache(t *testing.T) {
	now := time.Unix(100, 0)
	backend := &fakeAuthenticator{id: "canonical", ok: true}
	var logs []string
	h := &authHandler{
		radius: backend, ttl: 30 * time.Minute, now: func() time.Time { return now },
		cache: make(map[string]authResult),
		logf: func(format string, args ...any) {
			logs = append(logs, fmt.Sprintf(format, args...))
		},
	}
	request := func() authResponse {
		body := "{\"username\":\"alias\",\"password\":\"secret\",\"client\":\"127.0.0.1:1\",\"service\":\"plain\",\"futureKey\":true}"
		r := httptest.NewRequest(http.MethodPost, "/auth", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
		var response authResponse
		if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		return response
	}
	if got := request(); !got.OK || got.ID != "canonical" {
		t.Fatalf("response = %+v", got)
	}
	_ = request()
	if backend.calls != 1 {
		t.Fatalf("backend calls = %d, want 1", backend.calls)
	}
	if len(logs) != 1 || logs[0] != "authentication username=\"alias\" result=accepted" {
		t.Fatalf("logs after cache hit = %#v", logs)
	}
	now = now.Add(31 * time.Minute)
	_ = request()
	if backend.calls != 2 {
		t.Fatalf("backend calls after expiry = %d, want 2", backend.calls)
	}
	if len(logs) != 2 {
		t.Fatalf("log count after cache expiry = %d, want 2", len(logs))
	}
}

func TestPruneExpiredAuthenticationCache(t *testing.T) {
	now := time.Unix(200, 0)
	h := &authHandler{cache: map[string]authResult{
		"expired":  {expires: now.Add(-time.Second)},
		"boundary": {expires: now},
		"valid":    {expires: now.Add(time.Second)},
	}}

	h.pruneExpired(now)

	if _, ok := h.cache["expired"]; ok {
		t.Fatal("expired entry was not pruned")
	}
	if _, ok := h.cache["boundary"]; ok {
		t.Fatal("entry expiring at rollup time was not pruned")
	}
	if _, ok := h.cache["valid"]; !ok {
		t.Fatal("valid entry was pruned")
	}
}
