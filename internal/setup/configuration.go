// Created by Yanjunhui
package setup

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"golang.org/x/crypto/ssh"
)

func (d draft) secret(ref string) (string, error) {
	if value, ok := d.values[ref]; ok {
		return value, nil
	}
	return config.Secret(ref)
}

func prepareConfig(d draft, directory string) (*config.Config, map[string][]byte, error) {
	return prepareConfigWith(d, directory, prepareProfile)
}

type profilePreparer func(draft, config.Profile, string, string, map[string][]byte) (config.Profile, error)

func prepareConfigWith(d draft, directory string, prepare profilePreparer) (*config.Config, map[string][]byte, error) {
	c := *d.config
	c.Profiles = make(map[string]config.Profile)
	files := make(map[string][]byte)
	if c.Tunnel.ID == "" {
		c.Tunnel.ID = os.Getenv("CONTROL_PLANE_TUNNEL_ID")
	}
	if c.Tunnel.ID == "" {
		return nil, nil, fmt.Errorf("隧道 ID 不能为空")
	}
	if c.Tunnel.APIKey == "" {
		c.Tunnel.APIKey = "env:CONTROL_PLANE_API_KEY"
	}
	ref, err := captureSecret(d, c.Tunnel.APIKey, "tunnel-key", directory, files)
	if err != nil {
		return nil, nil, fmt.Errorf("隧道凭据: %w", err)
	}
	c.Tunnel.APIKey = ref
	for name, p := range d.config.Profiles {
		p, err = prepare(d, p, fmt.Sprintf("ssh-%x", sha256.Sum256([]byte(name))), directory, files)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH 配置 %q: %w", name, err)
		}
		p, err = prepareDesktop(d, p, fmt.Sprintf("vnc-%x", sha256.Sum256([]byte(name))), directory, files)
		if err != nil {
			return nil, nil, fmt.Errorf("VNC 配置 %q: %w", name, err)
		}
		c.Profiles[name] = p
	}
	c.Tunnel.Binary = filepath.Join(directory, "tunnel-client")
	if err = c.Validate(directory); err != nil {
		return nil, nil, err
	}
	return &c, files, nil
}

func captureSecret(d draft, ref, name, directory string, files map[string][]byte) (string, error) {
	if ref == "" {
		return "", nil
	}
	value, err := d.secret(ref)
	if err != nil {
		return "", err
	}
	if value == "" {
		return "", fmt.Errorf("凭据不能为空")
	}
	// Pin references to this generation so running processes never mix credentials
	// from different releases when the current symlink changes.
	// 中文：凭据引用固定到具体代次，避免 current 切换后运行进程混用新旧凭据。
	path := filepath.Join(directory, "secrets", name)
	files[path] = []byte(value)
	return "file:" + path, nil
}

func prepareProfile(d draft, p config.Profile, name, directory string, files map[string][]byte) (config.Profile, error) {
	if p.AgentSocket != "" {
		return p, fmt.Errorf("开机服务不能依赖登录会话的 SSH agent；请改用私钥文件或密码")
	}
	passphrase, err := d.secret(p.Passphrase)
	if err != nil {
		return p, err
	}
	if p.PrivateKey != "" {
		data, err := os.ReadFile(p.PrivateKey)
		if err != nil {
			return p, fmt.Errorf("读取私钥: %w", err)
		}
		if passphrase != "" {
			_, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
		} else {
			_, err = ssh.ParsePrivateKey(data)
		}
		if err != nil {
			return p, fmt.Errorf("无法解析 SSH 私钥，请检查格式与口令")
		}
		p.PrivateKey = filepath.Join(directory, "secrets", name+"-key")
		files[p.PrivateKey] = data
	}
	p.Password, err = captureSecret(d, p.Password, name+"-password", directory, files)
	if err != nil {
		return p, err
	}
	p.Passphrase, err = captureSecret(d, p.Passphrase, name+"-passphrase", directory, files)
	return p, err
}

func writeConfig(c *config.Config, files map[string][]byte, directory string) error {
	if err := os.MkdirAll(filepath.Join(directory, "secrets"), 0700); err != nil {
		return err
	}
	for path, data := range files {
		if err := os.WriteFile(path, data, 0600); err != nil {
			return err
		}
	}
	data, err := toml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(directory, "config.toml"), append([]byte("# Created by Yanjunhui\n"), data...), 0600)
}
