// Created by Yanjunhui
// Package remote implements SSH commands, retained sessions and SFTP transfers.
// 中文：remote 包实现 SSH 命令、持久会话和 SFTP 文件传输。
package remote

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
)

// Manager owns named sessions shared by all callers of this MCP process.
// 中文：Manager 管理当前 MCP 进程中所有调用方共享的命名会话。
type Manager struct {
	config   *config.Config
	mu       sync.Mutex
	sessions map[sessionKey]*Session
	closed   bool
	pending  map[sessionKey]bool
	ctx      context.Context
	cancel   context.CancelFunc
	desktop  chan struct{}
}

type sessionKey struct{ profile, name string }

// New uses a validated configuration without connecting; callers must call Close.
// The configuration must not be mutated while the manager is in use.
// 中文：New 接收已校验配置，不立即连接；使用期间不要修改配置，结束时调用 Close。
func New(c *config.Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	return &Manager{config: c, sessions: make(map[sessionKey]*Session), pending: make(map[sessionKey]bool), ctx: ctx, cancel: cancel, desktop: make(chan struct{}, 1)}
}

// ConnectionInfo exposes non-secret settings and the number of retained sessions.
// 中文：ConnectionInfo 返回不含凭据的配置及保留会话数，其中可包含已退出会话。
type ConnectionInfo struct {
	Name       string `json:"name"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	User       string `json:"user"`
	Workdir    string `json:"workdir"`
	Default    bool   `json:"default"`
	Sessions   int    `json:"sessions"`
	Status     string `json:"status"`
	VNCEnabled bool   `json:"vnc_enabled"`
}

// Connections reports configuration, not an unperformed connectivity check.
// Results are sorted by profile name; no credentials are returned.
// 中文：Connections 按名称返回配置，不探测连通性，也不返回凭据。
func (m *Manager) Connections() []ConnectionInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]ConnectionInfo, 0, len(m.config.Profiles))
	for name, p := range m.config.Profiles {
		info := ConnectionInfo{Name: name, Host: p.Host, Port: p.Port, User: p.User, Workdir: p.Workdir, Default: name == m.config.DefaultProfile, Status: "configured; connect on use"}
		info.VNCEnabled = p.VNC != nil && p.VNC.Enabled
		for key := range m.sessions {
			if key.profile == name {
				info.Sessions++
			}
		}
		result = append(result, info)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func (m *Manager) get(profile, name string) (*Session, error) {
	profile, _, err := m.config.Profile(profile)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	s := m.sessions[sessionKey{profile, name}]
	if s == nil {
		return nil, fmt.Errorf("session %q does not exist on %q", name, profile)
	}
	return s, nil
}

// List returns retained sessions, sorted by name, for one resolved profile.
// Empty profile selects the default; an unknown profile returns an error.
// 中文：List 返回指定配置的保留会话并按名称排序；空配置名使用默认项，未知配置报错。
func (m *Manager) List(profile string) ([]SessionInfo, error) {
	profile, _, err := m.config.Profile(profile)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	result := []SessionInfo{}
	for key, s := range m.sessions {
		if key.profile == profile {
			result = append(result, s.Info())
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

// Close disconnects retained sessions and cancels pending opens. It is idempotent.
// Independent commands and transfers are owned by their request contexts.
// 中文：Close 关闭保留会话并取消创建中的会话，可重复调用；
// 独立命令和文件传输由各自请求的上下文管理。
func (m *Manager) Close() {
	m.mu.Lock()
	m.closed = true
	m.cancel()
	sessions := m.sessions
	m.sessions = make(map[sessionKey]*Session)
	m.mu.Unlock()
	for _, s := range sessions {
		s.disconnect()
	}
}
