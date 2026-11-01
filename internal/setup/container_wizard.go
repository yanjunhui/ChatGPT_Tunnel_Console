// Created by Yanjunhui
package setup

import (
	"fmt"
	"os/user"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
)

const containerIdentity = "/var/lib/tunnel-ssh/client_key"

func containerWizard(c *Console) (draft, error) {
	fmt.Fprintln(c.Output, "配置 Docker 部署。凭据保存在挂载目录中，正式服务由 Docker Compose 启动。")
	d, err := tunnelQuestions(c)
	if err != nil {
		return d, err
	}
	target, err := c.Ask("SSH 目标：1 自带 Docker 开发环境 / 2 外部 SSH 主机 / 3 自带图形桌面", "1")
	if err != nil {
		return d, err
	}
	switch target {
	case "1", "3":
		d.config.DefaultProfile = "local-dev"
		p := config.Profile{Host: "dev", Port: 22, User: "developer", PrivateKey: containerIdentity, Workdir: "/workspace"}
		if target == "3" {
			p.VNC = &config.VNC{Enabled: true, Port: 5901}
		}
		d.config.Profiles["local-dev"] = p
	case "2":
		fmt.Fprintln(c.Output, "外部 SSH 私钥请先放入宿主机 init-input/，在此使用 /input/ 下的对应路径。")
		p, err := sshQuestions(c, &user.User{Username: "developer", HomeDir: "/input"}, &d)
		if err != nil {
			return d, err
		}
		d.config.Profiles["server"] = p
	default:
		return d, fmt.Errorf("SSH 目标必须为 1、2 或 3")
	}
	return d, nil
}

func prepareContainerProfile(d draft, p config.Profile, name, directory string, files map[string][]byte) (config.Profile, error) {
	// The dev entrypoint creates this shared identity only after initialization.
	// 中文：自带开发容器在初始化之后才生成此共享身份，因此允许暂时缺失。
	if p.Host == "dev" && (p.Port == 0 || p.Port == 22) && p.User == "developer" && p.PrivateKey == containerIdentity && p.Password == "" && p.Passphrase == "" && p.AgentSocket == "" {
		return p, nil
	}
	return prepareProfile(d, p, name, directory, files)
}

func containerStartHint(c *Console, cfg *config.Config) {
	command := "docker compose up -d --build"
	for _, p := range cfg.Profiles {
		if p.Host == "dev" && p.VNC != nil && p.VNC.Enabled && p.VNC.Port == 5901 {
			command = "docker compose -f compose.yaml -f compose.desktop.yaml up -d --build"
		}
	}
	fmt.Fprintf(c.Output, "请在宿主机执行 %s；已有容器需重启以加载新配置。初始化未启动隧道或验证远端连通性。\n", command)
}
