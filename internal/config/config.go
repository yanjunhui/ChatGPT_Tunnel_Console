// Created by Yanjunhui
// Package config loads SSH profiles and resolves credential references.
// 中文：config 包加载 SSH 配置，并解析凭据引用。
package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config contains connection settings only; there is no authorization policy.
// 中文：Config 仅包含连接设置，不定义调用方授权策略。
type Config struct {
	DefaultProfile string             `toml:"default_profile"`
	OutputBytes    int                `toml:"output_bytes"`
	Tunnel         Tunnel             `toml:"tunnel"`
	Profiles       map[string]Profile `toml:"profiles"`
}

// Tunnel configures the official supervisor; APIKey is an env: or file: reference.
// 中文：Tunnel 配置官方隧道进程；APIKey 使用 env: 或 file: 引用。
type Tunnel struct {
	Binary     string `toml:"binary"`
	ID         string `toml:"id"`
	APIKey     string `toml:"api_key"`
	HealthAddr string `toml:"health_addr"`
}

// Profile describes one SSH target. Password and Passphrase contain references,
// while PrivateKey names a file and AgentSocket names a socket or env: variable.
// 中文：Profile 描述一个 SSH 目标。密码和口令使用引用，私钥使用文件路径，
// agent 使用 socket 路径或 env: 环境变量。
type Profile struct {
	Host           string `toml:"host"`
	Port           int    `toml:"port"`
	User           string `toml:"user"`
	PrivateKey     string `toml:"private_key"`
	Passphrase     string `toml:"passphrase"`
	Password       string `toml:"password"`
	AgentSocket    string `toml:"agent_socket"`
	Workdir        string `toml:"workdir"`
	ConnectTimeout string `toml:"connect_timeout"`
	CommandTimeout string `toml:"command_timeout"`
}

// Load reads a TOML file. Relative file paths resolve against that file.
// It rejects unknown fields and invalid settings without contacting the target.
// 中文：Load 读取 TOML，以配置文件目录解析相对路径；拒绝未知字段及无效配置，
// 不连接远端，也不验证凭据是否可用于登录。
func Load(path string) (*Config, error) {
	var c Config
	meta, err := toml.DecodeFile(path, &c)
	if err != nil {
		return nil, fmt.Errorf("read configuration: %w", err)
	}
	if keys := meta.Undecoded(); len(keys) > 0 {
		return nil, fmt.Errorf("unknown configuration field: %s", keys[0])
	}
	base, err := filepath.Abs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}
	if err = c.normalize(base); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) normalize(base string) error {
	if len(c.Profiles) == 0 {
		return fmt.Errorf("at least one SSH profile is required")
	}
	if c.DefaultProfile == "" && len(c.Profiles) == 1 {
		for name := range c.Profiles {
			c.DefaultProfile = name
		}
	}
	if _, ok := c.Profiles[c.DefaultProfile]; !ok {
		return fmt.Errorf("default_profile must name an existing profile")
	}
	if c.OutputBytes == 0 {
		c.OutputBytes = 1024 * 1024
	}
	if c.OutputBytes < 0 || c.OutputBytes > 64*1024*1024 {
		return fmt.Errorf("output_bytes must be between 1 and 67108864 (0 uses default)")
	}
	for name, p := range c.Profiles {
		if err := p.normalize(base); err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		c.Profiles[name] = p
	}
	if c.Tunnel.Binary == "" {
		c.Tunnel.Binary = "tunnel-client"
	}
	if strings.ContainsAny(c.Tunnel.Binary, `/\`) {
		c.Tunnel.Binary = absolute(base, c.Tunnel.Binary)
	}
	c.Tunnel.APIKey = resolveFile(base, c.Tunnel.APIKey)
	if c.Tunnel.HealthAddr == "" {
		c.Tunnel.HealthAddr = "127.0.0.1:0"
	}
	return nil
}

// Validate applies defaults and validates configuration, resolving paths from base.
// It mutates the receiver and returns the first configuration error.
// 中文：Validate 原地填充默认值并以 base 解析路径，返回首个配置错误。
func (c *Config) Validate(base string) error { return c.normalize(base) }

func (p *Profile) normalize(base string) error {
	if strings.TrimSpace(p.Host) == "" || strings.TrimSpace(p.User) == "" {
		return fmt.Errorf("host and user are required")
	}
	if p.Port == 0 {
		p.Port = 22
	}
	if p.Port < 1 || p.Port > 65535 {
		return fmt.Errorf("port must be 1..65535")
	}
	if p.ConnectTimeout == "" {
		p.ConnectTimeout = "15s"
	}
	if p.CommandTimeout == "" {
		p.CommandTimeout = "60s"
	}
	for _, v := range []string{p.ConnectTimeout, p.CommandTimeout} {
		d, err := time.ParseDuration(v)
		if err != nil || d < 0 {
			return fmt.Errorf("invalid timeout %q", v)
		}
	}
	if p.PrivateKey != "" {
		p.PrivateKey = absolute(base, p.PrivateKey)
	}
	if p.AgentSocket != "" && !strings.HasPrefix(p.AgentSocket, "env:") {
		p.AgentSocket = absolute(base, p.AgentSocket)
	}
	p.Password = resolveFile(base, p.Password)
	p.Passphrase = resolveFile(base, p.Passphrase)
	if p.PrivateKey == "" && p.Password == "" && p.AgentSocket == "" {
		return fmt.Errorf("private_key, password, or agent_socket is required")
	}
	return nil
}

func absolute(base, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(base, path)
}

func resolveFile(base, value string) string {
	if strings.HasPrefix(value, "file:") {
		return "file:" + absolute(base, strings.TrimPrefix(value, "file:"))
	}
	return value
}

// Secret resolves env:NAME or file:PATH without placing secret contents in errors.
// Files may end with a newline; other whitespace is preserved.
// An empty reference returns an empty value; an empty referenced source is an error.
// 中文：Secret 解析环境变量或文件引用，错误不包含密钥内容；仅移除文件末尾换行。
// 空引用返回空值，但引用的凭据源为空或缺失时返回错误。
func Secret(ref string) (string, error) {
	if ref == "" {
		return "", nil
	}
	var value string
	switch {
	case strings.HasPrefix(ref, "env:"):
		value = os.Getenv(strings.TrimPrefix(ref, "env:"))
	case strings.HasPrefix(ref, "file:"):
		data, err := os.ReadFile(strings.TrimPrefix(ref, "file:"))
		if err != nil {
			return "", fmt.Errorf("cannot read credential file")
		}
		value = strings.TrimRight(string(data), "\r\n")
	default:
		return "", fmt.Errorf("credentials must use env:NAME or file:PATH")
	}
	if value == "" {
		return "", fmt.Errorf("credential source is empty or missing")
	}
	return value, nil
}

// Profile returns the named profile, using DefaultProfile when name is empty.
// An unknown name returns an error; the returned name is the resolved profile key.
// 中文：Profile 在 name 为空时使用默认项，未知名称返回错误；返回解析后的名称和配置。
func (c *Config) Profile(name string) (string, Profile, error) {
	if name == "" {
		name = c.DefaultProfile
	}
	p, ok := c.Profiles[name]
	if !ok {
		return "", Profile{}, fmt.Errorf("unknown SSH profile %q", name)
	}
	return name, p, nil
}

// Address formats the normalized host and port, including IPv6 brackets.
// 中文：Address 格式化已校验的主机和端口，正确处理 IPv6 方括号。
func (p Profile) Address() string { return net.JoinHostPort(p.Host, fmt.Sprint(p.Port)) }
