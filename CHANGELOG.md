<!-- Created by Yanjunhui -->
# Changelog

English | [简体中文](CHANGELOG.zh-CN.md)

## 0.3.1 — 2026-11-01

- Fix real ChatGPT screenshot visibility by omitting metadata-only structured output from both screenshot tools. Preserve PNG image content and text JSON metadata; clients using structured screenshot dimensions must read `content[1].text`.
- Retain typed SFTP download results and all independent native/VNC behavior.

## 0.3.0 — 2026-11-01

- Add four native `desktop-*` MCP tools over SSH/X11, independent of VNC; discovery now exposes 21 tools.
- Split the bundled Xvfb desktop from the optional x11vnc service. Either tool backend can be enabled alone; stopping VNC preserves native screenshot/input access.
- Add independent desktop configuration, an optional VNC initializer question, bounded XWD decoding and UTF-8 input handling for non-login SSH environments.
- Add real-container independence, Unicode input and authentication checks, plus bilingual operation and upgrade documentation.
- Clarify command metadata without changing target permissions; client-side approval rules remain in effect.

## 0.2.0 — 2026-11-01

- Rename the project/repository to ChatGPT_Tunnel_Console, the Go module to `github.com/yanjunhui/ChatGPT_Tunnel_Console`, and the primary CLI to `tunnel-console-mcp`.
- Add opt-in VNC screenshots, pointer gestures, keyboard chords and Unicode typing as four MCP tools over SSH forwarding; existing SSH/SFTP contracts remain available.
- Implement bounded RFB 3.3/3.7/3.8 Raw screenshots, desktop resize, None/VncAuth authentication and cancellation cleanup without adding a Go dependency.
- Add an optional TigerVNC/Openbox Docker desktop, an initializer desktop choice, real desktop acceptance and complete bilingual configuration/tool documentation.
- Retain existing deployment paths, Compose/volume identities, environment settings and the old CLI alias. Existing configurations keep VNC disabled. Running 0.1.1 services are not upgraded by this source release.

## 0.1.2 — 2026-11-01

- Fix `--help` returning an error for run, serve, check, doctor and install-tunnel; add a regression test covering these commands and init.
- Raise the minimum Go toolchain and Docker build baseline to 1.26.10 after vulnerability review.
- Align the Go module path with `github.com/yanjunhui/ChatGPT_Tunnel_SSH`.
- Allow separate Compose image tags for acceptance builds and formal deployments.
- Create the project temporary directory before Docker builds, including in a fresh checkout.
- Make Docker test fixtures portable across Linux host user IDs; document bind-mount ownership and preserve private configuration permissions.
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
