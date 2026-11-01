// Created by Yanjunhui
package setup

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/config"
	"golang.org/x/crypto/ssh"
)

func TestEncryptedPrivateKeyIsCopiedAndValidated(t *testing.T) {
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKeyWithPassphrase(key, "init-test", []byte("test-passphrase"))
	if err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source-key")
	if err = os.WriteFile(source, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}
	d := draft{values: map[string]string{"prompt:phrase": "test-passphrase"}}
	files := make(map[string][]byte)
	destination := t.TempDir()
	p, err := prepareProfile(d, config.Profile{PrivateKey: source, Passphrase: "prompt:phrase"}, "profile", destination, files)
	if err != nil {
		t.Fatal(err)
	}
	if p.PrivateKey == source || !strings.HasPrefix(p.PrivateKey, destination) || len(files) != 2 {
		t.Fatal("key was not self-contained")
	}
	d.values["prompt:phrase"] = "incorrect"
	if _, err = prepareProfile(d, config.Profile{PrivateKey: source, Passphrase: "prompt:phrase"}, "profile", destination, files); err == nil {
		t.Fatal("accepted wrong key passphrase")
	}
}

func TestServiceRejectsEphemeralAgent(t *testing.T) {
	_, err := prepareProfile(draft{}, config.Profile{AgentSocket: "env:SSH_AUTH_SOCK"}, "profile", t.TempDir(), map[string][]byte{})
	if err == nil {
		t.Fatal("accepted login-session agent for boot service")
	}
}
