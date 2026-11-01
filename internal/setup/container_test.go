// Created by Yanjunhui
package setup

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
)

func containerFixture(t *testing.T) (ContainerInstaller, *Console, *bytes.Buffer) {
	t.Helper()
	i := ContainerInstaller{Directory: filepath.Join(t.TempDir(), "data"), UID: os.Getuid(), GID: os.Getgid()}
	out := &bytes.Buffer{}
	c := &Console{Input: strings.NewReader("tunnel_test\n\n"), Output: out, ReadSecret: func() (string, error) { return "container-test-key", nil }}
	return i, c, out
}

func currentContainerConfig(t *testing.T, i ContainerInstaller) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(i.Directory, "current", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func TestContainerWizardPersistsBuiltinBeforeSSHExists(t *testing.T) {
	i, c, out := containerFixture(t)
	if err := i.Init(t.Context(), "", false, c); err != nil {
		t.Fatal(err)
	}
	cfg := currentContainerConfig(t, i)
	if cfg.DefaultProfile != "local-dev" || cfg.Profiles["local-dev"].PrivateKey != containerIdentity || cfg.Tunnel.Binary != "/usr/local/bin/tunnel-client" {
		t.Fatal("wrong container defaults")
	}
	value, err := config.Secret(cfg.Tunnel.APIKey)
	if err != nil || value != "container-test-key" {
		t.Fatal("credential did not persist", err)
	}
	target, err := os.Readlink(filepath.Join(i.Directory, "current"))
	if err != nil || filepath.IsAbs(target) {
		t.Fatal("current must resolve on both host and container", err)
	}
	for _, path := range []string{filepath.Join(i.Directory, "current", "config.toml"), strings.TrimPrefix(cfg.Tunnel.APIKey, "file:")} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0600 {
			t.Fatal("private file permissions", err)
		}
	}
	if strings.Contains(out.String(), value) || !strings.Contains(out.String(), "容器初始化完成") {
		t.Fatal("incorrect completion output")
	}
	for _, name := range []string{"tunnel-client", "tunnel-ssh-mcp"} {
		if _, err := os.Stat(filepath.Join(i.Directory, "current", name)); !os.IsNotExist(err) {
			t.Fatal("container init installed a binary", err)
		}
	}
}

func TestContainerReuseAndReconfigure(t *testing.T) {
	i, c, _ := containerFixture(t)
	if err := i.Init(t.Context(), "", false, c); err != nil {
		t.Fatal(err)
	}
	old := currentContainerConfig(t, i)
	if err := i.Init(t.Context(), "", true, c); err != nil {
		t.Fatal(err)
	}
	cfg := currentContainerConfig(t, i)
	if cfg.Tunnel.APIKey == old.Tunnel.APIKey {
		t.Fatal("credentials should be bound to a fresh generation")
	}
	i.Reconfigure = true
	c.Input = strings.NewReader("tunnel_changed\n\n")
	if err := i.Init(t.Context(), "", false, c); err != nil {
		t.Fatal(err)
	}
	if currentContainerConfig(t, i).Tunnel.ID != "tunnel_changed" {
		t.Fatal("reconfigure reused previous answers")
	}
	if _, err := config.Secret(old.Tunnel.APIKey); err != nil {
		t.Fatal("running process lost old credentials", err)
	}
}

func TestContainerImportExternalPersistsEnvironment(t *testing.T) {
	r := fixture(t, "linux")
	i, _, _ := containerFixture(t)
	if err := i.Init(t.Context(), r.source, true, r.console); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SETUP_TEST_API_KEY", "")
	t.Setenv("SETUP_TEST_SSH_PASSWORD", "")
	cfg := currentContainerConfig(t, i)
	for ref, want := range map[string]string{cfg.Tunnel.APIKey: "secret-test-api", cfg.Profiles["test"].Password: "secret-test-ssh"} {
		got, err := config.Secret(ref)
		if err != nil || got != want {
			t.Fatal("import retained external dependency", err)
		}
	}
	if len(r.calls) != 0 {
		t.Fatal("container init called service manager")
	}
}

func TestContainerInvalidImportKeepsCurrent(t *testing.T) {
	i, c, out := containerFixture(t)
	if err := i.Init(t.Context(), "", false, c); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Readlink(filepath.Join(i.Directory, "current"))
	r := fixture(t, "linux")
	t.Setenv("SETUP_TEST_API_KEY", "")
	out.Reset()
	if err := i.Init(t.Context(), r.source, true, c); err == nil {
		t.Fatal("accepted missing credential")
	}
	current, _ := os.Readlink(filepath.Join(i.Directory, "current"))
	entries, _ := os.ReadDir(filepath.Join(i.Directory, "generations"))
	if current != old || len(entries) != 1 || strings.Contains(out.String(), "初始化完成") {
		t.Fatal("failed import modified published state")
	}
}

func TestContainerFirstImportAndPublicationFailure(t *testing.T) {
	i, c, _ := containerFixture(t)
	if err := i.Init(t.Context(), "", true, c); err == nil {
		t.Fatal("first non-interactive init requires input")
	}
	current := filepath.Join(i.Directory, "current")
	if err := os.Mkdir(current, 0700); err != nil {
		t.Fatal(err)
	}
	i.Reconfigure = true
	if err := i.Init(t.Context(), "", false, c); err == nil {
		t.Fatal("accepted failed publication")
	}
	entries, err := os.ReadDir(filepath.Join(i.Directory, "generations"))
	if err != nil || len(entries) != 0 {
		t.Fatal("unpublished generation retained", err)
	}
}

func TestContainerConcurrentAndCanceledInit(t *testing.T) {
	i, c, _ := containerFixture(t)
	unlock, err := lockInstallation(i.Directory)
	if err != nil {
		t.Fatal(err)
	}
	if err = i.Init(t.Context(), "", false, c); err == nil {
		t.Fatal("concurrent init accepted")
	}
	unlock()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err = i.Init(ctx, "", false, c); err == nil {
		t.Fatal("canceled init published config")
	}
	if _, err = os.Lstat(filepath.Join(i.Directory, "current")); !os.IsNotExist(err) {
		t.Fatal("canceled init changed current", err)
	}
}

func TestContainerWizardExternalAndInvalidTarget(t *testing.T) {
	for _, target := range []string{"", "0\n", "4\n", "invalid\n"} {
		c := &Console{Input: strings.NewReader("tunnel_test\n" + target), Output: io.Discard, ReadSecret: func() (string, error) { return "key", nil }}
		if _, err := containerWizard(c); err == nil {
			t.Fatal("accepted EOF or invalid target", target)
		}
	}
	i, c, _ := containerFixture(t)
	c.Input = strings.NewReader("tunnel_test\n2\nexample.test\n22\nremote\n/tmp\n2\n")
	if err := i.Init(t.Context(), "", false, c); err != nil {
		t.Fatal(err)
	}
	p := currentContainerConfig(t, i).Profiles["server"]
	if p.Host != "example.test" || p.User != "remote" || p.Password == "" {
		t.Fatal("external answers lost")
	}
}

func TestContainerRejectsConflictingOptions(t *testing.T) {
	for _, item := range []struct {
		dir string
		uid int
		gid int
	}{{"", 0, 0}, {"/", 0, 0}, {"data", -1, 0}, {"data", 0, -1}, {"data", 2147483648, 0}} {
		i := ContainerInstaller{Directory: item.dir, UID: item.uid, GID: item.gid}
		if err := i.validate("", false); err == nil {
			t.Fatal("accepted invalid container options")
		}
	}
	i, _, _ := containerFixture(t)
	i.Reconfigure = true
	if i.validate("input.toml", false) == nil || i.validate("", true) == nil {
		t.Fatal("accepted conflicting initialization modes")
	}
}
