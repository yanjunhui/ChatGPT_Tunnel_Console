// Created by Yanjunhui
package remote

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/testssh"
)

func newTestManager(t *testing.T) (*Manager, *testssh.Server) {
	t.Helper()
	s := testssh.Start(t)
	m := New(s.Config)
	t.Cleanup(m.Close)
	return m, s
}

func TestCommandStreamsAndExit(t *testing.T) {
	m, _ := newTestManager(t)
	result, err := m.Run(t.Context(), CommandInput{Command: "printf 'first\\n'\nprintf second >&2\nexit 7"})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExitCode == nil || *result.ExitCode != 7 || result.Stdout.Text != "first\n" || result.Stderr.Text != "second" || result.Error == "" {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestCommandStdinAndDirectory(t *testing.T) {
	m, s := newTestManager(t)
	result, err := m.Run(t.Context(), CommandInput{Command: "pwd; cat", Stdin: "input\n"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Error != "" || !strings.Contains(result.Stdout.Text, s.Root) || !strings.HasSuffix(result.Stdout.Text, "input\n") {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestMissingWorkdirDoesNotExecuteScript(t *testing.T) {
	m, s := newTestManager(t)
	p := s.Config.Profiles["test"]
	p.Workdir += "/missing"
	s.Config.Profiles["test"] = p
	result, err := m.Run(t.Context(), CommandInput{Command: "echo first\necho MUST_NOT_EXECUTE"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(result.Stdout.Text, "MUST_NOT_EXECUTE") || result.Error == "" {
		t.Fatalf("script escaped failed cd: %+v", result)
	}
}

func TestCommandCancellationAndTimeout(t *testing.T) {
	m, _ := newTestManager(t)
	for _, timeout := range []bool{false, true} {
		ctx, cancel := context.WithCancel(t.Context())
		in := CommandInput{Command: "printf started; sleep 30"}
		if timeout {
			seconds := 1
			in.TimeoutSeconds = &seconds
		} else {
			time.AfterFunc(150*time.Millisecond, cancel)
		}
		start := time.Now()
		result, err := m.Run(ctx, in)
		cancel()
		if err != nil {
			t.Fatal(err)
		}
		if result.TimedOut != timeout || result.Canceled == timeout || result.ExitCode != nil || time.Since(start) > 3*time.Second {
			t.Fatalf("bad cancellation result: %+v", result)
		}
	}
}

func TestCommandValidation(t *testing.T) {
	m, _ := newTestManager(t)
	negative := -1
	huge := int(^uint(0) >> 1)
	for _, in := range []CommandInput{{Command: " "}, {Profile: "missing", Command: "true"}, {Command: "true", TimeoutSeconds: &negative}, {Command: "true", TimeoutSeconds: &huge}} {
		if _, err := m.Run(t.Context(), in); err == nil {
			t.Errorf("accepted invalid input: %+v", in)
		}
	}
}

func TestZeroTimeoutAndOutputLimit(t *testing.T) {
	m, s := newTestManager(t)
	s.Config.OutputBytes = 3
	zero := 0
	result, err := m.Run(t.Context(), CommandInput{Command: "printf abcdef", TimeoutSeconds: &zero})
	if err != nil || result.Error != "" || result.Stdout.Text != "def" || result.Stdout.DroppedBytes != 3 {
		t.Fatalf("zero timeout / bounded output: %+v %v", result, err)
	}
}

func TestBufferCursorAndTruncation(t *testing.T) {
	b := NewBuffer(5)
	_, _ = b.Write([]byte("abc"))
	_, _ = b.Write([]byte("defgh"))
	out, err := b.Read(0)
	if err != nil || out.Text != "defgh" || out.DroppedBytes != 3 || out.NextCursor != 8 {
		t.Fatalf("bad buffer: %+v %v", out, err)
	}
	out, err = b.Read(6)
	if err != nil || out.Text != "gh" || out.DroppedBytes != 0 {
		t.Fatalf("bad cursor: %+v %v", out, err)
	}
	for _, cursor := range []int64{-1, 9} {
		if _, err = b.Read(cursor); err == nil {
			t.Fatal("accepted invalid cursor")
		}
	}
}
