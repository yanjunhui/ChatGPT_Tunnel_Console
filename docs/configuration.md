<!-- Created by Yanjunhui -->
# Configuration and CLI reference

English | [简体中文](configuration.zh-CN.md) | [README](../README.md)

This reference describes the repository's 0.2.0 implementation. See `internal/config`, `internal/setup`, `internal/tunnel` and `cmd/tunnel-console-mcp` for the corresponding code. Examples use placeholders, never real credentials.

## TOML configuration

`check`, `serve`, `run` and `doctor` accept `--config PATH` (default `config.toml`). Unknown TOML fields are rejected. Relative credential/key/binary paths resolve against the configuration file's directory, not the invoking shell's directory. A bare binary name is looked up through PATH. TOML paths do not expand `~`; the interactive private-key wizard separately expands a leading `~/` for the selected account.

| Root field | Default / requirement | Meaning |
|---|---|---|
| `default_profile` | Automatically selects the sole profile; otherwise required | Must name an existing profile |
| `output_bytes` | 1048576; `0` selects that default | Each output buffer's retained bytes and the raw-byte limit for inline downloads; explicit range 1–67108864 |
| `[profiles.NAME]` | At least one | Named SSH targets; names are also used to scope sessions |
| `[tunnel]` | Needed for `run`, `doctor` and initialization | Official supervisor settings; local `serve` does not use a tunnel |

| Tunnel field | Default | Meaning |
|---|---|---|
| `binary` | `tunnel-client` | PATH lookup or a file path; Docker init selects `/usr/local/bin/tunnel-client` |
| `id` | `CONTROL_PLANE_TUNNEL_ID` when omitted | Required when launching or initializing a tunnel |
| `api_key` | `env:CONTROL_PLANE_API_KEY` when omitted | Credential reference; key contents are not passed on the command line |
| `health_addr` | `127.0.0.1:0` | Loopback IP and port for official diagnostics; `0` requests an available port; Docker import template uses 8080 |

| Profile field | Default / requirement | Meaning |
|---|---|---|
| `host`, `user` | Nonblank | SSH target host and account |
| `port` | 22; `0` selects 22 | Valid explicit range 1–65535 |
| `private_key` | Optional path | SSH private key, copied during initialization |
| `passphrase` | Optional credential reference | Passphrase for an encrypted key |
| `password` | Optional credential reference | SSH password |
| `agent_socket` | Optional path or `env:NAME` | Supported for manual operation; native/container initialization rejects agent-dependent profiles |
| `workdir` | Empty | Working directory for commands/sessions and base for remote relative file paths; empty leaves target defaults in effect |
| `connect_timeout` | `15s` | Nonnegative Go duration; `0s` removes the profile connection deadline, while request cancellation still applies |
| `command_timeout` | `60s` | Nonnegative Go duration for independent commands; `0s` disables that deadline; persistent sessions do not use it |

At least one of `private_key`, `password` or `agent_socket` must be configured. Multiple authentication methods may be supplied. The implementation does not check target host keys. Tunnel caller authentication does not establish the identity of an external SSH host.

### Optional VNC configuration

`[profiles.NAME.vnc]` enables desktop access only for that SSH target. Omitting it keeps VNC disabled and existing configurations valid.

| VNC field | Default / requirement | Meaning |
|---|---|---|
| `enabled` | false | Explicitly allow desktop tools |
| `host` | `127.0.0.1` | Hostname or IP reached from the SSH target, without a port |
| `port` | 5900; `0` selects 5900 | Range 1–65535; bundled desktop uses 5901 |
| `password` | Empty | Empty selects None; otherwise only `env:NAME` or `file:PATH` |
| `timeout` | `15s` | Positive and at most `120s`, covering queueing, connection and operation |

`check` validates fields/reference syntax without reading a password or probing VNC. Initialization persists password references; connection validates the classic VNC password limit of eight printable ASCII bytes. See the [VNC guide](vnc.md) for setup, authentication scope and operation bounds.

### Credential references

- `env:NAME` reads an environment variable.
- `file:PATH` reads a file and removes trailing CR/LF bytes, preserving other whitespace.
- An absent optional reference resolves to an empty value. A present but empty/missing source fails. Inline plaintext is rejected when a credential is resolved.
- `check` does not resolve credentials; use it for structure and defaults only. `run`/`doctor` resolve the tunnel key before handing its reference to the supervisor. SSH credentials are resolved when connecting.
- Initialization captures referenced credentials into files with mode 0600 and writes references pinned to that generation. Imported private-key format/passphrase is validated locally; SSH login is not attempted.
- Docker's exact built-in `dev`/`developer` profile may reference `/var/lib/tunnel-ssh/client_key` before it exists: the development container generates it on first startup. Other imported keys must exist.

Use [config.example.toml](../config.example.toml) for native/manual operation and [deploy/config.init.toml](../deploy/config.init.toml) for Docker imports. The existing `deploy/config.docker.toml` is a local regression-test fixture, not the production initialization template.

## Commands

| Command | Purpose and validation boundary |
|---|---|
| `version` | Print the MCP application version |
| `check --config PATH` | Parse, apply defaults and validate structure; no credential resolution or connection |
| `serve --config PATH` | Serve MCP on stdio; does not authenticate callers or start a tunnel |
| `run --config PATH` | Replace the process with the official supervisor, launching this executable as its stdio worker |
| `doctor --config PATH` | Run official diagnostics with `--explain`; a diagnostic success is not an end-to-end tool call |
| `install-tunnel --dir DIR --version VERSION` | Download and verify the official bundle; defaults `bin`, `v0.0.19`; supports macOS/Linux amd64/arm64 |
| `init` | Prepare a native installation and register/enable/start its system service |
| `init --container` | Publish persisted container configuration without starting services or installing a host binary |

All flag-based subcommands accept `--help` and exit successfully without configuration loading, installation or network access. Unknown flags still fail.

### Initialization flags

| Flag | Default | Scope / behavior |
|---|---|---|
| `--config PATH` | Empty | Import TOML; otherwise reuse current configuration or prompt on first initialization |
| `--non-interactive` | False | Require imported or existing configuration instead of prompting |
| `--service-user USER` | Sudo caller, or current account | Native only; requires an existing local user |
| `--container` | False | Select container initialization |
| `--output DIR` | `/etc/tunnel-ssh` | Container only; dedicated directory other than `/`, shared with runtime |
| `--owner-uid`, `--owner-gid` | 1000 each | Container only; allowed range 0–2147483647 |
| `--reconfigure` | False | Container only; reopen the wizard; cannot combine with `--config` or `--non-interactive` |

The Docker initializer runs as root to set ownership for the non-root runtime account. Native installation announces its system paths and elevates via sudo if necessary.

Native generations live under `/usr/local/lib/tunnel-ssh-mcp/releases/`; container generations live under the configured `generations/`. `current` selects the active generation. Preparation completes before publication. Native activation stops the old service first and attempts recovery if activation fails. Container init only publishes configuration; restart MCP explicitly to load it. Neither success message proves live tunnel or SSH access.

## Docker variables

`.env.example` contains host paths and image tags, not credentials:

| Variable | Default | Mount / use |
|---|---|---|
| `TUNNEL_SSH_CONFIG_DIR` | `./data` | `/etc/tunnel-ssh` in init and runtime; runtime is read-only |
| `TUNNEL_SSH_INIT_INPUT_DIR` | `./init-input` | `/input` read-only in init only |
| `TUNNEL_SSH_WORKSPACE_DIR` | `./workspace` | `/workspace` in both runtime and development containers |
| `TUNNEL_SSH_MCP_IMAGE` | `go-tunnel-ssh-mcp:latest` | Image tag shared by initialization and MCP runtime |
| `TUNNEL_SSH_DEV_IMAGE` | `go-tunnel-ssh-dev:latest` | Development image tag |
| `TUNNEL_CONSOLE_DESKTOP_IMAGE` | `go-tunnel-console-desktop:latest` | Optional desktop overlay image, replacing the development image |

Use the same configuration mount for initialization and runtime. Keep `data/` permissions and symlinks when backing up. `developer_home` and `ssh_identity` are persistent named volumes; `docker compose down -v` removes them.

## Interpreting readiness

For the standard Docker configuration, check the running supervisor without creating another tunnel instance:

```sh
docker compose exec -T mcp tunnel-console-mcp version
docker compose exec -T mcp tunnel-client health --port 8080 --require-control-plane-poll --json
```

The health port must match the configured listener. Readiness may precede the first successful control-plane poll; require the poll check and inspect startup logs. Calling `doctor` inside the running container can conflict with the occupied diagnostics port; the deployment record used an independent one-shot diagnostic container. Finally verify actual MCP tool calls through ChatGPT. Keep local stdio tests, tunnel health and end-to-end results separate in reports.
