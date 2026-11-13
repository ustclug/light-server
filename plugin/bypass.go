package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

type bypassRequest struct {
	Service string `json:"service"`
	Network string `json:"network"`
	Addr    string `json:"addr"`
	Host    string `json:"host"`
	Path    string `json:"path"`
	Client  string `json:"client"`
}

type bypassResponse struct {
	OK bool `json:"ok"`
}

type bypass struct {
	config   BypassConfig
	client   *http.Client
	reloadMu sync.Mutex
	mu       sync.RWMutex
	trie     *ipTrie
}

type bypassManager struct {
	items map[string]*bypass
}

func newBypassManager(ctx context.Context, configs []BypassConfig) (*bypassManager, error) {
	m := &bypassManager{items: make(map[string]*bypass, len(configs))}
	for _, config := range configs {
		b := &bypass{config: config, client: &http.Client{Timeout: 30 * time.Second}}
		if err := b.reload(ctx); err != nil {
			return nil, fmt.Errorf("load bypass %q: %w", config.Name, err)
		}
		m.items[config.Name] = b
	}
	return m, nil
}

func (m *bypassManager) start(ctx context.Context) {
	for _, b := range m.items {
		if b.config.Refresh > 0 {
			go b.refreshLoop(ctx)
		}
	}
}

func (m *bypassManager) reload(ctx context.Context) {
	for _, b := range m.items {
		if err := b.reload(ctx); err != nil {
			log.Printf("reload bypass %q: %v", b.config.Name, err)
		}
	}
}

func (m *bypassManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, "/bypass/")
	b, ok := m.items[name]
	if !ok || name == r.URL.Path {
		http.NotFound(w, r)
		return
	}
	b.ServeHTTP(w, r)
}

func (b *bypass) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req bypassRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10))
	if err := decoder.Decode(&req); err != nil || req.Addr == "" {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	host := req.Addr
	if parsed, _, err := net.SplitHostPort(req.Addr); err == nil {
		host = parsed
	}
	ips, err := net.DefaultResolver.LookupIP(r.Context(), "ip", host)
	if err != nil {
		log.Printf("resolve bypass host %q: %v", host, err)
		_ = json.NewEncoder(w).Encode(bypassResponse{})
		return
	}
	if b.config.PreferIPv6 {
		slices.SortStableFunc(ips, func(a, c net.IP) int {
			if (a.To4() == nil) == (c.To4() == nil) {
				return 0
			}
			if a.To4() == nil {
				return -1
			}
			return 1
		})
	}
	b.mu.RLock()
	matched := slices.ContainsFunc(ips, b.trie.search)
	b.mu.RUnlock()
	if b.config.Whitelist {
		matched = !matched
	}
	_ = json.NewEncoder(w).Encode(bypassResponse{OK: matched})
}

func (b *bypass) refreshLoop(ctx context.Context) {
	ticker := time.NewTicker(b.config.Refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := b.reload(ctx); err != nil {
				log.Printf("refresh bypass %q: %v", b.config.Name, err)
			}
		}
	}
}

func (b *bypass) reload(ctx context.Context) error {
	b.reloadMu.Lock()
	defer b.reloadMu.Unlock()

	reader, err := b.open(ctx)
	if err != nil {
		return err
	}
	defer reader.Close()

	trie := newIPTrie()
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.SplitN(scanner.Text(), "#", 2)[0])
		if line == "" {
			continue
		}
		if ip := net.ParseIP(line); ip != nil {
			bits := 128
			if ip.To4() != nil {
				bits = 32
			}
			trie.insert(ip, net.CIDRMask(bits, bits))
			continue
		}
		_, network, err := net.ParseCIDR(line)
		if err != nil {
			log.Printf("bypass %q: ignore invalid CIDR %q", b.config.Name, line)
			continue
		}
		trie.insert(network.IP, network.Mask)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	b.trie = trie
	b.mu.Unlock()
	return nil
}

func (b *bypass) open(ctx context.Context) (io.ReadCloser, error) {
	if b.config.File != "" {
		return os.Open(b.config.File)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.config.URL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", b.config.URL, resp.Status)
	}
	return resp.Body, nil
}
