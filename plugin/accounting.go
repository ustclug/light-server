package main

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type gostRecord struct {
	User        string `json:"user"`
	InputBytes  uint64 `json:"inputBytes"`
	OutputBytes uint64 `json:"outputBytes"`
}

type accountingSender interface {
	Account(context.Context, string, uint64, string) error
}

func aggregate(r io.Reader) (map[string]uint64, error) {
	totals := make(map[string]uint64)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == "" {
			continue
		}
		var record gostRecord
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		if record.User == "" {
			continue
		}
		sum := record.InputBytes + record.OutputBytes
		if sum == 0 {
			continue
		}
		if sum < record.InputBytes || totals[record.User]+sum < totals[record.User] {
			return nil, fmt.Errorf("byte total overflow for %q", record.User)
		}
		totals[record.User] += sum
	}
	return totals, scanner.Err()
}

type accountant struct {
	config Config
	sender accountingSender
	now    func() time.Time
	prune  func(time.Time)
}

func (a *accountant) run(ctx context.Context) {
	for {
		now := a.now()
		timer := time.NewTimer(time.Until(now.Truncate(time.Hour).Add(time.Hour)))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if err := a.process(ctx); err != nil {
				fmt.Fprintf(os.Stderr, "accounting: %v\n", err)
			}
		}
	}
}

func (a *accountant) process(ctx context.Context) error {
	if a.prune != nil {
		a.prune(a.now())
	}
	if err := os.MkdirAll(filepath.Dir(a.config.LogPath), 0755); err != nil {
		return err
	}
	if err := a.rotateActive(ctx); err != nil {
		return err
	}
	return a.processPending(ctx)
}

func (a *accountant) processPending(ctx context.Context) error {
	files, err := filepath.Glob(a.config.LogPath + ".pending.*")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, path := range files {
		if strings.Contains(path, ".done.") {
			continue
		}
		if err := a.processFile(ctx, path); err != nil {
			return err
		}
	}
	return os.WriteFile(a.config.HealthPath, []byte(fmt.Sprintf("%d\n", a.now().Unix())), 0644)
}

func (a *accountant) rotateActive(ctx context.Context) error {
	info, err := os.Stat(a.config.LogPath)
	if os.IsNotExist(err) || (err == nil && info.Size() == 0) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := runS6(ctx, "-d", "-wD"); err != nil {
		return fmt.Errorf("stop GOST for log rotation: %w", err)
	}
	restarted := false
	defer func() {
		if !restarted {
			_ = runS6(context.Background(), "-u", "-wU")
		}
	}()
	pending := fmt.Sprintf("%s.pending.%d", a.config.LogPath, a.now().UnixNano())
	if err := os.Rename(a.config.LogPath, pending); err != nil {
		return err
	}
	file, err := os.OpenFile(a.config.LogPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0640)
	if err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := runS6(ctx, "-u", "-wU"); err != nil {
		return fmt.Errorf("restart GOST after log rotation: %w", err)
	}
	restarted = true
	return nil
}

func runS6(ctx context.Context, action, wait string) error {
	commandCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	return exec.CommandContext(commandCtx, "s6-svc", action, wait, "-T", "10000", "/etc/s6/gost").Run()
}

func (a *accountant) processFile(ctx context.Context, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	totals, parseErr := aggregate(file)
	closeErr := file.Close()
	if parseErr != nil {
		return parseErr
	}
	if closeErr != nil {
		return closeErr
	}
	stamp := filepath.Base(path)
	users := make([]string, 0, len(totals))
	for user := range totals {
		users = append(users, user)
	}
	sort.Strings(users)
	for _, user := range users {
		done := path + ".done." + safeName(user)
		if _, err := os.Stat(done); err == nil {
			continue
		}
		if err := a.sender.Account(ctx, user, totals[user], stamp+":"+user); err != nil {
			return fmt.Errorf("account %q: %w", user, err)
		}
		if err := os.WriteFile(done, nil, 0640); err != nil {
			return err
		}
	}
	for _, user := range users {
		_ = os.Remove(path + ".done." + safeName(user))
	}
	return a.archive(path)
}

func safeName(value string) string { return fmt.Sprintf("%x", []byte(value)) }

func (a *accountant) archive(path string) error {
	last := fmt.Sprintf("%s.archive.%d.gz", a.config.LogPath, a.config.Retention-1)
	_ = os.Remove(last)
	for i := a.config.Retention - 2; i >= 0; i-- {
		oldName := fmt.Sprintf("%s.archive.%d", a.config.LogPath, i)
		newName := fmt.Sprintf("%s.archive.%d", a.config.LogPath, i+1)
		if i > 0 {
			oldName += ".gz"
			newName += ".gz"
		}
		_ = os.Rename(oldName, newName)
	}
	archive := a.config.LogPath + ".archive.0"
	if err := os.Rename(path, archive); err != nil {
		return err
	}
	return compressFile(archive + ".1")
}

func compressFile(path string) error {
	source, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.Create(path + ".gz")
	if err != nil {
		return err
	}
	writer, err := gzip.NewWriterLevel(target, gzip.BestCompression)
	if err != nil {
		_ = target.Close()
		return err
	}
	_, copyErr := io.Copy(writer, source)
	closeGzipErr := writer.Close()
	closeTargetErr := target.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeGzipErr != nil {
		return closeGzipErr
	}
	if closeTargetErr != nil {
		return closeTargetErr
	}
	return os.Remove(path)
}
