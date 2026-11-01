<!-- Created by Yanjunhui -->
# VNC desktop over MCP

English | [简体中文](vnc.zh-CN.md) | [README](../README.md)

Version 0.2.0 adds desktop screenshots and input to the existing SSH/SFTP console. The transport is **OpenAI MCP tunnel → Go MCP server → SSH direct-tcpip → VNC server**. The official tunnel carries MCP messages, including PNG image content; it is not a native VNC/TCP tunnel. There is no web viewer or public VNC listener.

## Bundled Docker desktop

On a new deployment, run the initializer and choose **3: bundled graphical desktop**, then answer **y** to the independent VNC question:

```sh
mkdir -p data init-input workspace
docker compose -f compose.init.yaml run --rm --build init
docker compose -f compose.yaml -f compose.desktop.yaml -f compose.vnc.yaml up -d --build
```

Since 0.3.0, `compose.desktop.yaml` supplies Xvfb/Openbox/xterm and native desktop tools without installing VNC. `compose.vnc.yaml` adds a separate x11vnc service that shares the X11 socket and development container's network namespace. The 1280×800 display `:1` is shared by both paths; VNC listens on loopback port 5901. No host ports are published. Keep all three Compose files in subsequent management commands. Stop or restart `vnc` independently; native tools remain operational. The SSH identity, developer home and workspace remain unchanged. See [native desktop operation](desktop.md).

For an existing deployment, back up the complete configuration and persistent volumes, stop MCP, then run the initializer with `--reconfigure`, choose 3, and start with the overlay. Reconfiguration replaces the profile selection; import a complete TOML instead if multiple existing profiles must be retained. Initialization alone neither starts nor tests VNC. Do not run two supervisors for the same tunnel ID. Existing paths, Compose project identity and environment variables retain their prior names; see [upgrade compatibility](../README.md#name-and-upgrade-compatibility).

## Existing VNC server

Add the following section under the appropriate SSH profile in the imported configuration:

```toml
[profiles.server.vnc]
enabled = true
host = "127.0.0.1"
port = 5900
password = "file:/absolute/path/to/vnc-password"
timeout = "15s"
```

`host` is resolved/reached **from the SSH target**, not from ChatGPT or the MCP container. Use port 5901 for the bundled desktop and omit `password` there. For Docker imports, put any password file under `init-input/` and reference `/input/...`; init copies it into the private persisted generation. Import through the same initialization path as SSH configuration and restart MCP to load changes. The native wizard configures SSH; use TOML imports for external VNC settings.

The SSH server must allow `direct-tcpip` forwarding to the configured address. The VNC service must already be running and permit shared client connections. Supported protocol versions are RFB 3.3, 3.7 and 3.8, with Raw pixels and DesktopSize updates. Supported security types are None (no password reference) and classic VncAuth (password reference). TLS/VeNCrypt, Apple-specific authentication and vendor extensions are not implemented. A configured password never falls back to None.

Classic VNC passwords are limited to eight printable ASCII bytes; longer/non-ASCII values fail instead of being silently truncated. VNC authentication is not transport encryption. Keep the VNC listener on loopback and use SSH transport. If VNC is on a separate LAN host, the SSH-to-VNC network segment is not encrypted by this implementation. Existing SSH host-key verification limitations still apply; see [security](../SECURITY.md).

## Screenshot and input workflow

Each line below is a separate MCP call:

```text
list-connections {}
vnc-screenshot   {"profile":"local-dev"}
vnc-pointer      {"profile":"local-dev","events":[{"x":100,"y":100,"buttons":1}]}
vnc-key          {"profile":"local-dev","keys":["Control_L","l"]}
vnc-type         {"profile":"local-dev","text":"hello"}
vnc-key          {"profile":"local-dev","keys":["Return"]}
vnc-screenshot   {"profile":"local-dev"}
```

`vnc-screenshot` returns `content[0]` as an MCP image (`image/png`, base64 on the JSON wire), `content[1].text` as metadata JSON containing `width`, `height`, `name`. The client must support MCP image display. It is not a local screenshot path or text-only base64 result. Coordinates use the original image size with origin at the top left; a client may visually scale the preview, so use native pixel coordinates.

Mouse buttons are a bit mask: left 1, middle 2, right 4, wheel up 8, wheel down 16. Zero releases all buttons. A batch is one gesture and automatically ends with a release. Examples:

```text
vnc-pointer {"events":[{"x":100,"y":100,"buttons":0}]}
vnc-pointer {"events":[{"x":100,"y":100,"buttons":1},{"x":300,"y":250,"buttons":1}]}
vnc-pointer {"events":[{"x":300,"y":250,"buttons":16},{"x":300,"y":250,"buttons":0}]}
```

`vnc-key` accepts case-insensitive X11-style names: Control_L/Control_R (ctrl), Shift_L/Shift_R (shift), Alt_L/Alt_R (alt), Super_L/Super_R (super), Return (enter), Tab, Escape (esc), Backspace, Delete, Home, End, Insert, Left/Up/Right/Down, Page_Up/Page_Down, Space and F1–F12. A single printable Unicode character is also accepted. Keys press in array order and release in reverse order; keys cannot remain held between calls.

`vnc-type` uses key press/release pairs, not clipboard paste. CRLF sends one Return; CR/LF and Tab send the corresponding keys. Newlines can execute a terminal command or submit a form. Target keyboard layout, VNC implementation, application and input method can affect Unicode input. Verify visible results rather than assuming all Unicode characters render correctly.

Input success returns text JSON with `desktop` metadata and `message`; it only confirms that input/release events were written. Applications may still be processing, focus may differ, and another desktop client can act concurrently. Capture again and verify the intended outcome. Network failure/cancellation may occur after partial input; do not blindly retry.

## Bounds, lifecycle and troubleshooting

| Boundary | Value / behavior |
|---|---|
| Enablement | Absent section or `enabled = false` rejects desktop tools; SSH/SFTP remain usable |
| Operation timeout | Default 15 seconds, positive and at most 120 seconds; includes waiting, SSH/VNC setup and operation |
| Screenshot | Each side at most 8192 pixels, at most 8,388,608 pixels total; encoded PNG at most 16 MiB; full-frame coverage required |
| Pointer batch | 1–256 events; all coordinates checked before sending, within the current desktop; button mask 0–31 |
| Key chord | 1–16 keys |
| Text | 1–4096 Unicode characters; invalid UTF-8 and unsupported control characters rejected |
| Connections | New shared SSH/VNC connection per call, closed on success/error/cancellation |
| Concurrency | Desktop operations serialized within one MCP process; no lock across other desktop clients |

The screenshot bounds protect server memory; client/tunnel message limits may be lower, particularly because base64 adds overhead. Use a smaller desktop if a client rejects a large screenshot. There is no video stream, image scaling, clipboard synchronization, sound, browser UI or cross-call key hold. Screenshots omit the local viewer cursor overlay.

If VNC is disabled, verify the active configuration and restart MCP. If SSH forwarding fails, check SSH `AllowTcpForwarding`, the target address and the running VNC listener. For unsupported authentication, configure None on protected loopback or VncAuth over SSH. An out-of-bounds pointer error requires a fresh screenshot; resize can change the desktop between calls. Authentication and frame errors return MCP `isError: true` without a successful image payload.

Local acceptance uses real x11vnc, OpenSSH, xterm and an MCP stdio client. It verifies screenshot decoding, actual pointer coordinates and keyboard effects, authentication, cleanup and unpublished ports. This does not itself prove a real ChatGPT application has discovered or displayed the new tools; that requires upgrading the intended deployment and making actual tunnel calls.

Protocol references: [RFB specification](https://github.com/rfbproto/rfbproto/blob/master/rfbproto.rst), [x11vnc server](https://github.com/LibVNC/x11vnc), [official tunnel configuration](https://github.com/openai/tunnel-client/blob/v0.0.19/docs/configuration.md).
