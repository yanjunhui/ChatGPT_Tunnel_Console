// Created by Yanjunhui
package main

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/remote"
)

func TestDockerOpenSSH(t *testing.T) {
	if os.Getenv("TUNNEL_SSH_DOCKER_TEST") != "1" {
		t.Skip("set TUNNEL_SSH_DOCKER_TEST=1 after building Docker images")
	}
	workspace := t.TempDir()
	prepareDockerFixture(t, workspace, workspace)
	t.Setenv("TUNNEL_SSH_TEST_WORKSPACE_DIR", workspace)
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	compose := filepath.Join(root, "deploy/compose.test.yaml")
	docker := func(args ...string) *exec.Cmd {
		return exec.Command("docker", append([]string{"compose", "-f", compose}, args...)...)
	}
	t.Cleanup(func() {
		out, err := docker("down", "-v", "--remove-orphans").CombinedOutput()
		if err != nil {
			t.Errorf("test container cleanup: %v %s", err, out)
		}
	})
	if out, err := docker("up", "-d", "--wait", "--wait-timeout", "60", "dev").CombinedOutput(); err != nil {
		t.Fatalf("start dev: %v %s", err, out)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := docker("run", "--rm", "--no-deps", "-T", "mcp")
	cmd.Stderr = os.Stderr
	c, err := mcp.NewClient(&mcp.Implementation{Name: "docker-test", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	dockerCheckCommand(t, ctx, c)
	dockerCheckFiles(t, ctx, c)
	dockerCheckTerminal(t, ctx, c)
}

func dockerCall(t *testing.T, ctx context.Context, c *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	result, err := c.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	text := result.Content[0].(*mcp.TextContent).Text
	if result.IsError {
		t.Fatalf("%s: %s", name, text)
	}
	return text
}

func dockerCheckCommand(t *testing.T, ctx context.Context, c *mcp.ClientSession) {
	t.Helper()
	text := dockerCall(t, ctx, c, "run-command", map[string]any{"command": "set -e\nid -un\npwd\nsudo -n id -u\ngo version\npython3 --version\nnode --version\nnpm --version\nfd --version"})
	var result remote.CommandResult
	if err := json.Unmarshal([]byte(text), &result); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Stdout.Text, "developer\n/workspace\n0\n") || !strings.Contains(result.Stdout.Text, "go1.26") {
		t.Fatal(result.Stdout.Text)
	}
	t.Log(strings.TrimSpace(result.Stdout.Text))
}

func dockerCheckFiles(t *testing.T, ctx context.Context, c *mcp.ClientSession) {
	t.Helper()
	dockerCall(t, ctx, c, "sftp-upload", map[string]any{"path": "acceptance.txt", "content": "Docker 往返传输\n"})
	dockerCall(t, ctx, c, "sftp-download-file", map[string]any{"path": "acceptance.txt", "local_path": "/workspace/acceptance-copy.txt"})
	dockerCall(t, ctx, c, "sftp-upload-file", map[string]any{"path": "acceptance-upload.txt", "local_path": "/workspace/acceptance-copy.txt"})
	for _, path := range []string{"acceptance.txt", "acceptance-copy.txt", "acceptance-upload.txt"} {
		dockerCheckDownload(t, ctx, c, path, "Docker 往返传输\n")
	}
	dockerCall(t, ctx, c, "run-command", map[string]any{"command": "rm -f /workspace/acceptance.txt /workspace/acceptance-copy.txt /workspace/acceptance-upload.txt"})
}

func dockerCheckTerminal(t *testing.T, ctx context.Context, c *mcp.ClientSession) {
	t.Helper()
	dockerCall(t, ctx, c, "open-session", map[string]any{"name": "docker-pty"})
	dockerCall(t, ctx, c, "write-session-input", map[string]any{"name": "docker-pty", "data": "export TERM_TEST=kept\ncd /tmp\n"})
	dockerCall(t, ctx, c, "write-session-input", map[string]any{"name": "docker-pty", "data": "printf 'PTY:%s:%s:END\\n' \"$TERM_TEST\" \"$PWD\"\n"})
	for deadline := time.Now().Add(5 * time.Second); time.Now().Before(deadline); {
		text := dockerCall(t, ctx, c, "read-session-output", map[string]any{"name": "docker-pty"})
		if strings.Contains(text, "PTY:kept:/tmp:END") {
			dockerCall(t, ctx, c, "close-session", map[string]any{"name": "docker-pty"})
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal("real OpenSSH PTY did not preserve shell state")
}
