// Created by Yanjunhui
package remote

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/testssh"
	"golang.org/x/crypto/ssh"
)

func nativeManager(t *testing.T) *Manager {
	t.Helper()
	f := testssh.Start(t)
	p := f.Config.Profiles["test"]
	p.Desktop = &config.Desktop{Enabled: true, Display: ":1", Timeout: "2s"}
	f.Config.Profiles["test"] = p
	m := New(f.Config)
	t.Cleanup(m.Close)
	return m
}

func TestNativeDesktopCancellation(t *testing.T) {
	for _, closeManager := range []bool{false, true} {
		m := nativeManager(t)
		ctx, cancel := context.WithCancel(t.Context())
		started, done := make(chan struct{}), make(chan error, 1)
		go func() {
			done <- m.withX11(ctx, "", func(c *ssh.Client, display string) error {
				return runX11(c, display, "printf x; sleep 10", "", func(r io.Reader) error {
					var b [1]byte
					if _, err := io.ReadFull(r, b[:]); err != nil {
						return err
					}
					close(started)
					_, err := io.Copy(io.Discard, r)
					return err
				})
			})
		}()
		select {
		case <-started:
		case <-time.After(time.Second):
			t.Fatal("SSH process did not start")
		}
		if closeManager {
			m.Close()
		} else {
			cancel()
		}
		select {
		case err := <-done:
			if !errors.Is(err, context.Canceled) || len(m.desktop) != 0 {
				t.Fatal("cancellation or gate release failed", err)
			}
		case <-time.After(time.Second):
			t.Fatal("native desktop cancellation blocked")
		}
		cancel()
	}
}

func TestNativeDesktopRejectsInputBeforeConnecting(t *testing.T) {
	m := nativeManager(t)
	if _, err := m.DesktopKey(t.Context(), KeyInput{Keys: []string{"a;id"}}); err == nil {
		t.Fatal("invalid chord accepted")
	}
	if _, err := m.DesktopType(t.Context(), TextInput{Text: "\x00"}); err == nil {
		t.Fatal("invalid text accepted")
	}
	if _, err := m.DesktopPointer(t.Context(), PointerInput{Events: []desktop.PointerEvent{{X: -1}}}); err == nil {
		t.Fatal("negative coordinate accepted")
	}
	m.desktop <- struct{}{}
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Millisecond)
	defer cancel()
	if _, _, err := m.DesktopScreenshot(ctx, DesktopInput{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("blocked gate ignored deadline", err)
	}
	<-m.desktop
}
