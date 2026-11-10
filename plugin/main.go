package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

func main() {
	config, err := loadConfig("/opt/plugin.yml")
	if err != nil {
		log.Fatal(err)
	}
	backend := newRadiusBackend(config)
	handler := &authHandler{radius: backend, ttl: config.CacheDuration, now: time.Now, cache: make(map[string]authResult)}
	server := &http.Server{Addr: config.Listen, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/auth" {
			http.NotFound(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	}), ReadHeaderTimeout: 5 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	acct := &accountant{config: config, sender: backend, now: time.Now}
	go func() {
		if err := os.MkdirAll(filepath.Dir(config.LogPath), 0755); err != nil {
			log.Printf("initialize accounting: %v", err)
		} else if err := acct.processPending(ctx); err != nil {
			log.Printf("initial accounting: %v", err)
		}
		acct.run(ctx)
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("authentication plugin listening on %s", config.Listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	if ctx.Err() == nil {
		os.Exit(1)
	}
}
