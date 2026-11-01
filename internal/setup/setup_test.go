// Created by Yanjunhui
package setup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/service"
)

type rig struct {
	installer Installer
	source    string
	console   *Console
	output    bytes.Buffer
	calls     []string
	fail      string
}

func fixture(t *testing.T, platform string) *rig {
	t.Helper()
	dir := t.TempDir()
	account, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	r := &rig{}
	r.installer = Installer{Manager: service.Manager{Platform: platform, Paths: service.Paths{Install: filepath.Join(dir, "install"), State: filepath.Join(dir, "state"), Log: filepath.Join(dir, "logs"), Definition: filepath.Join(dir, "service", "definition")}}, Account: account, Executable: filepath.Join(dir, "binary")}
	r.installer.Manager.Run = func(_ context.Context, name string, args ...string) (string, error) {
		call := strings.Join(append([]string{name}, args...), " ")
		r.calls = append(r.calls, call)
		if r.fail != "" && strings.Contains(call, r.fail) {
			r.fail = ""
			return "", fmt.Errorf("injected activation failure")
		}
		return "loaded", nil
	}
	r.installer.InstallTunnel = func(_ context.Context, path, _ string) error {
		return os.WriteFile(filepath.Join(path, "tunnel-client"), []byte("test tunnel executable"), 0755)
	}
	if err = os.WriteFile(r.installer.Executable, []byte("test service executable"), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SETUP_TEST_API_KEY", "secret-test-api")
	t.Setenv("SETUP_TEST_SSH_PASSWORD", "secret-test-ssh")
	c := config.Config{DefaultProfile: "test", Tunnel: config.Tunnel{ID: "tunnel_test", APIKey: "env:SETUP_TEST_API_KEY", Binary: filepath.Join(dir, "not-installed")}, Profiles: map[string]config.Profile{"test": {Host: "localhost", User: "test", Password: "env:SETUP_TEST_SSH_PASSWORD"}}}
	data, err := toml.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	r.source = filepath.Join(dir, "input.toml")
	if err = os.WriteFile(r.source, data, 0600); err != nil {
		t.Fatal(err)
	}
	r.console = &Console{Output: &r.output}
	return r
}

func TestImportPersistsCredentialsAndRegistersService(t *testing.T) {
	for _, platform := range []string{"darwin", "linux"} {
		t.Run(platform, func(t *testing.T) {
			r := fixture(t, platform)
			if err := r.installer.Init(t.Context(), r.source, true, r.console); err != nil {
				t.Fatal(err)
			}
			cfg, err := config.Load(r.installer.Manager.Paths.Config())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("SETUP_TEST_API_KEY", "")
			t.Setenv("SETUP_TEST_SSH_PASSWORD", "")
			for ref, want := range map[string]string{cfg.Tunnel.APIKey: "secret-test-api", cfg.Profiles["test"].Password: "secret-test-ssh"} {
				value, err := config.Secret(ref)
				if err != nil || value != want {
					t.Fatal("service depends on terminal environment", err)
				}
				info, err := os.Stat(strings.TrimPrefix(ref, "file:"))
				if err != nil || info.Mode().Perm() != 0600 {
					t.Fatal("credential permissions", err)
				}
			}
			data, _ := os.ReadFile(r.installer.Manager.Paths.Definition)
			if strings.Contains(string(data), "secret-test") || strings.Contains(r.output.String(), "secret-test") {
				t.Fatal("secret leaked")
			}
			if !strings.Contains(r.output.String(), "初始化完成") {
				t.Fatal("completion missing")
			}
		})
	}
}

func TestReinitializeReusesConfigAndRestoresOnFailure(t *testing.T) {
	r := fixture(t, "linux")
	if err := r.installer.Init(t.Context(), r.source, true, r.console); err != nil {
		t.Fatal(err)
	}
	old, _ := os.Readlink(r.installer.Manager.Paths.Current())
	r.output.Reset()
	r.fail = "systemctl start"
	err := r.installer.Init(t.Context(), "", true, r.console)
	if err == nil || !strings.Contains(err.Error(), "已恢复") || strings.Contains(r.output.String(), "初始化完成") {
		t.Fatalf("failure result: %v %s", err, r.output.String())
	}
	current, _ := os.Readlink(r.installer.Manager.Paths.Current())
	if current != old {
		t.Fatal("old generation not restored")
	}
	if err = r.installer.Init(t.Context(), "", true, r.console); err != nil {
		t.Fatal(err)
	}
	current, _ = os.Readlink(r.installer.Manager.Paths.Current())
	if current == old {
		t.Fatal("repeat init did not update binary generation")
	}
}

func TestFailedFirstInstallationRemovesActivation(t *testing.T) {
	r := fixture(t, "darwin")
	r.fail = "launchctl bootstrap"
	if err := r.installer.Init(t.Context(), r.source, true, r.console); err == nil {
		t.Fatal("accepted startup failure")
	}
	for _, path := range []string{r.installer.Manager.Paths.Current(), r.installer.Manager.Paths.Definition} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("partial activation retained: %s %v", path, err)
		}
	}
}

func TestInvalidCredentialDoesNotStopExistingService(t *testing.T) {
	r := fixture(t, "linux")
	t.Setenv("SETUP_TEST_API_KEY", "")
	if err := r.installer.Init(t.Context(), r.source, true, r.console); err == nil {
		t.Fatal("accepted missing key")
	}
	if len(r.calls) != 0 {
		t.Fatal("service was touched before configuration was ready", r.calls)
	}
	if err := r.installer.Init(t.Context(), "", true, r.console); err == nil {
		t.Fatal("non-interactive first install did not require config")
	}
}

func TestConcurrentInitCannotSwitchServiceTwice(t *testing.T) {
	r := fixture(t, "linux")
	unlock, err := lockInstallation(r.installer.Manager.Paths.Install)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if err = r.installer.Init(t.Context(), r.source, true, r.console); err == nil {
		t.Fatal("concurrent initialization accepted")
	}
	if len(r.calls) != 0 {
		t.Fatal("concurrent init modified service")
	}
}
