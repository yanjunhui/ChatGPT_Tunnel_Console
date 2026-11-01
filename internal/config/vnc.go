// Created by Yanjunhui
package config

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// VNC describes a desktop reached from the SSH host, never a public MCP listener.
// Password is an optional env:/file: reference for classic VNC authentication.
// 中文：VNC 描述从 SSH 主机访问的桌面，不创建公网 MCP 监听；
// Password 是传统 VNC 鉴权使用的可选 env:/file: 凭据引用。
type VNC struct {
	Enabled  bool   `toml:"enabled"`
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Password string `toml:"password,omitempty"`
	Timeout  string `toml:"timeout"`
}

func (v *VNC) normalize(base string) error {
	if v.Host == "" {
		v.Host = "127.0.0.1"
	}
	if net.ParseIP(v.Host) == nil && strings.ContainsAny(v.Host, ":/\\[] \t\r\n") {
		return fmt.Errorf("VNC host must be a hostname or IP without a port")
	}
	if v.Port == 0 {
		v.Port = 5900
	}
	if v.Port < 1 || v.Port > 65535 {
		return fmt.Errorf("VNC port must be 1..65535")
	}
	if v.Timeout == "" {
		v.Timeout = "15s"
	}
	d, err := time.ParseDuration(v.Timeout)
	if err != nil || d <= 0 || d > 2*time.Minute {
		return fmt.Errorf("VNC timeout must be positive and at most 120s")
	}
	if v.Password != "" && !strings.HasPrefix(v.Password, "env:") && !strings.HasPrefix(v.Password, "file:") {
		return fmt.Errorf("VNC password must use env:NAME or file:PATH")
	}
	v.Password = resolveFile(base, v.Password)
	return nil
}
