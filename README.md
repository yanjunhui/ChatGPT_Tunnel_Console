<!-- Created by Yanjunhui -->
# ChatGPT Tunnel Console

English | [简体中文](README.zh-CN.md)

Run commands, transfer files, and control Linux desktops from ChatGPT or Codex through OpenAI Tunnel.

ChatGPT Tunnel Console connects your assistant to a Linux development workspace or an existing SSH host. It provides an MCP server written in Go, with persistent terminals, SFTP transfers and desktop controls. The bundled Docker environment includes the tools needed to build and run software; you can also connect your own hosts using SSH profiles.

## What you can do

| Capability | Typical use |
|---|---|
| Remote commands | Inspect files, run scripts, build projects and read exit status |
| Persistent terminals | Keep shell state across calls, interact with programs and follow background jobs |
| File transfers | Browse directories, read and write files, upload ChatGPT attachments and stream larger files |
| Linux desktops | Capture an X11 desktop and operate its mouse and keyboard directly from the conversation |
| Multiple hosts | Select a named SSH profile for each operation |

Native desktop controls use SSH and X11. Optional VNC tools connect to the bundled desktop or a compatible existing VNC server. See the [desktop guide](docs/desktop.md) for setup and supported environments.

## Quick start

You need Docker with Compose v2, an OpenAI tunnel ID, and a runtime API key with **Read + Use** permission for that tunnel. Create a tunnel in [OpenAI Platform settings](https://platform.openai.com/settings/organization/tunnels); account prerequisites and setup are covered in the [installation guide](docs/installation.md).

```sh
git clone https://github.com/yanjunhui/ChatGPT_Tunnel_Console.git
cd ChatGPT_Tunnel_Console
mkdir -p data init-input workspace
docker compose -f compose.init.yaml run --rm --build init
```

Enter the tunnel ID and runtime key, then select the bundled command-line development environment. The initializer saves configuration and credentials under `data/`. Start the services:

```sh
docker compose up -d --build
```

In ChatGPT, create a custom MCP plugin with connection type **Tunnel**, select the same tunnel ID, and choose **No authentication**. Install and enable the plugin, then ask:

> List my SSH connections, then show the current user and working directory on local-dev.

The bundled environment runs as `developer` in `/workspace`, which maps to this checkout's `workspace/` directory. It includes Go, Python, Node.js, Git and common build tools. Follow [Connect ChatGPT](docs/installation.md#connect-chatgpt) for the full connection steps.

### Add a graphical desktop

During initialization, choose **3: bundled graphical desktop**. Leave optional VNC at **n** for native desktop controls, then start with the desktop overlay:

```sh
docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

For an existing installation, update its configuration as described in the [desktop guide](docs/desktop.md) before starting the overlay. This starts a 1280×800 Linux desktop with Openbox and xterm. Ask for a screenshot, then use mouse and keyboard actions to work with it. To add VNC, see [VNC setup](docs/vnc.md).

### Use your own SSH host

The Docker initializer also accepts an external SSH host. Alternatively, install a native service on macOS or Linux:

```sh
make build
./bin/tunnel-console-mcp init
```

Native builds require Go 1.26.10 or automatic toolchain download. The initializer configures SSH credentials and registers a system service. See [native installation](docs/installation.md#native-installation-for-existing-ssh-hosts) for supported platforms, service paths and manual operation.

## Everyday workflows

You can ask the connected assistant to:

- “Run the project's tests in `/workspace/my-project` and show any failures.”
- “Start this build in a background session and check its output.”
- “Upload the attached CSV to `/workspace/import/input.csv`.”
- “Take a desktop screenshot, focus the terminal, and type this command.”

Choose an existing project or create the destination directory first. Attachments are supplied by ChatGPT directly to the upload tool; you do not need to paste file contents or temporary download links. See [commands and sessions](docs/usage.md), [attachment uploads](docs/attachments.md), and the [tool reference](docs/tool-reference.md) for parameters and examples.

## How it connects

```text
ChatGPT / Codex
      |
OpenAI Tunnel <-- outbound HTTPS <-- tunnel-client
                                        | stdio
                                  Console MCP (Go)
                                        | SSH / SFTP
                                  Linux workspace or SSH host
                                        | X11 / optional VNC
                                  Desktop applications
```

The service establishes an outbound tunnel connection; it does not expose an MCP HTTP port. The default Docker setup publishes neither SSH nor MCP ports. SSH profiles determine the target host, account and working directory.

Callers have the SSH account's permissions, including sudo where configured. Callers of one deployment share profiles, sessions and desktops. SSH host keys are currently **not verified**. Review the [security policy](SECURITY.md) and use separate deployments for separate trust domains.

## Documentation

| Guide | Contents |
|---|---|
| [Installation and upgrades](docs/installation.md) | Docker, native services, ChatGPT connection, backups and upgrades |
| [Commands, sessions and files](docs/usage.md) | Common workflows and working-directory behavior |
| [ChatGPT attachments](docs/attachments.md) | Uploads, destination paths and transfer limits |
| [Native desktop](docs/desktop.md) · [VNC](docs/vnc.md) | Desktop setup, screenshots and input |
| [Configuration](docs/configuration.md) · [Tool reference](docs/tool-reference.md) | CLI options, TOML settings and all 21 MCP tools |
| [Troubleshooting](docs/troubleshooting.md) | Connection, tool discovery, transfer and desktop problems |

Browse the [documentation index](docs/README.md) for all guides. Development setup and checks are in [Contributing](CONTRIBUTING.md); release changes are in the [changelog](CHANGELOG.md).

## License

[MIT](LICENSE). This is an independent project using the [official OpenAI tunnel-client](https://github.com/openai/tunnel-client) and [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk). Dependency licenses and acknowledgements are listed in [third-party notices](THIRD_PARTY_NOTICES.md).
