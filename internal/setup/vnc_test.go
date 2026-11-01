// Created by Yanjunhui
package setup

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
)

func TestBuiltInVNCSecretCapturedWithoutMutatingImport(t *testing.T) {
	t.Setenv("VNC_TEST_PASSWORD", "testpass")
	d := draft{config: &config.Config{Tunnel: config.Tunnel{ID: "test", APIKey: "prompt:key"}, Profiles: map[string]config.Profile{
		"dev": {Host: "dev", User: "developer", PrivateKey: containerIdentity, Desktop: &config.Desktop{Enabled: true}, VNC: &config.VNC{Enabled: true, Password: "env:VNC_TEST_PASSWORD"}},
	}}, values: map[string]string{"prompt:key": "test-key"}}
	c, files, err := prepareConfigWith(d, t.TempDir(), prepareContainerProfile)
	if err != nil {
		t.Fatal(err)
	}
	ref := c.Profiles["dev"].VNC.Password
	if !strings.HasPrefix(ref, "file:") || string(files[strings.TrimPrefix(ref, "file:")]) != "testpass" {
		t.Fatal("VNC credential not captured")
	}
	if d.config.Profiles["dev"].VNC.Password != "env:VNC_TEST_PASSWORD" {
		t.Fatal("original import mutated")
	}
	if d.config.Profiles["dev"].Desktop.Display != "" || c.Profiles["dev"].Desktop.Display != ":1" {
		t.Fatal("native display import mutated or default missing")
	}
}

func TestContainerDesktopWizard(t *testing.T) {
	var output bytes.Buffer
	c := &Console{Input: strings.NewReader("test-tunnel\n3\ny\n"), Output: &output, ReadSecret: func() (string, error) { return "test-key", nil }}
	d, err := containerWizard(c)
	if err != nil {
		t.Fatal(err)
	}
	p := d.config.Profiles[d.config.DefaultProfile]
	if p.Host != "dev" || p.Desktop == nil || !p.Desktop.Enabled || p.VNC == nil || !p.VNC.Enabled || p.VNC.Port != 5901 {
		t.Fatalf("unexpected desktop profile: %+v", p)
	}
	containerStartHint(c, d.config)
	if !strings.Contains(output.String(), "-f compose.desktop.yaml -f compose.vnc.yaml") || strings.Contains(output.String(), "test-key") {
		t.Fatal("missing desktop command or leaked secret")
	}
}

func TestContainerNativeDesktopWithoutVNC(t *testing.T) {
	var output bytes.Buffer
	c := &Console{Input: strings.NewReader("test-tunnel\n3\n\n"), Output: &output, ReadSecret: func() (string, error) { return "test-key", nil }}
	d, err := containerWizard(c)
	if err != nil {
		t.Fatal(err)
	}
	p := d.config.Profiles[d.config.DefaultProfile]
	if p.Desktop == nil || !p.Desktop.Enabled || p.VNC != nil {
		t.Fatal("native desktop must not require VNC")
	}
	containerStartHint(c, d.config)
	if !strings.Contains(output.String(), "compose.desktop.yaml") || strings.Contains(output.String(), "compose.vnc.yaml") {
		t.Fatal("incorrect start command")
	}
}
