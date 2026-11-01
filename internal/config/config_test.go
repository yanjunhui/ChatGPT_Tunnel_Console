// Created by Yanjunhui
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func loadText(t *testing.T, text string) (*Config, error) {
	t.Helper()
	name := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(name, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	return Load(name)
}

const valid = `[profiles.test]
host = "localhost"
user = "user"
private_key = "keys/id"
`

func TestDefaultsAndRelativePaths(t *testing.T) {
	c, err := loadText(t, valid)
	if err != nil {
		t.Fatal(err)
	}
	p := c.Profiles["test"]
	if c.DefaultProfile != "test" || p.Port != 22 || c.OutputBytes != 1048576 || !filepath.IsAbs(p.PrivateKey) || p.CommandTimeout != "60s" {
		t.Fatalf("bad defaults %+v %+v", c, p)
	}
}

func TestInvalidConfiguration(t *testing.T) {
	for _, text := range []string{"", valid + "port = -1\n", valid + "port = 65536\n", valid + "command_timeout = '-1s'\n", valid + "role = 'admin'\n", "output_bytes = -1\n" + valid, "output_bytes = 67108865\n" + valid, "default_profile = 'missing'\n" + valid, strings.Replace(valid, "user = \"user\"", "user = \"\"", 1)} {
		if _, err := loadText(t, text); err == nil {
			t.Errorf("accepted invalid configuration: %s", text)
		}
	}
}

func TestSecretResolution(t *testing.T) {
	t.Setenv("TUNNEL_TEST_SECRET", " test value ")
	value, err := Secret("env:TUNNEL_TEST_SECRET")
	if err != nil || value != " test value " {
		t.Fatal("env resolution", err)
	}
	name := filepath.Join(t.TempDir(), "key")
	if err = os.WriteFile(name, []byte(" test value \r\n"), 0600); err != nil {
		t.Fatal(err)
	}
	value, err = Secret("file:" + name)
	if err != nil || value != " test value " {
		t.Fatal("file resolution", err)
	}
	for _, ref := range []string{"literal-sensitive-value", "env:MISSING_TUNNEL_TEST", "file:/missing-file"} {
		if _, err = Secret(ref); err == nil || strings.Contains(err.Error(), "literal-sensitive-value") {
			t.Fatal("bad credential error", err)
		}
	}
}
