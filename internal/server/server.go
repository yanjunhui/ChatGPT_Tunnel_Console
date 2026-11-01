// Created by Yanjunhui
// Package server maps SSH operations to MCP tools on a local stdio transport.
// 中文：server 包把 SSH 操作注册为通过本地 stdio 传输的 MCP 工具。
package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/remote"
)

// Version identifies both the CLI release and the MCP server implementation.
// 中文：Version 同时标识 CLI 版本和 MCP 服务实现版本。
const Version = "0.1.2"

type profileInput struct {
	Profile string `json:"profile,omitempty"`
}
type sessionInput struct {
	Profile string `json:"profile,omitempty"`
	Name    string `json:"name"`
}
type readInput struct {
	sessionInput
	Cursor int64 `json:"cursor,omitempty" jsonschema:"Absolute byte cursor from the previous next_cursor; zero reads retained history"`
}
type writeInput struct {
	sessionInput
	Data string `json:"data" jsonschema:"Raw terminal input; append a newline to execute a command, or send control characters"`
	EOF  bool   `json:"eof,omitempty" jsonschema:"Close stdin after sending data"`
}
type signalInput struct {
	Profile string `json:"profile,omitempty"`
	PID     int    `json:"pid" jsonschema:"Positive remote process ID"`
	Signal  string `json:"signal,omitempty" jsonschema:"INT, TERM (default), or KILL"`
}

// New creates a stdio MCP service. No network listener or auth middleware exists.
// The caller owns manager shutdown; OpenAI Tunnel authenticates external callers.
// 中文：New 创建无网络监听和额外鉴权中间件的 MCP 服务；调用者负责关闭 manager，
// 外部调用方身份由 OpenAI 隧道验证。
func New(manager *remote.Manager) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "openai-tunnel-ssh", Version: Version}, &mcp.ServerOptions{
		Instructions: "SSH access through OpenAI Tunnel. Discover profiles first. Commands and paths have no application policy restrictions. Use run-command for independent commands; use open-session and write-session-input/read-session-output for persistent terminals. Tool filesystem paths refer to the MCP server or SSH host, not the ChatGPT client. All callers of this tunnel share named sessions.",
	})
	bind(s, "list-connections", "List configured SSH profiles; does not probe connectivity or expose credentials.", true,
		func(_ context.Context, _ struct{}) ([]remote.ConnectionInfo, error) {
			return manager.Connections(), nil
		})
	bind(s, "run-command", "Execute any shell command or multiline script. Nonzero exits, timeouts and transport errors are reported. No command classification or approvals. Use sudo directly if the target account permits it.", false, manager.Run)
	registerSessions(s, manager)
	registerFiles(s, manager)
	bind(s, "signal-process", "Send INT, TERM or KILL to a positive process ID on a POSIX SSH host.", false,
		func(ctx context.Context, in signalInput) (remote.CommandResult, error) {
			return signal(ctx, manager, in)
		})
	return s
}

func registerSessions(s *mcp.Server, manager *remote.Manager) {
	bind(s, "open-session", "Open an interactive PTY with persistent shell state, or a background command. Sessions survive tool requests and remain until closed or the service restarts.", false, manager.Open)
	bind(s, "list-sessions", "List retained sessions and their running/exited status.", true,
		func(_ context.Context, in profileInput) ([]remote.SessionInfo, error) {
			return manager.List(in.Profile)
		})
	bind(s, "write-session-input", "Send raw input to a session. This acknowledges input delivery, not command completion. Include newline to submit shell commands; read-session-output retrieves responses.", false,
		func(ctx context.Context, in writeInput) (remote.SessionInfo, error) {
			return manager.Write(ctx, in.Profile, in.Name, in.Data, in.EOF)
		})
	bind(s, "read-session-output", "Read combined terminal output using a byte cursor. Reports missing bytes if the rolling output buffer has wrapped.", true,
		func(_ context.Context, in readInput) (remote.SessionOutput, error) {
			return manager.Read(in.Profile, in.Name, in.Cursor)
		})
	bind(s, "close-session", "Request TERM and close the SSH connection. Reports whether process exit was observed; detached children may survive.", false,
		func(ctx context.Context, in sessionInput) (remote.CloseResult, error) {
			return manager.CloseSession(ctx, in.Profile, in.Name)
		})
}

func registerFiles(s *mcp.Server, manager *remote.Manager) {
	bind(s, "sftp-list", "List a remote directory via SFTP, including file sizes, permissions and modification times.", true, manager.ListFiles)
	// Typed output exposes structured file data while the SDK retains JSON text for older clients.
	// 中文：类型化输出提供结构化文件字段，同时由 SDK 保留供旧客户端读取的 JSON 文本。
	mcp.AddTool(s, &mcp.Tool{Name: "sftp-download", Description: "Read a remote file into UTF-8 or base64 content. Returns content, encoding and bytes as structured data and JSON text. For large files use sftp-download-file.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(ctx context.Context, _ *mcp.CallToolRequest, in remote.DownloadInput) (*mcp.CallToolResult, remote.FileContent, error) {
			out, err := manager.Download(ctx, in)
			return nil, out, err
		})
	bind(s, "sftp-upload", "Write UTF-8 or base64 content to a remote file via SFTP. Existing files are overwritten. Parent directory must exist.", false, manager.Upload)
	bind(s, "sftp-download-file", "Stream a remote file to local_path on the MCP server filesystem. Publishes on success and overwrites an existing local file. No transfer-root restrictions.", false, manager.DownloadFile)
	bind(s, "sftp-upload-file", "Stream local_path on the MCP server filesystem to the SSH host. Overwrites an existing remote file. No transfer-root restrictions.", false, manager.UploadFile)
}

func signal(ctx context.Context, manager *remote.Manager, in signalInput) (remote.CommandResult, error) {
	if in.PID < 1 {
		return remote.CommandResult{}, fmt.Errorf("pid must be positive")
	}
	if in.Signal == "" {
		in.Signal = "TERM"
	}
	if in.Signal != "INT" && in.Signal != "TERM" && in.Signal != "KILL" {
		return remote.CommandResult{}, fmt.Errorf("signal must be INT, TERM or KILL")
	}
	return manager.Run(ctx, remote.CommandInput{Profile: in.Profile, Command: fmt.Sprintf("kill -%s %d", in.Signal, in.PID)})
}

// bind preserves the JSON-text contract and marks execution errors without losing
// captured stdout/stderr. ReadOnlyHint is metadata, not an access-control rule.
// 中文：bind 保留 JSON 文本契约，标记执行错误时仍返回已有输出；
// ReadOnlyHint 仅是工具元数据，不是访问控制规则。
func bind[I, O any](s *mcp.Server, name, description string, readOnly bool, handler func(context.Context, I) (O, error)) {
	mcp.AddTool(s, &mcp.Tool{Name: name, Description: description, Annotations: &mcp.ToolAnnotations{ReadOnlyHint: readOnly}},
		func(ctx context.Context, _ *mcp.CallToolRequest, in I) (*mcp.CallToolResult, any, error) {
			out, err := handler(ctx, in)
			if err != nil {
				return nil, nil, err
			}
			data, err := json.Marshal(out)
			if err != nil {
				return nil, nil, err
			}
			result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}
			if command, ok := any(out).(remote.CommandResult); ok {
				result.IsError = command.Error != ""
			}
			return result, nil, nil
		})
}
