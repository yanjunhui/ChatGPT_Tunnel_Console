// Created by Yanjunhui
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type dockerInitRig struct {
	root, data, input string
	project           string
	env               []string
}

func newDockerInitRig(t *testing.T) dockerInitRig {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	r := dockerInitRig{root: root, data: filepath.Join(dir, "data"), input: filepath.Join(dir, "input"), project: fmt.Sprintf("go-tunnel-ssh-init-test-%d", os.Getpid())}
	r.env = append(os.Environ(), "TUNNEL_SSH_CONFIG_DIR="+r.data, "TUNNEL_SSH_INIT_INPUT_DIR="+r.input, "TUNNEL_SSH_WORKSPACE_DIR="+filepath.Join(dir, "workspace"))
	for _, path := range []string{r.data, r.input, filepath.Join(dir, "workspace")} {
		if err = os.MkdirAll(path, 0755); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if out, err := r.command(ctx, false, "down", "-v", "--remove-orphans").CombinedOutput(); err != nil {
			t.Errorf("Docker init cleanup: %v %s", err, out)
		}
	})
	return r
}

func (r dockerInitRig) command(ctx context.Context, init bool, args ...string) *exec.Cmd {
	file := "compose.yaml"
	if init {
		file = "compose.init.yaml"
	}
	cmd := exec.CommandContext(ctx, "docker", append([]string{"compose", "-p", r.project, "-f", filepath.Join(r.root, file)}, args...)...)
	cmd.Env, cmd.Dir = r.env, r.root
	return cmd
}

func (r dockerInitRig) must(t *testing.T, init bool, args ...string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	out, err := r.command(ctx, init, args...).CombinedOutput()
	if err != nil {
		t.Fatalf("compose %v: %v %s", args, err, out)
	}
	return string(out)
}

func TestDockerContainerInitialization(t *testing.T) {
	if os.Getenv("TUNNEL_SSH_DOCKER_TEST") != "1" {
		t.Skip("set TUNNEL_SSH_DOCKER_TEST=1 after building Docker images")
	}
	r := newDockerInitRig(t)
	if out, err := r.command(t.Context(), true, "run", "--rm", "-T", "init", "--non-interactive").CombinedOutput(); err == nil || !strings.Contains(string(out), "首次非交互初始化") {
		t.Fatalf("unconfigured init should fail: %v %s", err, out)
	}
	r.writeInput(t)
	out := r.must(t, true, "run", "--rm", "-T", "init", "--non-interactive", "--config", "/input/config.toml")
	if strings.Contains(out, "docker-test-secret") || !strings.Contains(out, "容器初始化完成") {
		t.Fatal("unexpected init output")
	}
	r.checkPersistentConfiguration(t)
	r.must(t, false, "up", "-d", "--wait", "--wait-timeout", "60", "dev")
	r.checkWorker(t)
	// A second container reads persisted credentials after the input was removed.
	// 中文：移除导入文件后，用第二个容器验证运行已不再依赖输入凭据文件。
	r.must(t, true, "run", "--rm", "-T", "init", "--non-interactive")
	r.checkWorker(t)
	r.checkWizard(t)
	r.checkWorker(t)
}

func (r dockerInitRig) writeInput(t *testing.T) {
	t.Helper()
	data := `# Created by Yanjunhui
default_profile = "local-dev"
[tunnel]
id = "tunnel_docker_test"
api_key = "file:/input/openai-tunnel-key"
[profiles.local-dev]
host = "dev"
port = 22
user = "developer"
private_key = "/var/lib/tunnel-ssh/client_key"
workdir = "/workspace"
`
	for name, contents := range map[string]string{"config.toml": data, "openai-tunnel-key": "docker-test-secret"} {
		if err := os.WriteFile(filepath.Join(r.input, name), []byte(contents), 0600); err != nil {
			t.Fatal(err)
		}
	}
}

func (r dockerInitRig) checkPersistentConfiguration(t *testing.T) {
	t.Helper()
	if err := os.Remove(filepath.Join(r.input, "openai-tunnel-key")); err != nil {
		t.Fatal(err)
	}
	r.must(t, false, "run", "--rm", "--no-deps", "-T", "mcp", "check", "--config", "/etc/tunnel-ssh/current/config.toml")
	script := `set -eu
test "$(id -u)" = 1000
test "$(id -g)" = 1000
for f in /etc/tunnel-ssh/current/config.toml /etc/tunnel-ssh/current/secrets/tunnel-key; do
  test -r "$f"
  test "$(stat -c '%u:%g:%a' "$f")" = '1000:1000:600'
done
if touch /etc/tunnel-ssh/write-test 2>/dev/null; then exit 1; fi
`
	r.must(t, false, "run", "--rm", "--no-deps", "-T", "--entrypoint", "/bin/sh", "mcp", "-c", script)
	target, err := os.Readlink(filepath.Join(r.data, "current"))
	if err != nil || filepath.IsAbs(target) {
		t.Fatal("host cannot resolve config generation", err)
	}
	if _, err := os.Stat(filepath.Join(r.data, "current", "config.toml")); err != nil {
		t.Fatal("host configuration missing", err)
	}
}

func (r dockerInitRig) checkWorker(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer cancel()
	cmd := r.command(ctx, false, "run", "--rm", "--no-deps", "-T", "mcp", "serve", "--config", "/etc/tunnel-ssh/current/config.toml")
	cmd.Stderr = os.Stderr
	c, err := mcp.NewClient(&mcp.Implementation{Name: "docker-init-test", Version: "1"}, nil).Connect(ctx, &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	tools, err := c.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 13 {
		t.Fatalf("tool discovery: %+v %v", tools, err)
	}
	dockerCheckCommand(t, ctx, c)
	dockerCheckFiles(t, ctx, c)
	dockerCheckTerminal(t, ctx, c)
}
