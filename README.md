<!-- Created by Yanjunhui -->
# ChatGPT Tunnel Console · Go

English | [简体中文](README.zh-CN.md)

An independent Go console for SSH, SFTP, native X11 desktops and optional VNC access, exposed as an MCP server through the official OpenAI tunnel. The tunnel authenticates MCP callers. The server adds no second Bearer token, OAuth layer, roles, approval system, command classification, allow/deny lists, path sandbox or daily quota.

The feature scope draws on `ssh-mcp-main 2` and the deployment approach on `SSH-MCP-main`. This project neither copies nor wraps their TypeScript/Python implementations and does not depend on npm ssh-mcp. MCP, SSH, sessions, SFTP, VNC, configuration and startup are implemented in Go; the tunnel uses the official tunnel-client.

```text
ChatGPT / Codex
      | OpenAI tunnel authentication
      v
OpenAI Tunnel <-- outbound connection <-- official tunnel-client
                                                 | stdio
                                                 v
                                            Go Console MCP
                                                 | SSH / SFTP / native X11 / forwarded VNC
                                                 v
                                  Docker development environment / SSH host
```

There is no MCP HTTP listener. `serve` is a local stdio worker launched by tunnel-client and also used by tests; it does not authenticate callers itself. Do not separately expose it through HTTP, TCP or socat. The official tunnel diagnostics bind only to loopback.

Current source version: **0.3.0**. This independent project is licensed under [MIT](LICENSE). See the [security policy](SECURITY.md) before deployment: authorized callers have the SSH account’s full permissions, and SSH target host keys are not verified.

## Documentation

- [Configuration and CLI reference](docs/configuration.md): defaults, credential references, initialization and validation boundaries.
- [VNC desktop guide](docs/vnc.md): setup, screenshots, mouse/keyboard operations and limits.
- [MCP tool reference](docs/tool-reference.md): parameters, results and errors for all 21 tools.
- [Verification record](docs/verification.md) and [live deployment record](docs/live-deployment.md): historical tests and actual tunnel acceptance evidence.

- [Contributing](CONTRIBUTING.md), [security policy](SECURITY.md), [changelog](CHANGELOG.md) and [third-party notices](THIRD_PARTY_NOTICES.md).

## Implemented tools

| Tool | Behavior |
|---|---|
| `list-connections` | List SSH profiles and retained-session counts without credentials or an implied connectivity probe |
| `run-command` | Execute single-line or multiline shell commands with stdin, optional PTY and timeout override; return stdout, stderr and exit status |
| `open-session` | Create an interactive PTY or background command |
| `write-session-input` | Send raw input, newlines or control characters, optionally closing stdin |
| `read-session-output` | Read by absolute byte cursor and report discarded bytes |
| `list-sessions` | List retained running/exited sessions and exit information |
| `close-session` | Request TERM and disconnect; report whether an exit status was observed |
| `signal-process` | Send INT / TERM / KILL to a positive remote PID |
| `sftp-list` | List a directory through SFTP |
| `sftp-download` | Read UTF-8 or base64 content |
| `sftp-upload` | Write UTF-8 or base64 content, overwriting the remote file |
| `sftp-download-file` | Stream to the MCP server filesystem, replacing the local destination on success |
| `sftp-upload-file` | Stream from the MCP server filesystem, overwriting the remote destination |
| `desktop-screenshot` | Capture the native X11 desktop as PNG without VNC |
| `desktop-pointer` | Move, click, drag or scroll directly through X11 |
| `desktop-key` | Press and release a chord directly through X11 |
| `desktop-type` | Type Unicode directly without clipboard access |
| `vnc-screenshot` | Return a native PNG as MCP image content plus desktop metadata |
| `vnc-pointer` | Move, click, drag or scroll using screenshot pixel coordinates |
| `vnc-key` | Press and release one key or keyboard chord |
| `vnc-type` | Type Unicode text using key events, without clipboard access |

Since 0.1.1, `sftp-download` publishes an `outputSchema` and returns both MCP `structuredContent` and the existing `content[0].text` JSON. Both contain `content` (text or base64), `encoding`, and `bytes` (original file bytes). Empty files explicitly return an empty string and `0`. Callers may use structured fields or continue decoding the text JSON. Path, encoding, UTF-8 and inline-size failures return `isError: true` with error text and no successful file payload.

The old `read-command`, `privileged-command` and `reload-config` interfaces are not retained. All independent commands use `run-command`; sudo follows the target account's actual permissions. Restart to load configuration changes. Arbitrary shell tools are not labeled read-only.

## Independent native desktop and optional VNC

Choose **3: bundled graphical desktop** in the Docker initializer. Leave the optional VNC question at **n** for native desktop only, then use:

```sh
docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

This adds a 1280×800 Xvfb/Openbox desktop and xterm terminal. The four `desktop-*` tools capture and control it directly through SSH, without a VNC server or client. To add complementary VNC access, answer **y** and include `-f compose.vnc.yaml`. That separately managed x11vnc service attaches to the same display on loopback port 5901; stopping it leaves native tools operational. No host ports are published. See the [native desktop guide](docs/desktop.md) and [VNC guide](docs/vnc.md).

### Name and upgrade compatibility

The repository, Go module, MCP implementation and primary CLI are now `ChatGPT_Tunnel_Console`, `github.com/yanjunhui/ChatGPT_Tunnel_Console`, `chatgpt-tunnel-console` and `tunnel-console-mcp`. The old CLI alias remains available in local builds and runtime images. Existing native installation/service paths, Compose project name `go-tunnel-ssh`, volume identities and `TUNNEL_SSH_*` settings remain stable so upgrades reuse data. Existing checkout directories need not be renamed, especially when live bind mounts reference them. New checkouts use the new directory name. VNC is disabled in existing configurations; this source release does not upgrade running services automatically.

## Docker Linux development environment

Requires Docker Engine / Docker Desktop with Compose v2. The first build needs network access.

Get the source before running commands below:

```sh
git clone https://github.com/yanjunhui/ChatGPT_Tunnel_Console.git
cd ChatGPT_Tunnel_Console
```


1. Run the one-shot initializer from the project root:

   ```sh
   mkdir -p data init-input workspace
   docker compose -f compose.init.yaml run --rm --build init
   ```

2. Enter the tunnel ID and runtime API key, then choose an SSH target. The bundled development environment is the default; key input is hidden. Use runtime credentials with Read + Use permissions for this tunnel, not an administrative API key. The wizard saves configuration and credentials in `data/` and exits. `--rm` removes only the one-shot container.

3. Build and start:

   ```sh
   docker compose up -d --build
   docker compose logs --tail=100 mcp
   ```

4. Connect the OpenAI tunnel in a custom ChatGPT app, with no additional app authentication. Enable the app, call `list-connections`, then use `run-command` for `id -un; pwd; go version`. Expect `developer` and `/workspace`. This step verifies the complete ChatGPT → tunnel → MCP → SSH path.

The `dev` container includes Go, Python, Node.js/npm, Git, C/C++, CMake, Ninja, tmux, rg, fd-find, jq and tree. `developer` has passwordless sudo, consistent with the design's lack of application operation policies. Container permissions are not host permissions; changes in `/workspace` affect the same files under the project's `workspace/` directory.

The `mcp` container runs only the Go MCP and official tunnel components. The containers use an automatically generated SSH key. Neither SSH nor MCP has a published host port. `workspace/` is shared and also usable for streamed file transfers. The SSH identity and developer home are stored in named volumes.

`compose.init.yaml` is separate from the runtime Compose file. It needs no existing configuration, credential-bearing `.env` or running development container. The initializer writes files as container root and assigns them to runtime UID/GID 1000. The runtime MCP uses that non-root account and mounts configuration read-only. Initialization does not register host systemd/launchd services, need the Docker socket or start a tunnel.

| Persisted content | Location |
|---|---|
| Current configuration | `data/current/config.toml`; `/etc/tunnel-ssh/current/config.toml` inside containers |
| Configuration and credential generations | `data/generations/init-*/`; files mode 0600, generation and secret directories mode 0700 |
| Import files | `init-input/`, mounted read-only at `/input/`; runtime does not depend on them after import |
| Shared working files | `workspace/` |
| Developer home / generated SSH identity | Compose named volumes `developer_home` / `ssh_identity` |

These paths are excluded from Git and image build context. `current` is a relative symlink; credential references point to a specific generation. Failed reinitialization does not replace the active configuration. Back up the entire `data/` directory, preserving permissions and symlinks. Older generations remain available to running processes.

On native Linux, bind mounts preserve numeric ownership. Give runtime UID/GID 1000 write access to the dedicated host workspace; for a newly created `./workspace`, `sudo chown 1000:1000 ./workspace` sets its owner. Apply an appropriate ACL instead when ownership must stay with another host account. Existing files retain their own permissions. Protected configuration generations may require elevated host permissions to back up or remove; do not make credentials world-readable. Docker Desktop file sharing can behave differently.

Optionally copy `.env.example` to `.env` and set `TUNNEL_SSH_CONFIG_DIR`, `TUNNEL_SSH_INIT_INPUT_DIR` and `TUNNEL_SSH_WORKSPACE_DIR` to change host paths. Initialization and runtime must share the same configuration directory, mounted at `/etc/tunnel-ssh` in both. Do not put secrets in `.env`.

Use `TUNNEL_SSH_MCP_IMAGE` and `TUNNEL_SSH_DEV_IMAGE` to give acceptance builds separate image tags; default tags remain `go-tunnel-ssh-mcp:latest` and `go-tunnel-ssh-dev:latest`. Set the MCP tag consistently for initialization and runtime.

### Reinitialization and external SSH

Running initialization again reuses saved configuration by default. To reopen the wizard:

```sh
docker compose stop mcp
docker compose -f compose.init.yaml run --rm init --reconfigure
docker compose up -d --build
```

If initialization fails, the old configuration remains available; `docker compose start mcp` resumes the old service. Configuration is not hot-reloaded. Restarting MCP ends in-memory sessions.

For an external SSH host, enter its host, port, user, working directory and authentication method. Place imported private keys under `init-input/` and enter their `/input/...` path in the wizard. Passwords and passphrases are hidden. The initializer validates and copies keys and credentials; runtime no longer needs the import files. When only external hosts are used, start MCP alone:

```sh
docker compose up -d --build --no-deps mcp
```

### Non-interactive initialization

Copy the [import template](deploy/config.init.toml) to `init-input/config.toml`, set the tunnel ID, and save the runtime API key as `init-input/openai-tunnel-key` containing only the key. The template uses the bundled development environment; the [native template](config.example.toml) also shows how to configure multiple external profiles. Use file references under `/input/` for imported credentials.

```sh
docker compose -f compose.init.yaml run --rm --build -T init \
  --non-interactive --config /input/config.toml
docker compose up -d --build
```

First-time non-interactive initialization requires an import file. Once configuration exists, `--config` can be omitted. `--reconfigure` cannot be combined with `--config` or `--non-interactive`. Initialization success confirms persistence, structure, credential-file and private-key format checks; it does not confirm tunnel authentication or SSH connectivity.

### Container management

```sh
docker compose exec --user developer dev bash  # Enter the development environment
docker compose logs -f mcp                    # Follow tunnel logs
docker compose stop                          # Stop, retaining data
docker compose start                         # Start existing containers
docker compose down                          # Remove containers, retaining named volumes
```

Do not use `docker compose down -v` for data you need: it deletes the developer home and SSH identity volumes.

Both services use `restart: unless-stopped`. Docker manages recovery after process failures or engine restarts; configure the host to start Docker Engine/Desktop automatically. Manually stopped containers remain stopped under this policy. See [Docker restart policies](https://docs.docker.com/engine/containers/start-containers-automatically/).

## Native installation for existing SSH hosts

Supports macOS/Linux on amd64/arm64. The module requires Go 1.26.10 or newer; older Go versions with automatic toolchain support can download the toolchain requested by `go.mod`. The Makefile keeps build, module and temporary caches under `.cache/`.

### Initialize and register a boot service

```sh
make build
./bin/tunnel-console-mcp init
```

`init` prompts for the OpenAI tunnel ID, runtime key, SSH host, port, user, working directory, and private key or password. Secrets are not echoed; Ctrl+C restores terminal state. The installer announces system paths before writing and uses `sudo` when the invoking account lacks installation privileges.

It copies the Go executable, official tunnel client, configuration and SSH credentials to fixed locations. If no reusable tunnel-client is found, it downloads and verifies the official bundle. Once staging completes, it stops the old service, switches the installation, registers/enables the service and starts it:

- macOS: system LaunchDaemon `com.yanjunhui.tunnel-ssh-mcp`, independent of user login.
- Linux: systemd `tunnel-ssh-mcp.service`, enabled for `multi-user.target`.

The default runtime user is the original `sudo` caller; direct root invocation uses root. Set `--service-user USER` to choose another local account. Service HOME and working directory are `/var/lib/tunnel-ssh-mcp`; an interactive shell environment is not required.

| Content | Fixed location |
|---|---|
| Current executable and configuration | `/usr/local/lib/tunnel-ssh-mcp/current/` |
| Configuration | `/usr/local/lib/tunnel-ssh-mcp/current/config.toml` |
| Installation generations and private credentials | `/usr/local/lib/tunnel-ssh-mcp/releases/`; config/credential files mode 0600 |
| Service working directory | `/var/lib/tunnel-ssh-mcp` |
| macOS service definition | `/Library/LaunchDaemons/com.yanjunhui.tunnel-ssh-mcp.plist` |
| macOS logs | `/var/log/tunnel-ssh-mcp/stdout.log`, `stderr.log` |
| Linux service definition | `/etc/systemd/system/tunnel-ssh-mcp.service` |
| Linux logs | `journalctl -u tunnel-ssh-mcp.service` |

Import existing configuration, including multiple profiles:

```sh
./bin/tunnel-console-mcp init --config ./config.toml
sudo ./bin/tunnel-console-mcp init --non-interactive --config /absolute/path/config.toml
```

The initializer resolves `env:` credentials and persists file references for service restarts. Because sudo may remove environment variables, prefer `file:` references for non-interactive imports. Keys are never inserted into sudo arguments. Boot services cannot depend on a login session's `SSH_AUTH_SOCK`; use private-key files or passwords. Manual `run` mode still supports an SSH agent.

Repeated `init` reuses installed configuration and updates the program without re-entering credentials; `--config` imports replacements. Older generations are retained. A failed activation attempts to restore the previous configuration and service and returns an error; failed recovery is also reported. An installation lock prevents concurrent initialization.

**Successful initialization confirms registration, boot enablement and process startup, not OpenAI authentication or SSH connectivity.** Still perform actual ChatGPT tool calls. A macOS cold boot with FileVault requires disk unlock first.

Inspect or restart the service:

```sh
# macOS
sudo launchctl print system/com.yanjunhui.tunnel-ssh-mcp
sudo launchctl kickstart -k system/com.yanjunhui.tunnel-ssh-mcp

# Linux
sudo systemctl status tunnel-ssh-mcp.service
sudo systemctl restart tunnel-ssh-mcp.service
```

Stop and disable boot startup while retaining files:

```sh
# macOS
sudo launchctl disable system/com.yanjunhui.tunnel-ssh-mcp
sudo launchctl bootout system/com.yanjunhui.tunnel-ssh-mcp

# Linux
sudo systemctl disable --now tunnel-ssh-mcp.service
```

`init` without `--container` manages native services. Docker uses the separate initialization Compose file, which invokes `init --container`; runtime Compose manages startup.

### Manual foreground operation

```sh
make build
make install-tunnel
cp config.example.toml config.toml
```

Edit SSH `host`, `user`, `private_key`, `workdir` and tunnel settings. Relative paths resolve against the configuration file's directory; TOML paths do not expand `~`. Passwords and passphrases must use `env:NAME` or `file:/absolute/path` references, not inline plaintext. OpenSSH private keys are supported. Set `agent_socket = "env:SSH_AUTH_SOCK"` to use an agent.

```sh
./bin/tunnel-console-mcp check --config config.toml
./bin/tunnel-console-mcp doctor --config config.toml
./bin/tunnel-console-mcp run --config config.toml
```

`check` validates configuration structure, not credentials or remote connectivity. `doctor` runs official diagnostics and contacts the tunnel service. `run` replaces itself with tunnel-client, which supervises the Go stdio worker. A restart loses in-memory sessions.

There is no additional MCP password or token. SSH credentials authenticate to the target host separately from MCP caller authentication. This version has no SSH host-key trust policy and does not check the target fingerprint. OpenAI Tunnel authenticates callers, not the external SSH host; understand that trust boundary when using external targets.

## Tool examples

```json
{"command":"pwd\nprintf 'Multiline script\\n'\nls -la","profile":"local-dev"}
```

Use separate calls for a persistent terminal:

```text
open-session         {"name":"work","type":"interactive"}
write-session-input  {"name":"work","data":"cd /workspace\nexport DEMO=hello\n"}
write-session-input  {"name":"work","data":"printf '%s\\n' \"$DEMO\"\n"}
read-session-output  {"name":"work","cursor":0}
```

For subsequent reads, use the previous `output.next_cursor`. `write-session-input` confirms delivery, not completion. Interactive output includes prompts, echo and control sequences. Each `run-command` uses a separate shell and does not inherit named-session state.

```text
open-session        {"name":"build","type":"background","command":"make build"}
read-session-output {"name":"build","cursor":0}
sftp-upload         {"path":"hello.txt","content":"Hello"}
sftp-download-file  {"path":"artifact.tar","local_path":"/workspace/artifact-copy.tar"}
```

Remote relative paths use the configured `workdir`. Parent directories are not created automatically. Interrupted uploads may leave partially written remote files, and the error reports this. Downloads to the MCP filesystem use a temporary file and replace the destination only on success.

## Runtime behavior and boundaries

- Run one stdio tunnel-client per tunnel ID. Stop the old instance before an upgrade starts the new one. See the [official stdio deployment limits](https://github.com/openai/tunnel-client/blob/v0.0.19/docs/configuration.md#stdio-deployment-limits).
- All callers share profiles and named sessions. There are no application roles or tenant isolation; separate trust domains need separate deployments and tunnel IDs.
- Independent commands default to 60 seconds. `timeout_seconds: 0` or profile `command_timeout = "0s"` disables that deadline. Background sessions do not use this short-command timeout.
- `output_bytes` defaults to 1 MiB; valid explicit limits are 1 byte through 64 MiB, with 0 selecting the default. This is a per-buffer memory bound, not a command quota. Overflow retains the tail and reports `dropped_bytes`. Larger inline downloads return an error directing callers to streaming tools.
- Cancellation or timeout closes the corresponding SSH connection. Disconnecting, requesting TERM and observing process exit are separate events; detached children may survive. Use `signal-process` with a known PID when needed.
- Commands and terminals target POSIX shells; there is no Windows PowerShell/cmd adaptation.
- The server does not retry started commands or uploads. Sessions are not silently recreated or restored.

## Development and verification

```sh
make check   # Formatting and bilingual documentation checks
make test    # Race checks, real local SSH/SFTP/PTY, MCP memory and stdio clients
make vet
make build
make linux   # Linux arm64 and amd64
TUNNEL_SSH_MCP_IMAGE=go-tunnel-ssh-mcp:review \
TUNNEL_SSH_DEV_IMAGE=go-tunnel-ssh-dev:review \
make docker-build docker-test # Container init, real wizard, persistence/permissions, OpenSSH/PTY/SFTP
```

Automated tests create temporary SSH hosts and files without using existing business hosts or live tunnels. The 0.3.0 container checks verify native input/capture before VNC starts, both paths together, and native operation after VNC stops. See the [verification record](docs/verification.md) and [deployment record](docs/live-deployment.md) for revision-specific checks, formal runtime status, real ChatGPT evidence and rollback.

Dependencies are locked in `go.mod` / `go.sum`. The official tunnel version is pinned to v0.0.19 and downloaded archives are checked against the published `SHA256SUMS.txt`. Caches and credentials are excluded from Git/Docker build context. Development tools come from official Go 1.26.10 and Node 24 images and Debian packages. APT packages and base-image tags are not pinned to full snapshots, so builds are not claimed to be byte-for-byte reproducible.

References: [official MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk/tree/v1.8.0), [official OpenAI tunnel-client](https://github.com/openai/tunnel-client/tree/v0.0.19).

## Documentation and comment conventions

English documents use the original filenames; full Simplified Chinese companions use `.zh-CN.md` and link back to the English version. Update both when behavior, fields or commands change. Example parameters, defaults, versions, hashes and acceptance conclusions must agree. Source comments put English first, followed by Chinese, explaining contracts, boundaries and design reasons instead of restating code. Translating documentation does not change runtime wizard or error-message languages. New files retain the `Created by Yanjunhui` header.
