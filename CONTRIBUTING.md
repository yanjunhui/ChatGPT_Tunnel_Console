<!-- Created by Yanjunhui -->
# Contributing

English | [简体中文](CONTRIBUTING.zh-CN.md)

Use Go at the version specified in `go.mod`, Python 3 for documentation checks, and Docker Compose v2 for container acceptance. Development is supported on macOS/Linux; the runtime targets POSIX SSH hosts.

```sh
git clone https://github.com/yanjunhui/ChatGPT_Tunnel_SSH.git
cd ChatGPT_Tunnel_SSH
make check test vet build linux
```

`make` keeps Go/module/build/temporary files in `.cache/`; generated binaries are in `bin/`. Do not commit either directory. `make test` runs real local SSH/SFTP and PTY tests, including race detection. It does not contact a live tunnel. The two Docker tests are opt-in:

```sh
TUNNEL_SSH_MCP_IMAGE=go-tunnel-ssh-mcp:review \
TUNNEL_SSH_DEV_IMAGE=go-tunnel-ssh-dev:review \
make docker-build docker-test
```

Use separate image tags when a formal deployment exists. Tests create temporary containers and volumes, use stdio and synthetic credentials, and clean up their own resources. Never run `down -v` against a deployment with data you need.

Docker tests use private temporary workspaces. A short-lived root container assigns runtime UID/GID 1000 to those fixtures and restores the invoking host user's ownership before cleanup. This supports native Linux hosts whose user ID differs from the image; it never changes formal deployment paths.

Run `make vuln` for known reachable Go vulnerabilities and `make secrets` to scan Git history with redacted output. These tools are version-pinned in the Makefile and require network access for installation/database queries. They do not replace manual review. Run `make tidy` after dependency changes and include both module files. CI runs macOS/Linux race tests, builds, documentation checks, secret/dependency scans and Linux Docker integration; it never uses real tunnel credentials.

## Change process

1. Reproduce defects before editing and add a regression test for meaningful behavior changes.
2. Read the affected implementation and keep changes focused. Preserve existing MCP contracts unless a compatibility change is explicitly documented.
3. Use gofmt. Keep functions within 50 lines and files within 999 nonblank lines. New files start with `Created by Yanjunhui` in the appropriate comment syntax.
4. Keep English documentation primary with complete `.zh-CN.md` companions. Source comments put English before Chinese and explain contracts or design reasons. Update examples, defaults and error behavior in both languages.
5. Run the checks appropriate to the change and distinguish local tests from actual ChatGPT acceptance. Never claim a command succeeded merely because input was submitted.
6. Submit a focused PR describing the defect or need, resulting behavior, verification and compatibility impact. Use Conventional Commits with a Chinese description, for example `fix: 正确处理子命令帮助退出状态`.

Report ordinary defects through the issue form. For credential exposure or exploitable vulnerabilities, use the private channel in [SECURITY.md](SECURITY.md). Do not attach live configuration, private keys, account identifiers, tunnel IDs or private chat links. Use synthetic fixtures and preserve meaningful errors after redaction.

## Licensing

Contributions are accepted under the repository's [MIT license](LICENSE). Submit only code and documentation you have permission to contribute. Third-party components keep their own licenses; see [THIRD_PARTY_NOTICES.md](THIRD_PARTY_NOTICES.md).
