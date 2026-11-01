// Created by Yanjunhui
package setup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/config"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/tunnel"
)

// ContainerInstaller writes configuration for a fixed mount shared with runtime.
// UID/GID identify the runtime account; no host service or binary is installed.
// 中文：ContainerInstaller 为初始化和运行容器的固定挂载目录写入配置；
// UID/GID 指定运行账户，不安装宿主机服务或二进制。
type ContainerInstaller struct {
	Directory   string
	UID, GID    int
	Reconfigure bool
}

// Init imports, reuses or prompts for config, then atomically publishes it.
// Validation, credential or filesystem errors leave the current generation intact.
// Old generations remain available for running processes; no tunnel is started.
// 中文：Init 导入、复用或引导配置后原子发布；失败保留当前代次，
// 旧代次供运行中进程继续使用，此操作不启动隧道。
func (i ContainerInstaller) Init(ctx context.Context, source string, nonInteractive bool, c *Console) error {
	if err := i.normalize(source, nonInteractive); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	unlock, err := lockInstallation(i.Directory)
	if err != nil {
		return err
	}
	defer unlock()
	d, err := i.load(source, nonInteractive, c)
	if err != nil {
		return err
	}
	generation, err := i.prepare(ctx, d)
	if err != nil {
		return err
	}
	published := false
	defer func() {
		if !published {
			_ = os.RemoveAll(generation)
		}
	}()
	if err = ctx.Err(); err != nil {
		return err
	}
	target := filepath.Join("generations", filepath.Base(generation))
	if err = switchCurrent(filepath.Join(i.Directory, "current"), target); err != nil {
		return fmt.Errorf("发布容器配置: %w", err)
	}
	published = true
	fmt.Fprintf(c.Output, "容器初始化完成，配置已保存：%s\n", filepath.Join(i.Directory, "current", "config.toml"))
	fmt.Fprintln(c.Output, "请在宿主机执行 docker compose up -d --build；已有容器需重启以加载新配置。初始化未启动隧道或验证远端连通性。")
	return nil
}

func (i *ContainerInstaller) normalize(source string, nonInteractive bool) error {
	if err := i.validate(source, nonInteractive); err != nil {
		return err
	}
	var err error
	i.Directory, err = filepath.Abs(i.Directory)
	if err != nil {
		return err
	}
	return i.validate(source, nonInteractive)
}

func (i ContainerInstaller) validate(source string, nonInteractive bool) error {
	if i.Directory == "" || filepath.Clean(i.Directory) == string(filepath.Separator) {
		return fmt.Errorf("--output 必须是专用配置目录，不能留空或使用根目录")
	}
	if i.UID < 0 || i.GID < 0 || int64(i.UID) > 2147483647 || int64(i.GID) > 2147483647 {
		return fmt.Errorf("容器运行账户 UID/GID 必须为 0..2147483647")
	}
	if i.Reconfigure && (source != "" || nonInteractive) {
		return fmt.Errorf("--reconfigure 不能与 --config 或 --non-interactive 同时使用")
	}
	return nil
}

func (i ContainerInstaller) load(source string, nonInteractive bool, c *Console) (draft, error) {
	if source == "" && !i.Reconfigure {
		path := filepath.Join(i.Directory, "current", "config.toml")
		if _, err := os.Stat(path); err == nil {
			source = path
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
	return containerWizard(c)
}

func (i ContainerInstaller) prepare(ctx context.Context, d draft) (string, error) {
	parent := filepath.Join(i.Directory, "generations")
	if err := os.MkdirAll(parent, 0755); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp(parent, "init-")
	if err != nil {
		return "", err
	}
	complete := false
	defer func() {
		if !complete {
			_ = os.RemoveAll(dir)
		}
	}()
	cfg, files, err := prepareConfigWith(d, dir, prepareContainerProfile)
	if err != nil {
		return "", err
	}
	cfg.Tunnel.Binary = "/usr/local/bin/tunnel-client"
	if err = writeConfig(cfg, files, dir); err != nil {
		return "", err
	}
	if _, err = tunnel.Arguments(cfg, "/usr/local/bin/tunnel-ssh-mcp", filepath.Join(dir, "config.toml"), "run"); err != nil {
		return "", err
	}
	if err = i.ownership(dir); err != nil {
		return "", err
	}
	if err = ctx.Err(); err != nil {
		return "", err
	}
	complete = true
	return dir, nil
}

func (i ContainerInstaller) ownership(generation string) error {
	for _, path := range []string{i.Directory, filepath.Dir(generation)} {
		if err := os.Chown(path, i.UID, i.GID); err != nil {
			return fmt.Errorf("设置配置目录运行账户: %w", err)
		}
		if err := os.Chmod(path, 0755); err != nil {
			return err
		}
	}
	return filepath.WalkDir(generation, func(path string, _ fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return os.Chown(path, i.UID, i.GID)
	})
}
