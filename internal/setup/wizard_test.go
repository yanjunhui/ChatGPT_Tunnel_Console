// Created by Yanjunhui
package setup

import (
	"bytes"
	"io"
	"os/user"
	"strings"
	"testing"
)

func TestWizardPasswordDoesNotEchoSecrets(t *testing.T) {
	var out bytes.Buffer
	secrets := []string{"test-api-secret", "test-ssh-secret"}
	c := Console{Input: strings.NewReader("tunnel_123\n\n2222\nremote-user\n/workspace\n2\n"), Output: &out, ReadSecret: func() (string, error) { s := secrets[0]; secrets = secrets[1:]; return s, nil }}
	d, err := wizard(&c, &user.User{Username: "local-user", HomeDir: "/home/local"})
	if err != nil {
		t.Fatal(err)
	}
	p := d.config.Profiles["server"]
	if p.Host != "127.0.0.1" || p.Port != 2222 || p.User != "remote-user" || p.Workdir != "/workspace" {
		t.Fatalf("wrong answers: %+v", p)
	}
	if strings.Contains(out.String(), "test-api-secret") || strings.Contains(out.String(), "test-ssh-secret") {
		t.Fatal("secret echoed")
	}
}

func TestPromptEOFAndEmptySecret(t *testing.T) {
	c := Console{Input: strings.NewReader(""), Output: io.Discard, ReadSecret: func() (string, error) { return "", nil }}
	if _, err := c.Ask("host", "default"); err == nil {
		t.Fatal("EOF silently used a default")
	}
	if _, err := c.Secret("key", true); err == nil {
		t.Fatal("accepted empty key")
	}
}

func TestWizardRejectsInvalidPort(t *testing.T) {
	for _, port := range []string{"0", "-1", "65536", "abc"} {
		c := Console{Input: strings.NewReader("tunnel_123\nlocalhost\n" + port + "\n"), Output: io.Discard, ReadSecret: func() (string, error) { return "key", nil }}
		if _, err := wizard(&c, &user.User{Username: "user"}); err == nil {
			t.Fatal("accepted port", port)
		}
	}
}
