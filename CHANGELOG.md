<!-- Created by Yanjunhui -->
# Changelog

English | [简体中文](CHANGELOG.zh-CN.md)

## 0.1.2 — 2026-11-01

- Fix `--help` returning an error for run, serve, check, doctor and install-tunnel; add a regression test covering these commands and init.
- Raise the minimum Go toolchain and Docker build baseline to 1.26.10 after vulnerability review.
- Align the Go module path with `github.com/yanjunhui/ChatGPT_Tunnel_SSH`.
- Allow separate Compose image tags for acceptance builds and formal deployments.
- Create the project temporary directory before Docker builds, including in a fresh checkout.
- Prepare the public source release with MIT licensing, bilingual contribution/security/third-party documentation, CI, dependency updates, documentation checks and secret scanning. Redact private deployment identifiers from published records.

MCP tool contracts remain compatible with 0.1.1. This source release does not automatically upgrade an existing deployment; the recorded formal deployment remains 0.1.1.

## 0.1.1 — 2026-11-01

- Add `outputSchema` and typed `structuredContent` to inline SFTP downloads while retaining the existing text JSON response, including explicit empty-file values.
- Verify the formal container upgrade through actual tunnel, SSH and SFTP calls. See [verification](docs/verification.md).
- Complete English documentation and Simplified Chinese companions, with bilingual source comments.

## 0.1.0 — 2026-11-01

- Implement the independent Go MCP server with 13 SSH, PTY/session and SFTP tools.
- Integrate the official outbound OpenAI tunnel and provide native service installation and one-shot Docker initialization.
- Add local protocol tests and isolated Docker OpenSSH acceptance tests.
