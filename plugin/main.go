package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func main() {
	configPath := flag.String("C", "/opt/plugin.yml", "configuration file")
	flag.Parse()

	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	backend := newRadiusBackend(config)
	handler := &authHandler{radius: backend, ttl: config.CacheDuration, now: time.Now, cache: make(map[string]authResult)}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	bypasses, err := newBypassManager(ctx, config.Bypasses)
	if err != nil {
		log.Fatal(err)
	}
	bypasses.start(ctx)
	server := &http.Server{Addr: config.Listen, Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth" {
			handler.ServeHTTP(w, r)
		} else if strings.HasPrefix(r.URL.Path, "/bypass/") {
			bypasses.ServeHTTP(w, r)
		} else {
			http.NotFound(w, r)
		}
	}), ReadHeaderTimeout: 5 * time.Second}
	hup := make(chan os.Signal, 1)
	signal.Notify(hup, syscall.SIGHUP)
	defer signal.Stop(hup)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-hup:
				bypasses.reload(ctx)
			}
		}
	}()
	acct := &accountant{config: config, sender: backend, now: time.Now, prune: handler.pruneExpired}
	usr1 := make(chan os.Signal, 1)
	signal.Notify(usr1, syscall.SIGUSR1)
	defer signal.Stop(usr1)
	go func() {
		if err := os.MkdirAll(filepath.Dir(config.LogPath), 0755); err != nil {
			log.Printf("initialize accounting: %v", err)
		} else if err := acct.processPending(ctx); err != nil {
			log.Printf("initial accounting: %v", err)
		}
		acct.run(ctx, usr1)
	}()
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	log.Printf("plugin listening on %s", config.Listen)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	if ctx.Err() == nil {
		os.Exit(1)
	}
}
