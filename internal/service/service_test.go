// Created by Yanjunhui
package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestLaunchdDefinition(t *testing.T) {
	p := Paths{Install: "/opt/space & <tag>", State: "/var/state", Log: "/var/log", Definition: "/Library/LaunchDaemons/test.plist"}
	data, err := Render("darwin", p, "test-user")
	if err != nil {
		t.Fatal(err)
	}
	d := xml.NewDecoder(strings.NewReader(string(data)))
	var text strings.Builder
	for {
		token, err := d.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if value, ok := token.(xml.CharData); ok {
			text.Write(value)
		}
	}
	if !strings.Contains(text.String(), p.Current()) || !strings.Contains(text.String(), "test-user") {
		t.Fatal("plist lost literal paths or service user")
	}
	if runtime.GOOS == "darwin" {
		path := filepath.Join(t.TempDir(), "test.plist")
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if out, err := exec.Command("plutil", "-lint", path).CombinedOutput(); err != nil {
			t.Fatalf("plutil: %s %v", out, err)
		}
	}
}

func TestServiceArgumentsAndFailures(t *testing.T) {
	for _, platform := range []string{"linux", "darwin"} {
		var calls []string
		p, _ := SystemPaths(platform)
		m := Manager{Platform: platform, Paths: p, Run: func(_ context.Context, name string, args ...string) (string, error) {
			calls = append(calls, strings.Join(append([]string{name}, args...), " "))
			return "loaded", nil
		}}
		if err := m.Stop(t.Context()); err != nil {
			t.Fatal(err)
		}
		if err := m.Start(t.Context()); err != nil {
			t.Fatal(err)
		}
		want := []string{"systemctl show --property=LoadState --value " + Unit, "systemctl stop " + Unit, "systemctl daemon-reload", "systemctl enable " + Unit, "systemctl start " + Unit, "systemctl is-enabled --quiet " + Unit, "systemctl is-active --quiet " + Unit}
		if platform == "darwin" {
			want = []string{"launchctl print system/" + Label, "launchctl bootout system/" + Label, "launchctl enable system/" + Label, "launchctl bootstrap system " + p.Definition, "launchctl kickstart -p system/" + Label}
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("%s commands: %v", platform, calls)
		}
		m.Run = func(context.Context, string, ...string) (string, error) { return "", fmt.Errorf("manager unavailable") }
		if err := m.Start(t.Context()); err == nil {
			t.Fatal("startup error was hidden")
		}
	}
}

func TestRenderInvalidAndSystemdEscaping(t *testing.T) {
	p, _ := SystemPaths("linux")
	p.Install = `/path with spaces/$data%q"slash`
	data, err := Render("linux", p, "root")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "$$data%%q") || !strings.Contains(string(data), "WantedBy=multi-user.target") {
		t.Fatal("missing literal escaping or boot target")
	}
	if !strings.Contains(string(data), "User=root\n") || !strings.Contains(string(data), "WorkingDirectory="+p.State+"\n") {
		t.Fatal("User or WorkingDirectory was incorrectly quoted")
	}
	for _, platform := range []string{"windows", "linux"} {
		if _, err := Render(platform, p, "bad\naccount"); err == nil {
			t.Fatal("accepted invalid service definition")
		}
	}
}
