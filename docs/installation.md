<!-- Created by Yanjunhui -->
# Installation and upgrades

English | [简体中文](installation.zh-CN.md) | [Documentation](README.md)

Choose Docker for a bundled Linux workspace, or a native service on macOS/Linux to connect existing SSH hosts. Both use an outbound OpenAI Tunnel connection and require a tunnel ID and runtime API key.

## Prepare an OpenAI tunnel

1. Open [OpenAI Platform tunnel settings](https://platform.openai.com/settings/organization/tunnels) in the organization associated with your ChatGPT workspace or personal account.
2. Create a tunnel. Creation requires Read + Manage permissions; the key used by this service needs Read + Use for that tunnel.
3. Keep the tunnel ID and runtime key ready for initialization. Store credentials outside Git and pass them through the initializer or credential-file references.

The service host needs outbound HTTPS access to OpenAI Tunnel and network access to its SSH targets. Docker builds and native client installation also need access to package and release downloads. No inbound MCP port is required. See the [OpenAI Tunnel setup guide](https://developers.openai.com/api/docs/guides/secure-mcp-tunnels) for account prerequisites and permissions.

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

4. Follow [Connect ChatGPT](#connect-chatgpt), then ask it to run `id -un; pwd; go version`. The bundled environment returns `developer`, `/workspace` and its Go version.

The `dev` container includes Go, Python, Node.js/npm, Git, C/C++, CMake, Ninja, tmux, rg, fd-find, jq and tree. `developer` has passwordless sudo inside the container. Container permissions are not host permissions; changes in `/workspace` affect the same files under the project's `workspace/` directory.

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

Use `TUNNEL_SSH_MCP_IMAGE` and `TUNNEL_SSH_DEV_IMAGE` to choose custom image tags; default tags remain `go-tunnel-ssh-mcp:latest` and `go-tunnel-ssh-dev:latest`. Set the MCP tag consistently for initialization and runtime.

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

Copy the [import template](../deploy/config.init.toml) to `init-input/config.toml`, set the tunnel ID, and save the runtime API key as `init-input/openai-tunnel-key` containing only the key. The template uses the bundled development environment; the [native template](../config.example.toml) also shows how to configure multiple external profiles. Use file references under `/input/` for imported credentials.

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

After installation, follow [Connect ChatGPT](#connect-chatgpt) and run a command to confirm tunnel and SSH connectivity. A macOS cold boot with FileVault requires disk unlock first.

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

OpenAI Tunnel authenticates MCP callers; SSH credentials authenticate to the target host. The service currently does not verify SSH host keys. Review the [security policy](../SECURITY.md) before connecting external hosts.

## Connect ChatGPT

1. In ChatGPT, open **Plugins**, choose **+ → Add custom MCP server**, and select **Tunnel** as the connection method.
2. Select or enter the tunnel ID configured during installation. Choose **No authentication** for the MCP server; OpenAI Tunnel handles caller authentication.
3. Create and install the plugin, then enable it in your chat. If your Codex account has access to the connected plugin, enable it there as well.
4. Ask to list the SSH connections, then run `pwd` on the selected profile. `list-connections` lists configuration; the command confirms the SSH connection works.

The tunnel must belong to the Platform organization linked to the intended account/workspace. If a tunnel or plugin is missing, check that account association first. See [troubleshooting](troubleshooting.md) for discovery and connection problems.

## Back up and upgrade

Before an upgrade, save the deployed Git revision, keep the previous images or binaries, and back up configuration and user data. For Docker, preserve the entire `data/` directory, `workspace/`, and the `developer_home` and `ssh_identity` volumes. Include the chosen Compose overlays and local `.env` in your private backup. Preserve file ownership, permissions and symlinks; configuration backups contain credentials.

### Docker

From the existing checkout, stop MCP, update to your chosen revision, then rebuild and recreate the services. For the default command-line environment:

```sh
docker compose stop mcp
git pull --ff-only
docker compose up -d --build
```

For a desktop deployment, include the same desktop/VNC overlays in all management commands, or retain them in `COMPOSE_FILE`. Keep the checkout path, Compose project name and volume identities stable so the deployment reuses its data. A source-only upgrade does not require reinitialization. To change configuration, follow [Reinitialization and external SSH](#reinitialization-and-external-ssh); use a complete TOML import to preserve multiple profiles.

### Native service

Update the source, rebuild, and run the installer again:

```sh
git pull --ff-only
make build
./bin/tunnel-console-mcp init
```

The installer reuses saved configuration and retains previous installation generations. If the service uses a custom account, keep passing `--service-user USER`. Check the service logs and run a command through the plugin after the upgrade.

### Tool discovery and compatibility

After an upgrade that changes tools or their parameters, refresh the installed plugin's tools on its management page. Check the exact plugin and account used in the chat; reloading the browser does not update another account's tool catalog. See [refreshing tool metadata](https://developers.openai.com/plugins/deploy/connect-chatgpt#refresh-metadata).

The primary command is `tunnel-console-mcp`; `tunnel-ssh-mcp` remains an alias. Native service paths, the Compose project name `go-tunnel-ssh`, persistent volume names and `TUNNEL_SSH_*` variables retain their original names for existing installations. You can keep an existing checkout directory name. Desktop and VNC are enabled explicitly in configuration; updating the program does not enable them.

Run only one stdio tunnel client per tunnel ID, including during rollback. Stop the new instance before restoring a saved revision and rebuilding, or restoring the previous images/binaries and configuration. Restarting MCP clears in-memory sessions. Check the [changelog](../CHANGELOG.md) for version-specific changes and use [troubleshooting](troubleshooting.md) if the service or tool catalog is unexpected.
