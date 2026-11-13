package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAggregateProvidedGOSTLog(t *testing.T) {
	file, err := os.Open("../inbox/gost.log")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	totals, err := aggregate(file)
	if err != nil {
		t.Fatal(err)
	}
	if got := totals["ibug"]; got != 178136 {
		t.Fatalf("inputBytes + outputBytes = %d, want 178136", got)
	}
}

func TestRotateActiveRunsPostAccountingAction(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "gost.log")
	if err := os.WriteFile(logPath, []byte("old log\n"), 0640); err != nil {
		t.Fatal(err)
	}
	now := time.Unix(123, 456)
	a := accountant{
		config: Config{LogPath: logPath, PostAccountingAction: "notify-gost --reload"},
		now:    func() time.Time { return now },
		postAction: func(_ context.Context, action string) error {
			if action != "notify-gost --reload" {
				t.Fatalf("action = %q", action)
			}
			if info, err := os.Stat(logPath); err != nil || info.Size() != 0 {
				t.Fatalf("new active log: info=%v err=%v", info, err)
			}
			return nil
		},
	}
	if err := a.rotateActive(context.Background()); err != nil {
		t.Fatal(err)
	}
	pending := logPath + ".pending.123000000456"
	data, err := os.ReadFile(pending)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old log\n" {
		t.Fatalf("pending log = %q", data)
	}
}

func TestRotateActiveSkipsEmptyPostAccountingAction(t *testing.T) {
	for _, action := range []string{"", "   "} {
		t.Run(fmt.Sprintf("action_%q", action), func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "gost.log")
			if err := os.WriteFile(logPath, []byte("old log\n"), 0640); err != nil {
				t.Fatal(err)
			}
			a := accountant{
				config: Config{LogPath: logPath, PostAccountingAction: action},
				now:    func() time.Time { return time.Unix(123, 0) },
				postAction: func(context.Context, string) error {
					t.Fatal("post accounting action was called")
					return nil
				},
			}
			if err := a.rotateActive(context.Background()); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRotateActiveKeepsRotatedFilesWhenPostAccountingActionFails(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "gost.log")
	if err := os.WriteFile(logPath, []byte("old log\n"), 0640); err != nil {
		t.Fatal(err)
	}
	a := accountant{
		config: Config{LogPath: logPath, PostAccountingAction: "false"},
		now:    func() time.Time { return time.Unix(123, 0) },
		postAction: func(context.Context, string) error {
			return errors.New("failed")
		},
	}
	if err := a.rotateActive(context.Background()); err == nil || !strings.Contains(err.Error(), "post accounting action") {
		t.Fatalf("error = %v", err)
	}
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("new active log = %q", data)
	}
	matches, err := filepath.Glob(logPath + ".pending.*")
	if err != nil || len(matches) != 1 {
		t.Fatalf("pending logs = %v, err = %v", matches, err)
	}
	data, err = os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "old log\n" {
		t.Fatalf("pending log = %q", data)
	}
}

func TestAggregateIgnoresUnrelatedAndZeroByteLines(t *testing.T) {
	log := strings.Join([]string{
		"not JSON",
		"{\"level\":\"info\",\"inputBytes\":100,\"outputBytes\":200}",
		"{\"user\":\"alice\"}",
		"{\"user\":\"alice\",\"inputBytes\":0,\"outputBytes\":0}",
		"{\"user\":\"alice\",\"inputBytes\":7}",
		"{\"user\":\"alice\",\"outputBytes\":11}",
		"{\"user\":\"bob\",\"clientID\":\"different\",\"inputBytes\":13,\"outputBytes\":17}",
	}, "\n")

	totals, err := aggregate(strings.NewReader(log))
	if err != nil {
		t.Fatal(err)
	}
	if got := totals["alice"]; got != 18 {
		t.Fatalf("alice total = %d, want 18", got)
	}
	if got := totals["bob"]; got != 30 {
		t.Fatalf("bob total = %d, want 30", got)
	}
	if len(totals) != 2 {
		t.Fatalf("totals = %#v", totals)
	}
}
