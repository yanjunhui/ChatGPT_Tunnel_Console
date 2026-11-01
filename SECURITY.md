<!-- Created by Yanjunhui -->
# Security policy

English | [简体中文](SECURITY.zh-CN.md)

## Reporting

Report vulnerabilities privately through [GitHub private vulnerability reporting](https://github.com/yanjunhui/ChatGPT_Tunnel_SSH/security/advisories/new). Include the affected version, a minimal synthetic reproduction, expected/actual behavior and impact. Do not post credentials, private configuration or exploitable details in public issues. Response times are best effort; this project has no contracted security SLA.

The latest code on `main` is maintained. Older releases do not have a separate backport commitment. A successful scan is evidence about the scanned revision and current advisory database, not a guarantee that the software has no vulnerabilities.

## Trust model and intended capabilities

This is a remote-administration tool for a trusted operator, not a multi-tenant restricted shell:

- OpenAI Tunnel authenticates access to the MCP service. The stdio worker has no independent authentication layer and must not be exposed through another public transport.
- All authorized callers share profiles, filesystem access and named sessions. The application deliberately has no roles, command approvals, allow/deny lists, path sandbox or daily quota. Tool annotations are metadata, not permission enforcement.
- Commands run with the SSH account's OS permissions. The bundled development account has passwordless sudo. Mounting a host directory into `/workspace` grants access to those host files.
- SSH target host keys are currently **not verified**. Tunnel authentication does not protect against impersonation of an external SSH target. Use trusted, controlled target networks; do not claim verified target identity.
- Credentials are referenced through environment variables or private files. Initialization persists them as files with mode 0600. These files are not encrypted by the application; protect the host, backups and Docker volumes.
- Cancellation closes SSH; it does not guarantee detached processes stop. Explicitly inspect and terminate known child processes where required.

These documented capabilities are intentional. Unexpected credential disclosure, incorrect isolation between deployments, unintended file corruption and behavior outside the stated contracts are reportable defects. Do not weaken host protections or broaden a deployment's access to reproduce an issue.

## Operational precautions

Use one tunnel instance per tunnel ID and runtime credentials scoped to that tunnel. Keep health listeners on loopback and do not publish SSH/MCP container ports. Preserve configuration and volume backups before changes. Use dedicated working directories and separate accounts/tunnels for different trust domains.

Keep Go and dependencies patched; run `make vuln` and review dependency-update PRs. Never commit `data/`, `init-input/`, `secrets/`, `.env*` files other than the placeholder example, private keys, workspace files or raw acceptance evidence. Repository scans must use redacted output. Rotate exposed credentials through the issuing service; deleting a committed secret does not remove it from history.
