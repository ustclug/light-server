package main

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

type authRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Client   string `json:"client"`
	Service  string `json:"service"`
}
type authResponse struct {
	OK bool   `json:"ok"`
	ID string `json:"id,omitempty"`
}
type authResult struct {
	response authResponse
	expires  time.Time
}
type radiusAuthenticator interface {
	Authenticate(context.Context, string, string) (string, bool, error)
}
type authHandler struct {
	radius radiusAuthenticator
	ttl    time.Duration
	now    func() time.Time
	mu     sync.Mutex
	cache  map[string]authResult
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req authRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil || req.Username == "" || req.Password == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	key := req.Username + "\x00" + req.Password
	h.mu.Lock()
	cached, ok := h.cache[key]
	if ok && h.now().Before(cached.expires) {
		h.mu.Unlock()
		_ = json.NewEncoder(w).Encode(cached.response)
		return
	}
	delete(h.cache, key)
	h.mu.Unlock()
	id, accepted, err := h.radius.Authenticate(r.Context(), req.Username, req.Password)
	if err != nil {
		http.Error(w, "authentication backend unavailable", http.StatusServiceUnavailable)
		return
	}
	response := authResponse{OK: accepted}
	if accepted {
		if id == "" {
			id = req.Username
		}
		response.ID = id
	}
	h.mu.Lock()
	h.cache[key] = authResult{response: response, expires: h.now().Add(h.ttl)}
	h.mu.Unlock()
	_ = json.NewEncoder(w).Encode(response)
}
