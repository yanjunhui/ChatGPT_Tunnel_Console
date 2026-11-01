<!-- Created by Yanjunhui -->
# Console rename and VNC implementation plan

English | [简体中文](2026-11-01-console-vnc.zh-CN.md)

> Historical record: this document describes the implementation and checks for the version or phase named below. For current setup and usage, see the [documentation](../../README.md).

**Goal:** Rename the public project to ChatGPT_Tunnel_Console and let ChatGPT/Codex view and operate a VNC desktop through MCP screenshots and input tools.

**Architecture:** Keep the official OpenAI MCP tunnel. Each enabled SSH profile may describe a VNC target reached using SSH direct TCP forwarding. Bounded RFB clients make shared, short-lived connections; no public VNC listener or browser viewer is added. Existing SSH/SFTP interfaces remain compatible.

**Tech stack:** Go, the existing MCP SDK and SSH library, RFB 3.3/3.7/3.8 Raw encoding, PNG MCP images, optional TigerVNC/Openbox Docker environment.

## Tasks

1. Rename the Go module, CLI source package, displayed project name, repository links and GitHub repository. Keep existing installation paths, Compose project/volumes and environment variable names as documented deployment compatibility identifiers. Do not restart the existing deployment.
2. Add `internal/config/vnc.go`, optional profile VNC fields and tests for defaults, invalid ports/timeouts and credential references. Extend `internal/setup/configuration.go` to persist VNC secrets even for the built-in development profile.
3. Add `internal/vnc` with bounded RFB handshakes, None/VNC authentication, Raw framebuffer decoding, resizing and input events. Test protocol versions, challenge responses, malformed/truncated packets, dimensions, authentication failures and cancellation through closed transports.
4. Add `internal/remote/desktop.go` for SSH forwarding and bounded, serialized operations. Add `internal/server/desktop.go` with `vnc-screenshot`, `vnc-pointer`, `vnc-key` and `vnc-type`. Validate all input before connecting; release pressed buttons/keys within the request. Return native MCP image content and explicit input-delivery results.
5. Add an optional desktop Docker target and `compose.desktop.yaml`. Bind VNC to target loopback and reach it only through SSH. Extend isolated Docker tests with actual screenshot decoding, mouse movement, keyboard input and no published ports.
6. Update both READMEs, CLI/config/tool references, security notes, third-party notices, changelog and acceptance records. Run formatting, race tests, vet, cross-builds, vulnerability/secret scans and real Docker tests; inspect a captured desktop. Rename/push the GitHub repository and verify CI for the published commit.

## Acceptance and limits

- Old configurations keep working with VNC disabled. Desktop enablement requires explicit configuration and an available VNC server.
- Support standard RFB Raw updates and classic VNC password authentication over SSH; reject unsupported security/encoding types. Do not silently downgrade configured password authentication.
- Screenshots retain their original pixel dimensions, with bounded memory/output and request deadlines. Input results confirm delivery, not application-level completion.
- No persistent key holds, clipboard synchronization, video streaming, browser viewer or OS desktop installation on an external host is included.
- Retain the existing formal 0.1.1 container; separate local/CI acceptance from a future live ChatGPT deployment.

## Progress

- [x] Confirm MCP screenshot and keyboard/mouse interaction scope.
- [x] Implement rename, protocol, configuration and MCP tools.
- [x] Pass local and real Docker desktop verification.
- [x] Synchronize bilingual documentation and publish verified source.
