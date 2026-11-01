// Created by Yanjunhui
package tunnel

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
)

func TestArgumentsKeepSecretsOutOfArgv(t *testing.T) {
	t.Setenv("CONTROL_PLANE_API_KEY", "fake-secret-must-not-leak")
	t.Setenv("CONTROL_PLANE_TUNNEL_ID", "tunnel-test")
	c := &config.Config{Tunnel: config.Tunnel{HealthAddr: "127.0.0.1:0"}}
	args, err := Arguments(c, "/path with ' quotes/program", "/configuration with spaces.toml", "run")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "fake-secret") || !strings.Contains(joined, "env:CONTROL_PLANE_API_KEY") || !strings.Contains(joined, "--mcp.command=") {
		t.Fatal("incorrect tunnel args")
	}
	c.Tunnel.HealthAddr = "0.0.0.0:8080"
	if _, err = Arguments(c, "binary", "config", "run"); err == nil {
		t.Fatal("health interface exposed")
	}
}

func TestVerifyRejectsChangedArchive(t *testing.T) {
	dir := t.TempDir()
	archive := filepath.Join(dir, "asset.zip")
	manifest := filepath.Join(dir, "SHA256SUMS.txt")
	data := []byte("release")
	if err := os.WriteFile(archive, data, 0600); err != nil {
		t.Fatal(err)
	}
	line := fmt.Sprintf("%x  asset.zip\n", sha256.Sum256(data))
	if err := os.WriteFile(manifest, []byte(line), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verify(archive, manifest, "asset.zip"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archive, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := verify(archive, manifest, "asset.zip"); err == nil {
		t.Fatal("accepted changed archive")
	}
}

func TestUnpackOfficialBundleExecutables(t *testing.T) {
	dir := t.TempDir()
	file, err := os.Create(filepath.Join(dir, "bundle.zip"))
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(file)
	for _, name := range []string{"release/tunnel-client", "release/cloudflared", "../../ignored"} {
		entry, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = entry.Write([]byte("binary")); err != nil {
			t.Fatal(err)
		}
	}
	if err = w.Close(); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	if err = unpack(file.Name(), dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"tunnel-client", "cloudflared"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil || info.Mode().Perm() != 0755 {
			t.Fatalf("extracted executable %s %v", name, err)
		}
	}
}
