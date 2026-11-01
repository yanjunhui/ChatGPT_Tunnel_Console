// Created by Yanjunhui
// Package tunnel installs and launches the official OpenAI tunnel-client.
// 中文：tunnel 包安装并启动 OpenAI 官方 tunnel-client。
package tunnel

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/config"
)

// Arguments builds an official tunnel-client invocation without exposing keys.
// It validates credential references and loopback diagnostics, but makes no network request.
// 中文：Arguments 构造参数时只传凭据引用，校验凭据来源及回环诊断地址，不发起网络请求。
func Arguments(c *config.Config, executable, configPath, action string) ([]string, error) {
	if action != "run" && action != "doctor" {
		return nil, fmt.Errorf("invalid tunnel action")
	}
	id := c.Tunnel.ID
	if id == "" {
		id = os.Getenv("CONTROL_PLANE_TUNNEL_ID")
	}
	if strings.TrimSpace(id) == "" {
		return nil, fmt.Errorf("set tunnel.id or CONTROL_PLANE_TUNNEL_ID")
	}
	key := c.Tunnel.APIKey
	if key == "" {
		key = "env:CONTROL_PLANE_API_KEY"
	}
	if _, err := config.Secret(key); err != nil {
		return nil, fmt.Errorf("tunnel API key: %w", err)
	}
	host, _, err := net.SplitHostPort(c.Tunnel.HealthAddr)
	if err != nil || net.ParseIP(host) == nil || !net.ParseIP(host).IsLoopback() {
		return nil, fmt.Errorf("tunnel health_addr must use a loopback IP")
	}
	command := quote(executable) + " serve --config " + quote(configPath)
	args := []string{action, "--control-plane.tunnel-id=" + id, "--control-plane.api-key=" + key,
		"--mcp.command=" + command, "--health.listen-addr=" + c.Tunnel.HealthAddr, "--log.format=json"}
	if action == "doctor" {
		args = append(args, "--explain")
	}
	return args, nil
}

func quote(s string) string { return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'" }

// Launch replaces this process, allowing signals to reach the official supervisor.
// Success does not return; lookup, validation or exec failures return an error.
// 中文：Launch 用官方进程替换当前进程，使信号直接到达监督进程；
// 成功不返回，查找、校验或进程替换失败时返回错误。
func Launch(c *config.Config, configPath, action string) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	configPath, err = filepath.Abs(configPath)
	if err != nil {
		return err
	}
	args, err := Arguments(c, executable, configPath, action)
	if err != nil {
		return err
	}
	binary, err := exec.LookPath(c.Tunnel.Binary)
	if err != nil {
		return fmt.Errorf("official tunnel-client not installed: run install-tunnel and configure tunnel.binary")
	}
	return syscall.Exec(binary, append([]string{binary}, args...), os.Environ())
}
