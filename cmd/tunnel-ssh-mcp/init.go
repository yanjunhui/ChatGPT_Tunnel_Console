// Created by Yanjunhui
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"syscall"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/service"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/setup"
)

type initOptions struct {
	source, username, output      string
	nonInteractive, containerMode bool
	reconfigure                   bool
	uid, gid                      int
}

func initialize(ctx context.Context, args []string) error {
	o := initOptions{}
	flags := flag.NewFlagSet("init", flag.ContinueOnError)
	flags.StringVar(&o.source, "config", "", "导入已有 TOML；默认复用已安装配置，否则打开向导")
	flags.BoolVar(&o.nonInteractive, "non-interactive", false, "使用已有配置，不显示向导")
	flags.StringVar(&o.username, "service-user", "", "运行原生服务的本机账户；默认 sudo 调用者")
	flags.BoolVar(&o.containerMode, "container", false, "只生成容器持久化配置，不安装系统服务")
	flags.StringVar(&o.output, "output", "/etc/tunnel-ssh", "容器配置挂载目录；初始化与运行容器必须使用同一路径")
	flags.IntVar(&o.uid, "owner-uid", 1000, "容器运行账户 UID")
	flags.IntVar(&o.gid, "owner-gid", 1000, "容器运行账户 GID")
	flags.BoolVar(&o.reconfigure, "reconfigure", false, "容器模式重新打开配置向导")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("init 不接受额外位置参数")
	}
	if err := validateInitFlags(flags, o.containerMode); err != nil {
		return err
	}
	if o.containerMode {
		i := setup.ContainerInstaller{Directory: o.output, UID: o.uid, GID: o.gid, Reconfigure: o.reconfigure}
		return i.Init(ctx, o.source, o.nonInteractive, setup.Terminal(ctx, os.Stdin, os.Stdout))
	}
	return initializeNative(ctx, args, o)
}

func validateInitFlags(flags *flag.FlagSet, containerMode bool) error {
	var err error
	flags.Visit(func(f *flag.Flag) {
		if containerMode && f.Name == "service-user" {
			err = fmt.Errorf("--service-user 仅适用于原生安装")
		}
		if !containerMode && (f.Name == "output" || f.Name == "owner-uid" || f.Name == "owner-gid" || f.Name == "reconfigure") {
			err = fmt.Errorf("--%s 需要同时指定 --container", f.Name)
		}
	})
	return err
}

func initializeNative(ctx context.Context, args []string, o initOptions) error {
	paths, err := service.SystemPaths(runtime.GOOS)
	if err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "将写入系统安装目录 %s、%s、%s 和服务文件 %s。\n", paths.Install, paths.State, paths.Log, paths.Definition)
	if os.Geteuid() != 0 {
		return elevateInit(ctx, executable, args)
	}
	account, err := serviceAccount(o.username)
	if err != nil {
		return err
	}
	installer := setup.Installer{Manager: service.Manager{Platform: runtime.GOOS, Paths: paths, Run: service.Run}, Account: account, Executable: executable}
	return installer.Init(ctx, o.source, o.nonInteractive, setup.Terminal(ctx, os.Stdin, os.Stdout))
}

func serviceAccount(name string) (*user.User, error) {
	if name == "" {
		name = os.Getenv("SUDO_USER")
	}
	if name != "" {
		return user.Lookup(name)
	}
	return user.Current()
}

func elevateInit(ctx context.Context, executable string, args []string) error {
	fmt.Fprintln(os.Stderr, "注册开机服务需要管理员权限，接下来由 sudo 请求系统密码。")
	if err := ctx.Err(); err != nil {
		return err
	}
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return fmt.Errorf("需要 root 权限且未找到 sudo，请使用管理员账户运行 init")
	}
	return syscall.Exec(sudo, append([]string{sudo, "--", executable, "init"}, args...), os.Environ())
}
