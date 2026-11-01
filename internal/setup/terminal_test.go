// Created by Yanjunhui
package setup

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/creack/pty"
	"golang.org/x/term"
)

func terminalPair(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	master, slave, err := pty.Open()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = master.Close(); _ = slave.Close() })
	return master, slave
}

func TestHiddenInputCancellationRestoresTerminal(t *testing.T) {
	_, slave := terminalPair(t)
	before, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	_, err = terminalLine(ctx, int(slave.Fd()), true)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancellation: %v", err)
	}
	after, err := term.GetState(int(slave.Fd()))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("terminal mode was not restored")
	}
}

func TestHiddenInputEnterBackspaceAndControlC(t *testing.T) {
	for _, input := range []string{"passx\x7fword\r", "\x03"} {
		master, slave := terminalPair(t)
		ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
		timer := time.AfterFunc(50*time.Millisecond, func() { _, _ = master.Write([]byte(input)) })
		value, err := terminalLine(ctx, int(slave.Fd()), true)
		timer.Stop()
		cancel()
		if input == "\x03" {
			if !errors.Is(err, context.Canceled) {
				t.Fatal("Ctrl+C ignored", err)
			}
		} else if err != nil || value != "password" {
			t.Fatalf("password editing: %q %v", value, err)
		}
	}
}
