<!-- Created by Yanjunhui -->
# Docker initialization implementation plan

English | [简体中文](2026-11-01-docker-init.zh-CN.md)

> Historical record: this document describes the implementation and checks for the version or phase named below. For current setup and usage, see the [documentation](../../README.md).

Historical plan. Acceptance statements describe that stage; see the [deployment record](../live-deployment.md) for subsequent live status.

Goal: implement independent container initialization in Go while retaining native service installation. The container exposes MCP only through OpenAI Tunnel.

- [x] Add `init --container`, import/reuse/reconfigure flows and atomic publication of persisted configuration.
- [x] Provide separate `compose.init.yaml`; runtime Compose mounts configuration read-only and manages restart policy.
- [x] Cover prompts, credential persistence, permissions, failure preservation, CLI flag separation and native regressions.
- [x] Build images and verify OpenSSH, PTY and SFTP using the real initializer and runtime Compose definitions.
- [x] Update README and verification records.

The default host configuration directory is `data/`, mounted at `/etc/tunnel-ssh` in both initialization and runtime containers. Each run creates `generations/init-*`, validates it and atomically switches the relative `current` symlink. Credential references point to specific generations so running processes cannot mix old and new credentials. Runtime UID/GID is 1000; configuration and credential files use mode 0600.

The initial wizard defaults to bundled `dev`; its SSH key is generated when the development container first starts. External SSH accepts passwords or private keys imported through `/input`. Initialization neither connects to OpenAI, installs a system service nor starts formal containers. Real tunnel integration requires separate acceptance using the user's runtime credentials.
