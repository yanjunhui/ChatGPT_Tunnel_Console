// Created by Yanjunhui
package remote

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"golang.org/x/crypto/ssh"
)

// OpenInput selects an interactive PTY or a background command. Names are unique
// within a profile; zero terminal dimensions use 40 rows and 120 columns.
// 中文：OpenInput 选择交互 PTY 或后台命令；名称在配置内唯一，零尺寸使用 40 行、120 列。
type OpenInput struct {
	Profile string `json:"profile,omitempty"`
	Name    string `json:"name" jsonschema:"Unique name within the SSH profile"`
	Type    string `json:"type,omitempty" jsonschema:"interactive (default) or background"`
	Command string `json:"command,omitempty" jsonschema:"Required for background sessions"`
	Rows    int    `json:"rows,omitempty"`
	Cols    int    `json:"cols,omitempty"`
}

// SessionInfo is a snapshot; exited sessions remain listed until explicitly closed.
// 中文：SessionInfo 是状态快照；已退出会话仍保留在列表中，直到显式关闭。
type SessionInfo struct {
	Name     string    `json:"name"`
	Profile  string    `json:"profile"`
	Type     string    `json:"type"`
	Status   string    `json:"status"`
	Started  time.Time `json:"started"`
	ExitCode *int      `json:"exit_code"`
	Error    string    `json:"error,omitempty"`
}

// Session owns one SSH connection, serialized stdin and a shared output buffer.
// 中文：Session 独占 SSH 连接，串行写入 stdin，并合并缓存 stdout 与 stderr。
type Session struct {
	mu      sync.Mutex
	inputMu sync.Mutex
	info    SessionInfo
	client  *ssh.Client
	channel *ssh.Session
	stdin   io.WriteCloser
	output  *Buffer
	done    chan struct{}
	once    sync.Once
}

// Open reserves a name before connecting and retains the session after the request
// ends. Invalid input, duplicate names, shutdown or SSH startup failures return errors.
// 中文：Open 先预留名称再连接，成功会话在请求结束后继续保留；
// 参数无效、名称重复、服务关闭或 SSH 启动失败时返回错误。
func (m *Manager) Open(ctx context.Context, in OpenInput) (SessionInfo, error) {
	profile, p, err := m.config.Profile(in.Profile)
	if err != nil {
		return SessionInfo{}, err
	}
	if err = validateOpen(&in); err != nil {
		return SessionInfo{}, err
	}
	key := sessionKey{profile, in.Name}
	if err = m.reserve(key); err != nil {
		return SessionInfo{}, err
	}
	defer func() { m.mu.Lock(); delete(m.pending, key); m.mu.Unlock() }()
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(m.ctx, cancel)
	defer cancel()
	defer stop()
	s, err := startSession(ctx, p, in, profile, m.config.OutputBytes)
	if err != nil {
		return SessionInfo{}, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		s.disconnect()
		return SessionInfo{}, fmt.Errorf("server is shutting down")
	}
	m.sessions[key] = s
	return s.Info(), nil
}

func validateOpen(in *OpenInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("session name must not be empty")
	}
	if in.Type == "" {
		in.Type = "interactive"
	}
	if in.Type != "interactive" && in.Type != "background" {
		return fmt.Errorf("type must be interactive or background")
	}
	if in.Type == "background" && strings.TrimSpace(in.Command) == "" {
		return fmt.Errorf("background command is required")
	}
	if in.Type == "interactive" && in.Command != "" {
		return fmt.Errorf("interactive sessions accept input through write-session-input")
	}
	if in.Rows == 0 {
		in.Rows = 40
	}
	if in.Cols == 0 {
		in.Cols = 120
	}
	if in.Rows < 1 || in.Rows > 65535 || in.Cols < 1 || in.Cols > 65535 {
		return fmt.Errorf("terminal size must be 1..65535")
	}
	return nil
}

func (m *Manager) reserve(key sessionKey) error {
	// Reserve across the network handshake so concurrent opens cannot share a name.
	// 中文：网络握手期间也保留名称，避免并发创建同名会话。
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return fmt.Errorf("server is shutting down")
	}
	if m.sessions[key] != nil || m.pending[key] {
		return fmt.Errorf("session %q already exists", key.name)
	}
	m.pending[key] = true
	return nil
}

func startSession(ctx context.Context, p config.Profile, in OpenInput, profile string, limit int) (*Session, error) {
	client, err := dial(ctx, p)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			_ = client.Close()
		}
	}()
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stop()
	ch, err := client.NewSession()
	if err != nil {
		return nil, err
	}
	stdin, err := ch.StdinPipe()
	if err != nil {
		return nil, err
	}
	s := &Session{client: client, channel: ch, stdin: stdin, output: NewBuffer(limit), done: make(chan struct{})}
	s.info = SessionInfo{Name: in.Name, Profile: profile, Type: in.Type, Status: "running", Started: time.Now().UTC()}
	ch.Stdout, ch.Stderr = s.output, s.output
	command := in.Command
	if in.Type == "interactive" {
		if err = ch.RequestPty("xterm-256color", in.Rows, in.Cols, ssh.TerminalModes{}); err != nil {
			return nil, err
		}
		command = "exec \"${SHELL:-/bin/sh}\" -i"
	}
	if err = ch.Start(inDirectory(command, p.Workdir)); err != nil {
		return nil, err
	}
	if !stop() || ctx.Err() != nil {
		return nil, ctx.Err()
	}
	ok = true
	go s.wait()
	return s, nil
}

func (s *Session) wait() {
	err := s.channel.Wait()
	s.mu.Lock()
	s.info.ExitCode, s.info.Error = exitStatus(err)
	s.info.Status = "exited"
	s.mu.Unlock()
	s.disconnect()
	close(s.done)
}

func (s *Session) disconnect() { s.once.Do(func() { _ = s.client.Close() }) }

// Info returns a synchronized copy of the current session state.
// 中文：Info 在锁保护下复制当前会话状态。
func (s *Session) Info() SessionInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.info
}

// SessionOutput pairs session status with combined stdout/stderr, including PTY echo.
// 中文：SessionOutput 同时返回状态和合并输出，PTY 输出可能包含输入回显。
type SessionOutput struct {
	Session SessionInfo `json:"session"`
	Output  Output      `json:"output"`
}

// Read returns currently retained output without waiting for a command to finish.
// Unknown sessions and invalid absolute byte cursors return errors.
// 中文：Read 立即读取保留输出，不等待命令完成；会话不存在或字节游标无效时报错。
func (m *Manager) Read(profile, name string, cursor int64) (SessionOutput, error) {
	s, err := m.get(profile, name)
	if err != nil {
		return SessionOutput{}, err
	}
	out, err := s.output.Read(cursor)
	return SessionOutput{Session: s.Info(), Output: out}, err
}

// Write sends raw terminal bytes. It does not claim that a shell command finished.
// Set eof to close stdin after data; cancellation disconnects this session.
// 中文：Write 仅确认原始输入已写入，不表示命令完成；eof 在写入后关闭 stdin，
// 请求取消会断开该会话。
func (m *Manager) Write(ctx context.Context, profile, name, data string, eof bool) (SessionInfo, error) {
	s, err := m.get(profile, name)
	if err != nil {
		return SessionInfo{}, err
	}
	stop := context.AfterFunc(ctx, s.disconnect)
	defer stop()
	s.inputMu.Lock()
	defer s.inputMu.Unlock()
	if ctx.Err() != nil {
		return s.Info(), ctx.Err()
	}
	if s.Info().Status != "running" {
		return s.Info(), fmt.Errorf("session has exited")
	}
	if _, err = io.WriteString(s.stdin, data); err == nil && eof {
		err = s.stdin.Close()
	}
	return s.Info(), err
}

// CloseResult distinguishes closing SSH from observing a remote exit status.
// 中文：CloseResult 区分“已关闭 SSH 连接”和“已观察到远端退出码”。
type CloseResult struct {
	Session             SessionInfo `json:"session"`
	ProcessExitObserved bool        `json:"process_exit_observed"`
	Note                string      `json:"note"`
}

// CloseSession requests TERM, waits up to 500 ms, disconnects and removes the entry.
// Missing sessions return errors; detached children may survive a successful close.
// 中文：CloseSession 请求 TERM，最多等待 500 毫秒后断开并移除记录；
// 会话不存在时报错，成功关闭也不保证脱离会话的子进程退出。
func (m *Manager) CloseSession(ctx context.Context, profile, name string) (CloseResult, error) {
	s, err := m.get(profile, name)
	if err != nil {
		return CloseResult{}, err
	}
	stop := context.AfterFunc(ctx, s.disconnect)
	defer stop()
	go func() { _ = s.channel.Signal(ssh.SIGTERM) }()
	select {
	case <-s.done:
	case <-time.After(500 * time.Millisecond):
	case <-ctx.Done():
	}
	s.disconnect()
	info := s.Info()
	m.mu.Lock()
	key := sessionKey{info.Profile, name}
	if m.sessions[key] == s {
		delete(m.sessions, key)
	}
	m.mu.Unlock()
	return CloseResult{Session: info, ProcessExitObserved: info.ExitCode != nil, Note: "SSH connection closed. Detached children are not guaranteed to terminate; use signal-process for a known PID."}, nil
}
