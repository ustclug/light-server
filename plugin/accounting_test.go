package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAggregateProvidedGOSTLog(t *testing.T) {
	file, err := os.Open("../inbox/gost.log")
	require.NoError(t, err)
	defer file.Close()
	totals, err := aggregate(file)
	require.NoError(t, err)
	assert.Equal(t, uint64(178136), totals["ibug"])
}

func TestRotateActiveRunsPostAccountingAction(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "gost.log")
	require.NoError(t, os.WriteFile(logPath, []byte("old log\n"), 0640))
	now := time.Unix(123, 456)
	a := accountant{
		config: Config{LogPath: logPath, PostAccountingAction: "notify-gost --reload"},
		now:    func() time.Time { return now },
		postAction: func(_ context.Context, action string) error {
			assert.Equal(t, "notify-gost --reload", action)
			info, err := os.Stat(logPath)
			require.NoError(t, err)
			assert.Zero(t, info.Size())
			return nil
		},
	}
	require.NoError(t, a.rotateActive(context.Background()))
	pending := logPath + ".pending.123000000456"
	data, err := os.ReadFile(pending)
	require.NoError(t, err)
	assert.Equal(t, "old log\n", string(data))
}

func TestPostAccountingActionLogging(t *testing.T) {
	for _, test := range []struct {
		name   string
		action string
		want   []string
	}{
		{
			name: "not set",
			want: []string{`"post_accounting_action" not set`},
		},
		{
			name:   "executed",
			action: "notify-gost --reload",
			want: []string{
				"executing post accounting action: notify-gost --reload",
				"executed post accounting action: status=0 duration=",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var messages []string
			a := accountant{
				config:     Config{PostAccountingAction: test.action},
				postAction: func(context.Context, string) error { return nil },
				logf: func(format string, args ...any) {
					messages = append(messages, fmt.Sprintf(format, args...))
				},
			}
			require.NoError(t, a.runPostAccountingAction(context.Background()))
			require.Len(t, messages, len(test.want))
			for i, want := range test.want {
				assert.Contains(t, messages[i], want)
			}
			if test.action != "" {
				assert.Regexp(t, `duration=\d+\.\d{3}s$`, messages[1])
			}
		})
	}
}

func TestAccountantForceRun(t *testing.T) {
	dir := t.TempDir()
	healthPath := filepath.Join(dir, "accounting.last")
	messages := make(chan string, 1)
	a := accountant{
		config: Config{
			LogPath:    filepath.Join(dir, "gost.log"),
			HealthPath: healthPath,
			Retention:  2,
		},
		now: time.Now,
		logf: func(format string, args ...any) {
			messages <- fmt.Sprintf(format, args...)
		},
	}
	force := make(chan os.Signal, 1)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		a.run(ctx, force)
	}()

	force <- syscall.SIGUSR1
	select {
	case message := <-messages:
		assert.Equal(t, "starting accounting on SIGUSR1", message)
	case <-time.After(time.Second):
		t.Fatal("accountant did not log its trigger")
	}
	require.Eventually(t, func() bool {
		_, err := os.Stat(healthPath)
		return err == nil
	}, time.Second, time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("accountant did not stop after cancellation")
	}
}

func TestRotateActiveSkipsEmptyPostAccountingAction(t *testing.T) {
	for _, action := range []string{"", "   "} {
		t.Run(fmt.Sprintf("action_%q", action), func(t *testing.T) {
			dir := t.TempDir()
			logPath := filepath.Join(dir, "gost.log")
			require.NoError(t, os.WriteFile(logPath, []byte("old log\n"), 0640))
			a := accountant{
				config: Config{LogPath: logPath, PostAccountingAction: action},
				now:    func() time.Time { return time.Unix(123, 0) },
				postAction: func(context.Context, string) error {
					assert.Fail(t, "post accounting action was called")
					return nil
				},
			}
			require.NoError(t, a.rotateActive(context.Background()))
		})
	}
}

func TestRotateActiveKeepsRotatedFilesWhenPostAccountingActionFails(t *testing.T) {
	dir := t.TempDir()
	logPath := filepath.Join(dir, "gost.log")
	require.NoError(t, os.WriteFile(logPath, []byte("old log\n"), 0640))
	a := accountant{
		config: Config{LogPath: logPath, PostAccountingAction: "false"},
		now:    func() time.Time { return time.Unix(123, 0) },
		postAction: func(context.Context, string) error {
			return errors.New("failed")
		},
	}
	assert.ErrorContains(t, a.rotateActive(context.Background()), "post accounting action")
	data, err := os.ReadFile(logPath)
	require.NoError(t, err)
	assert.Empty(t, data)
	matches, err := filepath.Glob(logPath + ".pending.*")
	require.NoError(t, err)
	require.Len(t, matches, 1)
	data, err = os.ReadFile(matches[0])
	require.NoError(t, err)
	assert.Equal(t, "old log\n", string(data))
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
	require.NoError(t, err)
	assert.Equal(t, map[string]uint64{"alice": 18, "bob": 30}, totals)
}
