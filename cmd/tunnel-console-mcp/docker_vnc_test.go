// Created by Yanjunhui
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/remote"
)

func TestDockerVNC(t *testing.T) {
	if os.Getenv("TUNNEL_CONSOLE_VNC_TEST") != "1" {
		t.Skip("set TUNNEL_CONSOLE_VNC_TEST=1 after building the desktop image")
	}
	r := newDockerInitRig(t)
	r.desktop, r.vnc = true, true
	image := os.Getenv("TUNNEL_CONSOLE_DESKTOP_IMAGE")
	if image == "" {
		image = "go-tunnel-console-desktop:latest"
	}
	r.env = append(r.env, "TUNNEL_CONSOLE_DESKTOP_IMAGE="+image)
	vncImage := os.Getenv("TUNNEL_CONSOLE_VNC_IMAGE")
	if vncImage == "" {
		vncImage = "go-tunnel-console-vnc:latest"
	}
	r.env = append(r.env, "TUNNEL_CONSOLE_VNC_IMAGE="+vncImage)
	r.writeVNCInput(t)
	r.must(t, true, "run", "--rm", "-T", "init", "--non-interactive", "--config", "/input/config.toml")
	for _, name := range []string{"vnc-password", "vnc-wrong-password"} {
		if err := os.Remove(filepath.Join(r.input, name)); err != nil {
			t.Fatal(err)
		}
	}
	r.must(t, false, "up", "-d", "--wait", "--wait-timeout", "60", "dev")
	r.must(t, false, "exec", "-T", "--user", "developer", "dev", "sh", "-c", `export DISPLAY=:1; for i in $(seq 1 100); do xdotool search --onlyvisible --class XTerm && exit 0; sleep 0.1; done; exit 1`)
	ctx, c := r.vncClient(t)
	r.must(t, false, "exec", "-T", "dev", "sh", "-c", "! command -v x11vnc && ! command -v Xtigervnc")
	checkDesktopInput(t, ctx, c, "desktop", "NATIVE_WITHOUT_VNC")
	r.must(t, false, "up", "-d", "--wait", "--wait-timeout", "60", "vnc")
	checkDesktopInput(t, ctx, c, "vnc", "VNC_WITH_NATIVE")
	checkDesktopInput(t, ctx, c, "desktop", "NATIVE_WITH_VNC")
	r.checkVNCAuthentication(t, ctx, c)
	r.must(t, false, "stop", "vnc")
	result, err := c.CallTool(ctx, &mcp.CallToolParams{Name: "vnc-screenshot", Arguments: map[string]any{}})
	if err != nil || !result.IsError {
		t.Fatal("stopped VNC unexpectedly succeeded", err)
	}
	checkDesktopInput(t, ctx, c, "desktop", "NATIVE_AFTER_VNC_STOP")
	r.checkDesktopPorts(t)
}

func checkDesktopInput(t *testing.T, ctx context.Context, c *mcp.ClientSession, backend, marker string) {
	t.Helper()
	desktopScreenshot(t, ctx, c, backend, "", 1280, 800)
	dockerCall(t, ctx, c, backend+"-pointer", map[string]any{"events": []any{map[string]any{"x": 100, "y": 100, "buttons": 1}}})
	position := desktopCommand(t, ctx, c, "DISPLAY=:1 xdotool getmouselocation --shell")
	if !strings.Contains(position, "X=100\nY=100") {
		t.Fatal("pointer not delivered", position)
	}
	dockerCall(t, ctx, c, backend+"-key", map[string]any{"keys": []string{"Control_L", "u"}})
	dockerCall(t, ctx, c, backend+"-type", map[string]any{"text": "printf '" + marker + ":中文\\n' > /workspace/" + marker + ".txt"})
	dockerCall(t, ctx, c, backend+"-key", map[string]any{"keys": []string{"Return"}})
	result := desktopCommand(t, ctx, c, "for i in $(seq 1 30); do test ! -f /workspace/"+marker+".txt || { cat /workspace/"+marker+".txt; exit 0; }; sleep 0.1; done; exit 1")
	if result != marker+":中文\n" {
		t.Fatal("keyboard did not reach terminal", result)
	}
	desktopScreenshot(t, ctx, c, backend, "", 1280, 800)
	t.Log(marker + ": screenshot, pointer, chord, Unicode typing and Enter verified")
}

func (r dockerInitRig) checkDesktopPorts(t *testing.T) {
	t.Helper()
	id := strings.TrimSpace(r.must(t, false, "ps", "-q", "dev"))
	ports, err := exec.Command("docker", "inspect", "--format", "{{len .HostConfig.PortBindings}}", id).CombinedOutput()
	if err != nil || strings.TrimSpace(string(ports)) != "0" {
		t.Fatal("desktop ports exposed", err)
	}
}

func (r dockerInitRig) writeVNCInput(t *testing.T) {
	t.Helper()
	r.writeInput(t)
	path := filepath.Join(r.input, "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, []byte("\n[profiles.local-dev.desktop]\nenabled = true\n[profiles.local-dev.vnc]\nenabled = true\nport = 5901\n")...)
	for _, profile := range []struct{ name, password string }{{"auth", "vnc-password"}, {"bad-auth", "vnc-wrong-password"}} {
		text := "\n[profiles." + profile.name + "]\nhost = 'dev'\nuser = 'developer'\nprivate_key = '/var/lib/tunnel-ssh/client_key'\n[profiles." + profile.name + ".vnc]\nenabled = true\nport = 5902\npassword = 'file:/input/" + profile.password + "'\n"
		data = append(data, []byte(text)...)
	}
	for name, content := range map[string][]byte{"config.toml": data, "vnc-password": []byte("testpass"), "vnc-wrong-password": []byte("wrongkey")} {
		if err = os.WriteFile(filepath.Join(r.input, name), content, 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func (r dockerInitRig) vncClient(t *testing.T) (context.Context, *mcp.ClientSession) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 180*time.Second)
	t.Cleanup(cancel)
	cmd := r.command(ctx, false, "run", "--rm", "--no-deps", "-T", "mcp", "serve", "--config", "/etc/tunnel-ssh/current/config.toml")
	cmd.Stderr = os.Stderr
	c, err := mcp.NewClient(&mcp.Implementation{Name: "desktop-test", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return ctx, c
}

func desktopScreenshot(t *testing.T, ctx context.Context, c *mcp.ClientSession, backend, profile string, width, height int) {
	t.Helper()
	result, err := c.CallTool(ctx, &mcp.CallToolParams{Name: backend + "-screenshot", Arguments: map[string]any{"profile": profile}})
	if err != nil || result.IsError || result.StructuredContent != nil || len(result.Content) != 2 {
		t.Fatalf("screenshot failed: %v %+v", err, result)
	}
	image, ok := result.Content[0].(*mcp.ImageContent)
	if !ok || image.MIMEType != "image/png" {
		t.Fatal("missing MCP image")
	}
	decoded, err := png.Decode(bytes.NewReader(image.Data))
	if err != nil || decoded.Bounds().Dx() != width || decoded.Bounds().Dy() != height {
		t.Fatal("incorrect PNG", err)
	}
	if path := os.Getenv("TUNNEL_CONSOLE_CAPTURE_PATH"); path != "" && profile == "" {
		if err = os.WriteFile(path, image.Data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	metadata, ok := result.Content[1].(*mcp.TextContent)
	var info struct{ Width, Height int }
	if !ok || json.Unmarshal([]byte(metadata.Text), &info) != nil || info.Width != width || info.Height != height {
		t.Fatal("screenshot metadata missing or inconsistent")
	}
	t.Logf("PNG %dx%d: %d bytes, image and text metadata present", width, height, len(image.Data))
}

func desktopCommand(t *testing.T, ctx context.Context, c *mcp.ClientSession, command string) string {
	t.Helper()
	text := dockerCall(t, ctx, c, "run-command", map[string]any{"command": command})
	var result remote.CommandResult
	if err := json.Unmarshal([]byte(text), &result); err != nil || result.ExitCode == nil || *result.ExitCode != 0 {
		t.Fatal("desktop verification command failed", err, text)
	}
	return result.Stdout.Text
}

func (r dockerInitRig) checkVNCAuthentication(t *testing.T, ctx context.Context, c *mcp.ClientSession) {
	t.Helper()
	script := `set -eu
x11vnc -storepasswd testpass /tmp/vnc-test-password >/dev/null
chmod 600 /tmp/vnc-test-password
x11vnc -display :1 -localhost -no6 -rfbport 5902 -rfbauth /tmp/vnc-test-password -forever -shared -noxdamage -noshm >/tmp/vnc-auth.log 2>&1 &
for i in $(seq 1 100); do grep -q ':170E .* 0A ' /proc/net/tcp && exit 0; sleep 0.1; done
exit 1`
	r.must(t, false, "exec", "-T", "vnc", "sh", "-c", script)
	desktopScreenshot(t, ctx, c, "vnc", "auth", 1280, 800)
	result, err := c.CallTool(ctx, &mcp.CallToolParams{Name: "vnc-screenshot", Arguments: map[string]any{"profile": "bad-auth"}})
	if err != nil || !result.IsError {
		t.Fatal("invalid VNC password accepted", err)
	}
	data, _ := json.Marshal(result)
	if strings.Contains(string(data), "wrongkey") || strings.Contains(string(data), "testpass") {
		t.Fatal("VNC password leaked")
	}
}
