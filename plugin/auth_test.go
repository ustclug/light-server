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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		require.Equal(t, http.StatusOK, w.Code, w.Body.String())
		var response authResponse
		require.NoError(t, json.NewDecoder(w.Body).Decode(&response))
		return response
	}
	assert.Equal(t, authResponse{OK: true, ID: "canonical"}, request())
	_ = request()
	assert.Equal(t, 1, backend.calls)
	assert.Equal(t, []string{"authentication username=\"alias\" result=accepted"}, logs)
	now = now.Add(31 * time.Minute)
	_ = request()
	assert.Equal(t, 2, backend.calls)
	assert.Len(t, logs, 2)
}

func TestPruneExpiredAuthenticationCache(t *testing.T) {
	now := time.Unix(200, 0)
	h := &authHandler{cache: map[string]authResult{
		"expired":  {expires: now.Add(-time.Second)},
		"boundary": {expires: now},
		"valid":    {expires: now.Add(time.Second)},
	}}

	h.pruneExpired(now)

	assert.NotContains(t, h.cache, "expired")
	assert.NotContains(t, h.cache, "boundary")
	assert.Contains(t, h.cache, "valid")
}
