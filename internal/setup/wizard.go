// Created by Yanjunhui
package setup

import (
	"fmt"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/config"
)

type draft struct {
	config *config.Config
	values map[string]string
}

func wizard(c *Console, account *user.User) (draft, error) {
	fmt.Fprintln(c.Output, "配置 OpenAI 隧道和 SSH 主机。初始化完成后将注册并启动系统服务。")
	d, err := tunnelQuestions(c)
	if err != nil {
		return d, err
	}
	p, err := sshQuestions(c, account, &d)
	if err != nil {
		return d, err
	}
	d.config.Profiles["server"] = p
	return d, nil
}

func tunnelQuestions(c *Console) (draft, error) {
	d := draft{config: &config.Config{DefaultProfile: "server", Profiles: make(map[string]config.Profile)}, values: make(map[string]string)}
	id, err := c.Ask("OpenAI 隧道 ID", "")
	if err != nil {
		return d, err
	}
	if id == "" {
		return d, fmt.Errorf("隧道 ID 不能为空")
	}
	key, err := c.Secret("OpenAI 运行时 API Key", true)
	if err != nil {
		return d, err
	}
	d.config.Tunnel = config.Tunnel{ID: id, APIKey: "prompt:tunnel"}
	d.values["prompt:tunnel"] = key
	return d, nil
}

func sshQuestions(c *Console, account *user.User, d *draft) (config.Profile, error) {
	var p config.Profile
	var err error
	p.Host, err = c.Ask("SSH 主机", "127.0.0.1")
	if err != nil {
		return p, err
	}
	port, err := c.Ask("SSH 端口", "22")
	if err != nil {
		return p, err
	}
	p.Port, err = strconv.Atoi(port)
	if err != nil || p.Port < 1 || p.Port > 65535 {
		return p, fmt.Errorf("SSH 端口必须为 1..65535")
	}
	p.User, err = c.Ask("SSH 用户", account.Username)
	if err != nil {
		return p, err
	}
	p.Workdir, err = c.Ask("远端工作目录（留空使用登录目录）", "")
	if err != nil {
		return p, err
	}
	auth, err := c.Ask("SSH 登录方式：1 私钥 / 2 密码", "1")
	if err != nil {
		return p, err
	}
	if auth == "1" {
		return keyQuestions(c, account, d, p)
	}
	if auth != "2" {
		return p, fmt.Errorf("登录方式必须为 1 或 2")
	}
	password, err := c.Secret("SSH 密码", true)
	if err != nil {
		return p, err
	}
	p.Password = "prompt:ssh-password"
	d.values[p.Password] = password
	return p, nil
}

func keyQuestions(c *Console, account *user.User, d *draft, p config.Profile) (config.Profile, error) {
	key, err := c.Ask("SSH 私钥文件", filepath.Join(account.HomeDir, ".ssh", "id_ed25519"))
	if err != nil {
		return p, err
	}
	if strings.HasPrefix(key, "~/") {
		key = filepath.Join(account.HomeDir, key[2:])
	}
	p.PrivateKey, err = filepath.Abs(key)
	if err != nil {
		return p, err
	}
	passphrase, err := c.Secret("私钥口令（未加密可留空）", false)
	if err != nil {
		return p, err
	}
	if passphrase != "" {
		p.Passphrase = "prompt:ssh-passphrase"
		d.values[p.Passphrase] = passphrase
	}
	return p, nil
}
