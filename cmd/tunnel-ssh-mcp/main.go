// Created by Yanjunhui
// Command tunnel-ssh-mcp installs services, runs the tunnel supervisor or serves MCP over stdio.
// 中文：tunnel-ssh-mcp 命令用于服务安装、隧道启动及 stdio MCP 服务。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/config"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/remote"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/server"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/tunnel"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "tunnel-ssh-mcp:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	if args[0] == "version" {
		fmt.Println(server.Version)
		return nil
	}
	if args[0] == "help" || args[0] == "--help" {
		usage()
		return nil
	}
	if args[0] == "install-tunnel" {
		return install(ctx, args[1:])
	}
	if args[0] == "init" {
		return initialize(ctx, args[1:])
	}
	return runConfigured(ctx, args)
}

func runConfigured(ctx context.Context, args []string) error {
	if args[0] != "run" && args[0] != "serve" && args[0] != "check" && args[0] != "doctor" {
		return fmt.Errorf("unknown command %q", args[0])
	}
	flags := flag.NewFlagSet(args[0], flag.ContinueOnError)
	path := flags.String("config", "config.toml", "TOML configuration file")
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	c, err := config.Load(*path)
	if err != nil {
		return err
	}
	if args[0] == "run" || args[0] == "doctor" {
		return tunnel.Launch(c, *path, args[0])
	}
	if args[0] == "check" {
		fmt.Printf("Configuration valid: %d SSH profile(s). Credentials and connectivity not tested.\n", len(c.Profiles))
		return nil
	}
	// Keep stdout reserved for MCP frames; diagnostics belong on stderr.
	// 中文：stdio 模式的 stdout 专用于 MCP 消息，诊断信息应写入 stderr。
	manager := remote.New(c)
	defer manager.Close()
	return server.New(manager).Run(ctx, &mcp.StdioTransport{})
}

func install(ctx context.Context, args []string) error {
	flags := flag.NewFlagSet("install-tunnel", flag.ContinueOnError)
	dir := flags.String("dir", "bin", "Installation directory")
	version := flags.String("version", tunnel.Release, "Official tunnel-client release")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected positional arguments")
	}
	return tunnel.Install(ctx, *dir, *version)
}

func usage() {
	fmt.Fprintln(os.Stderr, `OpenAI Tunnel SSH MCP (Go)
  init                          Initialize config, install and start a boot service
  init --config existing.toml    Import config and install/update the boot service
  init --container              Write persistent Docker config without a host service
  init --container --reconfigure Reopen the Docker setup wizard
  run --config config.toml       Start official OpenAI tunnel and Go MCP child
  doctor --config config.toml    Run official tunnel diagnostics
  check --config config.toml     Validate configuration without connecting
  serve --config config.toml     Internal stdio worker; also used by local tests
  install-tunnel --dir bin       Install verified official tunnel-client bundle
  version                       Print version`)
}
