// Created by Yanjunhui
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestContainerCLIWithoutHostInstallation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "input.toml")
	data := `[tunnel]
id = "test_cli"
api_key = "env:TEST_CONTAINER_KEY"
[profiles.dev]
host = "dev"
user = "developer"
private_key = "/var/lib/tunnel-ssh/client_key"
`
	if err := os.WriteFile(source, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_CONTAINER_KEY", "dummy-key")
	args := []string{"init", "--container", "--non-interactive", "--config", source, "--output", filepath.Join(dir, "data"), "--owner-uid", fmt.Sprint(os.Getuid()), "--owner-gid", fmt.Sprint(os.Getgid())}
	if err := run(t.Context(), args); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "data", "current", "config.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestInitRejectsMixedInstallationFlags(t *testing.T) {
	for _, args := range [][]string{
		{"--container", "--service-user", "root"},
		{"--output", "data"}, {"--owner-uid", "1000"},
		{"--owner-gid", "1000"}, {"--reconfigure"},
		{"--container", "--reconfigure", "--non-interactive"},
		{"--container", "--output", ""},
		{"--container", "--owner-uid", "-1"},
	} {
		if err := run(t.Context(), append([]string{"init"}, args...)); err == nil {
			t.Fatal("accepted invalid init flags", args)
		}
	}
}
