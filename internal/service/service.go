// Created by Yanjunhui
// Package service manages the native system service for the tunnel supervisor.
// 中文：service 包管理官方隧道进程的原生系统服务。
package service

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Label identifies the system-domain launchd job on macOS.
// 中文：Label 是 macOS 系统域中的 launchd 服务标识。
const Label = "com.yanjunhui.tunnel-ssh-mcp"

// Unit identifies the systemd service on Linux.
// 中文：Unit 是 Linux systemd 服务名称。
const Unit = "tunnel-ssh-mcp.service"

// Paths separates versioned installation files, runtime state, logs and the service definition.
// 中文：Paths 区分版本安装目录、运行目录、日志目录和系统服务定义。
type Paths struct{ Install, State, Log, Definition string }

// SystemPaths returns fixed paths for darwin or linux, rejecting other platforms.
// 中文：SystemPaths 返回 darwin 或 linux 的固定路径，其他平台返回错误。
func SystemPaths(platform string) (Paths, error) {
	p := Paths{Install: "/usr/local/lib/tunnel-ssh-mcp", State: "/var/lib/tunnel-ssh-mcp", Log: "/var/log/tunnel-ssh-mcp"}
	switch platform {
	case "darwin":
		p.Definition = "/Library/LaunchDaemons/" + Label + ".plist"
	case "linux":
		p.Definition = "/etc/systemd/system/" + Unit
	default:
		return Paths{}, fmt.Errorf("系统服务仅支持 macOS 和 Linux")
	}
	return p, nil
}

// Current names the symlink selecting the active installation generation.
// 中文：Current 返回指向当前安装代次的符号链接路径。
func (p Paths) Current() string { return filepath.Join(p.Install, "current") }

// Config names the configuration reached through the current generation.
// 中文：Config 返回经当前代次符号链接访问的配置路径。
func (p Paths) Config() string { return filepath.Join(p.Current(), "config.toml") }

// Runner invokes a service manager without a shell; tests inject a recorder.
// 中文：Runner 不通过 shell 调用服务管理器，测试可注入调用记录器。
type Runner func(context.Context, string, ...string) (string, error)

// Run returns combined command output and wraps nonzero exits with command context.
// Arguments must not contain credentials because errors include the invocation.
// 中文：Run 返回合并输出，并为失败附加命令上下文；参数不能含凭据，因为错误会包含参数。
func Run(ctx context.Context, name string, args ...string) (string, error) {
	data, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return string(data), fmt.Errorf("%s %s: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(data)))
	}
	return string(data), nil
}

// Manager controls one platform's system service using an injectable Runner.
// 中文：Manager 通过可注入 Runner 管理指定平台的系统服务。
type Manager struct {
	Platform string
	Paths    Paths
	Run      Runner
}

// Stop waits for the old service to stop before a new tunnel instance is started.
// A missing service is accepted; other service-manager failures are returned.
// 中文：Stop 在启动新隧道前停止旧服务；服务不存在视为成功，其他管理器错误向上传递。
func (m Manager) Stop(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m.Platform == "darwin" {
		if out, err := m.Run(ctx, "launchctl", "print", "system/"+Label); err != nil {
			if strings.Contains(strings.ToLower(out), "could not find service") {
				return nil
			}
			return err
		}
		_, err := m.Run(ctx, "launchctl", "bootout", "system/"+Label)
		return err
	}
	state, err := m.Run(ctx, "systemctl", "show", "--property=LoadState", "--value", Unit)
	if strings.TrimSpace(state) == "not-found" {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = m.Run(ctx, "systemctl", "stop", Unit)
	return err
}

// Start registers the definition, enables boot startup, and starts the process.
// A successful result confirms service-manager activation, not tunnel readiness.
// 中文：Start 注册、启用并启动服务；成功仅确认管理器操作，不能证明隧道已就绪。
func (m Manager) Start(ctx context.Context) error {
	commands := [][]string{{"systemctl", "daemon-reload"}, {"systemctl", "enable", Unit}, {"systemctl", "start", Unit}, {"systemctl", "is-enabled", "--quiet", Unit}, {"systemctl", "is-active", "--quiet", Unit}}
	if m.Platform == "darwin" {
		commands = [][]string{{"launchctl", "enable", "system/" + Label}, {"launchctl", "bootstrap", "system", m.Paths.Definition}, {"launchctl", "kickstart", "-p", "system/" + Label}}
	}
	for _, command := range commands {
		if _, err := m.Run(ctx, command[0], command[1:]...); err != nil {
			return err
		}
	}
	return nil
}

// Disable removes a failed first installation from the service manager.
// 中文：Disable 停用首次安装失败的服务，不删除配置或凭据文件。
func (m Manager) Disable(ctx context.Context) error {
	if m.Platform == "darwin" {
		if err := m.Stop(ctx); err != nil {
			return err
		}
		_, err := m.Run(ctx, "launchctl", "disable", "system/"+Label)
		return err
	}
	_, err := m.Run(ctx, "systemctl", "disable", "--now", Unit)
	return err
}
