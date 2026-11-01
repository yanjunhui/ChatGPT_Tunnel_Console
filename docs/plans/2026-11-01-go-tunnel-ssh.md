<!-- Created by Yanjunhui -->
# Go OpenAI Tunnel SSH MCP implementation plan

English | [简体中文](2026-11-01-go-tunnel-ssh.zh-CN.md)

Historical plan. Acceptance statements describe that stage; see the [deployment record](../live-deployment.md) for subsequent live status.

**Goal:** independently implement a Go SSH MCP service exposed only through OpenAI Tunnel, with a Docker Linux development environment.

**Architecture:** official tunnel-client supervises a Go stdio MCP worker, which operates targets through SSH/SFTP. There is no MCP HTTP port, separate Bearer/OAuth layer, command policy, approval system, roles, path allow/deny lists or daily quota.

**Stack:** Go, official MCP Go SDK, x/crypto/ssh, pkg/sftp, TOML, official tunnel-client and Docker Compose.

## Checklist

- [x] Configuration and startup: `internal/config`, `internal/tunnel`, `cmd/tunnel-ssh-mcp`; validate invalid configuration, escaping and avoidance of credential contents in command lines.
- [x] SSH/SFTP: `internal/remote`; execution, timeout, cancellation, PTY, background tasks, input/output and transfers without retrying possibly executed commands.
- [x] MCP: `internal/server`; tool registration and SDK-client verification of discovery, parameters and errors.
- [x] Deployment: `Dockerfile`, `compose.yaml`, `deploy`; local development environment, external-host configuration and official tunnel installation.
- [x] Verification at that stage: 27 tests passed, including race checks and real Docker OpenSSH acceptance; static checks and native/Linux dual-architecture builds passed. Live acceptance then awaited credentials; see [verification](../verification.md).

## Design decisions

- Do not copy or wrap the two reference implementations or link npm ssh-mcp.
- OpenAI Tunnel authenticates MCP callers; SSH keys/passwords authenticate only to target hosts.
- Persistent sessions use native SSH PTY and explicit input/output tools, not shell sentinels that pretend a command finished.
- Each independent command/transfer opens its own connection; each persistent session owns one, so canceling one operation does not interrupt others.
- The target shell and OS handle multiline commands, pipes, redirection and sudo directly.
- Bounded output and timeouts serve runtime stability; report discarded bytes and timeout state explicitly.
- At this implementation stage, do not deploy to existing live tunnels or reuse other projects' credentials. End-to-end acceptance needs a separately configured tunnel.
