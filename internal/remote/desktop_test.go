// Created by Yanjunhui
package remote

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/testssh"
)

func stalledDesktop(t *testing.T) (*Manager, <-chan struct{}) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	connected := make(chan struct{})
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		close(connected)
		_, _ = io.Copy(io.Discard, conn)
	}()
	server := testssh.Start(t)
	p := server.Config.Profiles["test"]
	p.VNC = &config.VNC{Enabled: true, Host: "127.0.0.1", Port: listener.Addr().(*net.TCPAddr).Port, Timeout: "1s"}
	server.Config.Profiles["test"] = p
	m := New(server.Config)
	t.Cleanup(m.Close)
	return m, connected
}

func TestDesktopHandshakeCancellationClosesForward(t *testing.T) {
	m, connected := stalledDesktop(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, _, err := m.Screenshot(ctx, DesktopInput{}); done <- err }()
	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("no forwarded connection")
	}
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal("wrong cancellation error", err)
		}
	case <-time.After(time.Second):
		t.Fatal("canceled VNC handshake blocked")
	}
	if len(m.desktop) != 0 {
		t.Fatal("desktop gate leaked")
	}
}

func TestDesktopManagerCloseCancelsPendingRead(t *testing.T) {
	m, connected := stalledDesktop(t)
	done := make(chan error, 1)
	go func() { _, _, err := m.Screenshot(t.Context(), DesktopInput{}); done <- err }()
	select {
	case <-connected:
	case <-time.After(2 * time.Second):
		t.Fatal("no forwarded connection")
	}
	m.Close()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("manager shutdown blocked")
	}
}

func TestDesktopDeadlineAndInvalidInput(t *testing.T) {
	m, _ := stalledDesktop(t)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	if _, _, err := m.Screenshot(ctx, DesktopInput{}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal(err)
	}
	if _, err := m.Key(t.Context(), KeyInput{Keys: []string{"not-a-key"}}); err == nil {
		t.Fatal("invalid key accepted")
	}
	if _, err := m.Type(t.Context(), TextInput{Text: "\x00"}); err == nil {
		t.Fatal("invalid text accepted")
	}
}
