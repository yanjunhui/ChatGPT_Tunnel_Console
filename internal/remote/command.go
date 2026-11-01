// Created by Yanjunhui
package remote

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// CommandInput describes an independent shell invocation; nil timeout uses the
// profile deadline and an explicit zero disables that deadline.
// 中文：CommandInput 描述独立 shell 调用；空超时使用配置值，显式零值禁用命令期限。
type CommandInput struct {
	Profile        string `json:"profile,omitempty" jsonschema:"SSH profile; omit for default"`
	Command        string `json:"command" jsonschema:"Shell command or multiline script passed verbatim to the target shell"`
	Stdin          string `json:"stdin,omitempty" jsonschema:"Standard input; closed after this data is sent"`
	TTY            bool   `json:"tty,omitempty" jsonschema:"Allocate a terminal; stderr then usually merges into stdout"`
	TimeoutSeconds *int   `json:"timeout_seconds,omitempty" jsonschema:"Override command timeout; zero disables it"`
}

// CommandResult retains output even after execution fails. A nil ExitCode means
// no usable exit status was received; Error is distinct from stderr output.
// 中文：CommandResult 在执行失败后仍保留输出；空 ExitCode 表示没有可用退出码，
// Error 与远端 stderr 内容不同。
type CommandResult struct {
	Stdout     Output `json:"stdout"`
	Stderr     Output `json:"stderr"`
	ExitCode   *int   `json:"exit_code"`
	Error      string `json:"error,omitempty"`
	TimedOut   bool   `json:"timed_out"`
	Canceled   bool   `json:"canceled"`
	DurationMS int64  `json:"duration_ms"`
}

// Run executes in a fresh SSH connection and the profile workdir. Validation and
// connection failures return errors; execution failures populate CommandResult.
// Cancellation closes SSH but does not guarantee detached children have exited.
// 中文：Run 在独立 SSH 连接和配置目录中执行；参数或连接失败返回 error，
// 执行失败写入 CommandResult。取消会关闭 SSH，但不保证脱离会话的子进程退出。
func (m *Manager) Run(ctx context.Context, in CommandInput) (CommandResult, error) {
	_, p, err := m.config.Profile(in.Profile)
	if err != nil {
		return CommandResult{}, err
	}
	if strings.TrimSpace(in.Command) == "" {
		return CommandResult{}, fmt.Errorf("command must not be empty")
	}
	duration, _ := time.ParseDuration(p.CommandTimeout)
	if in.TimeoutSeconds != nil {
		if *in.TimeoutSeconds < 0 || int64(*in.TimeoutSeconds) > int64((1<<63-1)/time.Second) {
			return CommandResult{}, fmt.Errorf("invalid timeout_seconds")
		}
		duration = time.Duration(*in.TimeoutSeconds) * time.Second
	}
	if duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}
	client, err := dial(ctx, p)
	if err != nil {
		return CommandResult{}, err
	}
	defer client.Close()
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stop()
	session, err := client.NewSession()
	if err != nil {
		return CommandResult{}, fmt.Errorf("open SSH channel: %w", err)
	}
	defer session.Close()
	return execute(ctx, session, in, p.Workdir, m.config.OutputBytes)
}

func execute(ctx context.Context, session *ssh.Session, in CommandInput, dir string, limit int) (CommandResult, error) {
	out, stderr := NewBuffer(limit), NewBuffer(limit)
	session.Stdout, session.Stderr = out, stderr
	session.Stdin = strings.NewReader(in.Stdin)
	if in.TTY {
		if err := session.RequestPty("xterm-256color", 40, 120, ssh.TerminalModes{ssh.ECHO: 0}); err != nil {
			return CommandResult{}, fmt.Errorf("allocate PTY: %w", err)
		}
	}
	started := time.Now()
	err := session.Run(inDirectory(in.Command, dir))
	result := CommandResult{DurationMS: time.Since(started).Milliseconds()}
	result.Stdout, _ = out.Read(0)
	result.Stderr, _ = stderr.Read(0)
	result.ExitCode, result.Error = exitStatus(err)
	if ctx.Err() != nil {
		result.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
		result.Canceled = errors.Is(ctx.Err(), context.Canceled)
		result.Error = ctx.Err().Error() + "; SSH channel closed, detached remote children may still run"
		result.ExitCode = nil
	}
	return result, nil
}

func exitStatus(err error) (*int, string) {
	code := 0
	if err == nil {
		return &code, ""
	}
	var exit *ssh.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitStatus()
		return &code, err.Error()
	}
	return nil, err.Error()
}
