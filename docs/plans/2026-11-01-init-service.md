<!-- Created by Yanjunhui -->
# Initialization and native boot services

English | [简体中文](2026-11-01-init-service.zh-CN.md)

Historical plan. Acceptance statements describe that stage; see the [deployment record](../live-deployment.md) for subsequent live status.

Goal: add `init` to the native Go CLI, prompt for tunnel/SSH settings on first use, and install/start a macOS launchd or Linux systemd service. The native Go service approach was approved by the user.

- [x] Go initialization wizard with hidden secret input and TOML import.
- [x] Fixed installation locations for executables, file credentials and configuration; repeated runs reuse settings by default.
- [x] Register a system service, start immediately and enable boot startup; default runtime account is the sudo caller.
- [x] Stage the new installation first, stop the old instance before switching, and restore the previous installation if activation fails.
- [x] Test arguments, escaping, credential persistence, service command order and rollback; run static checks and cross-platform builds.
- [x] Update CLI help, README and verification boundaries.

Layout: `/usr/local/lib/tunnel-ssh-mcp/releases/` holds complete generations, selected by `current`; `/var/lib/tunnel-ssh-mcp` is the runtime directory and `/var/log/tunnel-ssh-mcp` stores macOS logs. Service definitions live under `/Library/LaunchDaemons` on macOS and `/etc/systemd/system` on Linux. Services do not require an interactive terminal or login session.

This stage implemented initialization and isolated verification only. It did not register a real tunnel with invented credentials or claim live startup/reboot acceptance without real credentials.
