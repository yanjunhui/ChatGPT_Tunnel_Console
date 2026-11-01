<!-- Created by Yanjunhui -->
# Native desktop and independent VNC

English | [简体中文](desktop.zh-CN.md) | [README](../README.md)

From 0.3.0, ChatGPT_Tunnel_Console provides screenshots and keyboard/mouse control through its own `desktop-*` MCP tools without a VNC service. Users start this project and operate through ChatGPT/Codex; a separate viewer is unnecessary. Optional `vnc-*` tools provide a second, independently configured path. Neither path automatically falls back to the other.

| Capability | Native desktop | Optional VNC |
|---|---|---|
| Transport | MCP tunnel → Go → SSH → xwd/xdotool | MCP tunnel → Go → SSH forwarding → VNC |
| Configuration | `profiles.NAME.desktop` | `profiles.NAME.vnc` |
| Target requirement | Accessible Linux X11 display, xwd, xdotool, C.UTF-8 | Existing compatible RFB server |
| Bundled service | `dev`: Xvfb, Openbox, xterm and SSH | Separate `vnc`: x11vnc |
| Other backend disabled/stopped | Continues working | Continues working with an available display |

The two paths may share a display and complement each other. Independence refers to backend configuration, protocol and service lifecycle, not separate desktop sessions: both still require SSH, and the bundled VNC service needs the X display owned by `dev`. Stopping `dev` stops access to that desktop through both paths. Disabling the native tool setting does not stop Xvfb or VNC. Calls share a queue inside one MCP process; external viewers are not locked out.

## Start only the native desktop

Run the initializer, choose **3**, then answer **n** (default) for optional VNC:

```sh
mkdir -p data init-input workspace
docker compose -f compose.init.yaml run --rm --build init
docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

The image has no VNC server installed. It creates a 1280×800, depth-24 Xvfb display `:1`, an Openbox window manager and an xterm terminal. The X server has no TCP listener. Native capture uses `xwd` through an SSH exec channel; Go validates and encodes the original framebuffer without scaling. Input uses XTEST through `xdotool`; text travels on stdin with a UTF-8 locale, not in process arguments or the clipboard. This backend targets Linux X11, not native Wayland, macOS or Windows desktop capture.

For an existing deployment, back up configuration, workspace and persistent volumes first. Stop MCP before importing a complete configuration; retain existing profiles and credentials. The relevant additions are:

```toml
[profiles.local-dev.desktop]
enabled = true
display = ":1"
timeout = "15s"
```

Use `compose.init.yaml` with `--non-interactive --config /input/config.toml`, then start with the desktop overlay. Existing profiles do not implicitly enable either backend. For external Linux targets, install the utilities and ensure the SSH account already has access to the selected X display and its authentication environment. Do not broadly disable display access control to make a probe pass.

## Add or stop complementary VNC

Choose **y** in the initializer's optional VNC question, or import this additional section:

```toml
[profiles.local-dev.vnc]
enabled = true
host = "127.0.0.1"
port = 5901
timeout = "15s"
```

Manage the same deployment with all three files:

```sh
docker compose -f compose.yaml -f compose.desktop.yaml -f compose.vnc.yaml up -d --build
docker compose -f compose.yaml -f compose.desktop.yaml -f compose.vnc.yaml stop vnc
docker compose -f compose.yaml -f compose.desktop.yaml -f compose.vnc.yaml start vnc
```

Stopping `vnc` leaves SSH, the X display and native tools running. Its health check is independent; neither `dev` nor `mcp` depends on VNC readiness. The optional service shares the X11 socket and `dev` network namespace, listens on loopback 5901 and publishes no host port. When recreating `dev`, use `up -d` with all three files so Compose also updates VNC's network attachment. Standard VNC viewers may connect through an explicitly configured SSH forward; no viewer is required for MCP operation. See [VNC authentication and protocol limits](vnc.md).

To avoid accidentally omitting overlays later, an optional local `.env` can contain `COMPOSE_FILE=compose.yaml:compose.desktop.yaml:compose.vnc.yaml` on macOS/Linux. Keep the chosen image tags consistent between init and runtime; `.env.example` lists them. The desktop socket volume is runtime state, not user data. Preserve `developer_home`, `ssh_identity`, `data/` and `workspace/`; never use `down -v` for data that must survive.

## Operate through MCP

Each line is a separate call. First inspect the screenshot and choose coordinates in its original pixels:

```text
list-connections    {}
desktop-screenshot  {"profile":"local-dev"}
desktop-pointer     {"profile":"local-dev","events":[{"x":100,"y":100,"buttons":1}]}
desktop-key         {"profile":"local-dev","keys":["Control_L","u"]}
desktop-type        {"profile":"local-dev","text":"printf 'DESKTOP_OK\\n'"}
desktop-key         {"profile":"local-dev","keys":["Return"]}
desktop-screenshot  {"profile":"local-dev"}
```

`list-connections` reports `desktop_enabled` and `vnc_enabled` separately; these are configuration flags, not live probes. Screenshots return MCP PNG image content and structured width/height/name. Input results only acknowledge event delivery. Re-capture and verify the application effect; a wrong focus can send input elsewhere. Use the same argument shapes with `vnc-*` when deliberately selecting that path. Only act on the desktop and applications authorized for the task.

Both paths enforce 1–256 pointer events, 1–16 keys, 1–4096 Unicode characters, an 8192-pixel side limit, 8,388,608 total pixels, a 16 MiB PNG limit and a default 15-second operation deadline (maximum 120 seconds). Key names, button bits and error behavior are in the [tool reference](tool-reference.md). Native capture requires 24-bit TrueColor with 24/32-bit storage. Invalid/truncated frames return errors without a partial image. Unicode rendering still depends on the target application and input method.

If ChatGPT reports missing tools after an upgrade, refresh the existing app's tools in its management page and re-check discovery. Local tests and tunnel health alone do not prove ChatGPT has discovered the new schema. Do not create another tunnel supervisor with the same ID for this check.

Native typing uses a 12 ms key delay so applications can consume temporary Unicode key mappings. Long text may need a larger `desktop.timeout` (up to 120 seconds).
