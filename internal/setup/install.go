// Created by Yanjunhui
// Package setup prepares persistent credentials and activates native or container installs.
// 中文：setup 包持久化凭据，并发布原生或容器安装配置。
package setup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/config"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/service"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/tunnel"
)

// Installer stages a native release before switching the active service.
// InstallTunnel may be injected for tests; nil uses the official bundle installer.
// 中文：Installer 先准备完整原生安装再切换服务；InstallTunnel 可由测试注入，
// 为空时使用官方发布包安装器。
type Installer struct {
	Manager       service.Manager
	Account       *user.User
	Executable    string
	InstallTunnel func(context.Context, string, string) error
}

// Init prompts or imports configuration, installs files, then enables the service.
// Existing settings are reused unless source names another TOML file.
// Preparation errors preserve the old install; activation errors attempt rollback
// and report any rollback failure. Success does not establish tunnel connectivity.
// 中文：Init 引导或导入配置，未指定 source 时复用已有配置；准备失败保留旧安装，
// 激活失败尝试恢复并报告恢复错误。成功不表示已验证隧道连通性。
func (i Installer) Init(ctx context.Context, source string, nonInteractive bool, console *Console) error {
	unlock, err := lockInstallation(i.Manager.Paths.Install)
	if err != nil {
		return err
	}
	defer unlock()
	d, err := i.load(source, nonInteractive, console)
	if err != nil {
		return err
	}
	definition, err := service.Render(i.Manager.Platform, i.Manager.Paths, i.Account.Username)
	if err != nil {
		return err
	}
	generation, err := i.prepare(ctx, d)
	if err != nil {
		return err
	}
	defer func() {
		target, _ := os.Readlink(i.Manager.Paths.Current())
		if target != generation {
			_ = os.RemoveAll(generation)
		}
	}()
	if err = i.activate(ctx, generation, definition); err != nil {
		return err
	}
	fmt.Fprintf(console.Output, "初始化完成：系统服务已注册、启用开机启动并启动。\n运行账户：%s\n配置：%s\n服务文件：%s\n", i.Account.Username, i.Manager.Paths.Config(), i.Manager.Paths.Definition)
	fmt.Fprintln(console.Output, "这确认了系统服务启动；OpenAI 隧道在线状态仍需通过 doctor 或 ChatGPT 实际调用验证。")
	return nil
}

func (i Installer) load(source string, nonInteractive bool, c *Console) (draft, error) {
	if source == "" {
		if _, err := os.Stat(i.Manager.Paths.Config()); err == nil {
			source = i.Manager.Paths.Config()
		} else if !os.IsNotExist(err) {
			return draft{}, err
		}
	}
	if source != "" {
		cfg, err := config.Load(source)
		if err != nil {
			return draft{}, err
		}
		fmt.Fprintf(c.Output, "使用配置：%s\n", source)
		return draft{config: cfg}, nil
	}
	if nonInteractive {
		return draft{}, fmt.Errorf("首次非交互初始化需要 --config 指定已有 TOML 配置")
	}
	return wizard(c, i.Account)
}

func (i Installer) prepare(ctx context.Context, d draft) (string, error) {
	releases := filepath.Join(i.Manager.Paths.Install, "releases")
	if err := os.MkdirAll(releases, 0755); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(releases, "install-")
	if err != nil {
		return "", err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(dir)
		}
	}()
	c, files, err := prepareConfig(d, dir)
	if err != nil {
		return "", err
	}
	if err = writeConfig(c, files, dir); err != nil {
		return "", err
	}
	if _, err = tunnel.Arguments(c, i.Executable, filepath.Join(dir, "config.toml"), "run"); err != nil {
		return "", err
	}
	if err = copyFile(i.Executable, filepath.Join(dir, "tunnel-ssh-mcp"), 0755); err != nil {
		return "", err
	}
	if err = i.prepareTunnel(ctx, d.config.Tunnel.Binary, dir); err != nil {
		return "", err
	}
	if err = i.ownership(dir); err != nil {
		return "", err
	}
	complete = true
	return dir, nil
}

func (i Installer) ownership(generation string) error {
	uid, err := strconv.Atoi(i.Account.Uid)
	if err != nil {
		return err
	}
	gid, err := strconv.Atoi(i.Account.Gid)
	if err != nil {
		return err
	}
	for _, dir := range []string{i.Manager.Paths.State, i.Manager.Paths.Log} {
		if err = os.MkdirAll(dir, 0700); err != nil {
			return err
		}
		if err = os.Chown(dir, uid, gid); err != nil {
			return err
		}
	}
	return filepath.WalkDir(generation, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(path, uid, gid)
	})
}
