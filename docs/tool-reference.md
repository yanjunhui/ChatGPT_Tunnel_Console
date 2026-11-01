<!-- Created by Yanjunhui -->
# MCP tool reference

English | [简体中文](tool-reference.zh-CN.md) | [Documentation](README.md)

Input parameters, successful results and errors for all 21 MCP tools in version 0.3.3. For common workflows, see the [usage guide](usage.md); defaults and profile settings are in the [configuration reference](configuration.md).

## Shared conventions

- Unless stated otherwise, `profile` is an optional string selecting an SSH profile; omitted/empty selects `default_profile`. An unknown name is an error. `list-sessions` selects one profile, not all profiles.
- Required fields must be supplied even when an empty string is meaningful, such as upload content or input sent only to close stdin. Remote `path` must be nonempty. Session names must be nonblank and are unique within a profile, including retained exited sessions.
- `path` belongs to the SSH host. Relative remote paths use its configured `workdir`. `local_path` belongs to the MCP server, not the ChatGPT client, and relative local paths use the MCP process working directory. Use absolute paths for predictable streaming behavior. No transfer-root restriction is applied.
- Except for screenshots, tools return JSON in MCP `content[0].text` on success. `sftp-download` additionally declares an `outputSchema` and supplies `structuredContent`; both screenshot tools return an image and text metadata (below). Decode text for other tools.
- Validation, connection and operation failures return MCP tool errors (`isError: true`). Independent command execution failures retain stdout/stderr and status fields in their JSON result and set `isError: true` when `error` is nonempty. Nonzero exit status is a failure, even if output was produced.
- Read-only annotations describe intended behavior for clients; they are not enforcement or authorization. Arbitrary commands, uploads and server-side file downloads are not read-only.

## Connections and independent commands

### `list-connections`

Input: `{}`. Returns an array sorted by name. Each entry contains `name`, `host`, `port`, `user`, `workdir`, `default`, `sessions`, `status`, and `vnc_enabled` and `desktop_enabled`. Session counts include retained exited sessions. Status is `configured; connect on use`: this call neither authenticates to SSH nor probes connectivity, and does not expose credentials.

### `run-command`

| Field | Required | Default / meaning |
|---|---|---|
| `profile` | No | Default SSH profile |
| `command` | Yes | Nonblank shell command or multiline script |
| `stdin` | No | Empty string; stdin closes after the supplied data |
| `tty` | No | False; true allocates a 40×120 `xterm-256color` PTY with echo disabled |
| `timeout_seconds` | No | Profile command timeout; explicit `0` disables it; negative/overflow values fail |

Each call opens its own SSH connection and shell in the configured workdir. A failed directory change exits before any script line executes. Shell syntax, sudo and permissions follow the target host. The result is:

```json
{
  "stdout": {"text":"developer\n","next_cursor":10,"dropped_bytes":0},
  "stderr": {"text":"","next_cursor":0,"dropped_bytes":0},
  "exit_code":0,
  "timed_out":false,
  "canceled":false,
  "duration_ms":6
}
```

`error` appears on execution failure; `exit_code` is null when no usable exit status is available. Timeout/cancellation closes SSH and clears the exit code, but detached remote children may survive. A PTY normally combines stderr into stdout. Output is bounded per stream; a nonempty stderr alone is not an execution error. `duration_ms` measures command execution, excluding connection setup; the example is illustrative, not a timing guarantee.

### `signal-process`

Input: required positive integer `pid`, optional `profile`, optional `signal` (`INT`, `TERM` default, or `KILL`, uppercase). Executes `kill -SIGNAL PID` through the same independent-command path and returns the same result shape. It uses target permissions and the configured workdir. Success confirms the kill command's exit status, not completion of every affected process.

## Persistent sessions

### `open-session`

| Field | Required | Default / meaning |
|---|---|---|
| `profile` | No | Default SSH profile |
| `name` | Yes | Nonblank name unique within that profile |
| `type` | No | `interactive`; alternative `background` |
| `command` | Background only | Required/nonblank for background; must be absent or empty for interactive |
| `rows`, `cols` | No | 40 and 120; `0` selects the default; explicit values 1–65535 |

Interactive sessions start `${SHELL:-/bin/sh} -i` in a PTY. Background sessions run the command without a PTY. Both retain a dedicated connection and combined stdout/stderr after the opening request completes. Short-command timeouts do not apply. Duplicate names, invalid arguments or failed SSH startup return errors. Service restart loses session state; there is no implicit restoration.

Returns a session snapshot with `name`, `profile`, `type`, `status` (`running` or `exited`), `started` (UTC timestamp), `exit_code` (integer or null), and optional `error`. A process may exit immediately after creation.

### `list-sessions`

Input: optional `profile`. Returns the same session snapshots, sorted by name. Includes exited sessions until explicitly closed, and returns `[]` when none remain. It does not create a new SSH connection.

### `write-session-input`

Input: required `name` and string `data`, optional `profile` and boolean `eof` (default false). Sends raw input; include `\n` to submit a shell command. An empty `data` with `eof: true` closes stdin without sending text. Returns a session snapshot, acknowledging delivery only. Unknown/exited sessions and write errors fail; cancellation disconnects this session.

### `read-session-output`

Input: required `name`, optional `profile`, optional integer `cursor` (default 0). Returns:

```json
{
  "session": {"name":"work","profile":"local-dev","type":"interactive","status":"running","started":"2026-11-01T09:00:00Z","exit_code":null},
  "output": {"text":"hello\n","next_cursor":6,"dropped_bytes":0}
}
```

Cursors are absolute byte offsets, not character positions. Reads do not consume output or wait for completion. Reuse `output.next_cursor` to request new output. A cursor older than the retained buffer reports missing bytes in `dropped_bytes`; a negative or future cursor is an error. Prompts, echo and terminal control sequences are part of PTY output. Confirm a real expected result or process exit instead of treating a successful write as completion.

### `close-session`

Input: required `name`, optional `profile`. Requests TERM, waits up to 500 ms (or request cancellation), disconnects and removes the retained entry. Returns `session`, `process_exit_observed` and `note`. The observed flag is true only when a usable exit code was captured; false does not mean disconnect failed. A subsequent empty list proves removal from the manager, not termination of detached children. Closing an unknown/already-removed session returns an error.

Example sequence (each line is a separate tool call):

```text
open-session         {"name":"work","type":"interactive"}
write-session-input  {"name":"work","data":"cd /workspace\nexport DEMO=hello\n"}
write-session-input  {"name":"work","data":"printf 'RESULT:%s:%s\\n' \"$DEMO\" \"$PWD\"\n"}
read-session-output  {"name":"work","cursor":0}
close-session        {"name":"work"}
list-sessions        {}
```

## SFTP operations

Each operation uses a fresh SSH/SFTP connection. Remote and local parent directories must already exist. Context cancellation closes the connection. Started uploads are not retried automatically.

### `sftp-list`

Input: required `path`, optional `profile`. Returns entries with `name`, `size` (bytes), `mode` (readable permission string), `modified` (timestamp), and `directory` (boolean). An empty directory returns `[]`; path and SFTP errors remain errors.

### `sftp-download`

Input: required `path`, optional `profile`, optional `encoding` (`utf8` default or `base64`). Reads up to `output_bytes` raw bytes; larger files fail with guidance to use `sftp-download-file`. Invalid UTF-8 requires base64 or streaming. Successful structured and text results contain the same three required fields:

```json
{"content":"Hello\n","encoding":"utf8","bytes":6}
```

`bytes` counts original bytes, not the base64 string length. An empty file is `{"content":"","encoding":"utf8","bytes":0}`. Missing files, invalid encoding, invalid UTF-8 and oversized files return errors without successful structured content.

### `sftp-upload`

Input: required `path` and string `content`; optional `profile`, `encoding` (`utf8` default or `base64`) and integer `mode` (default 0). Content may be empty. `mode` accepts decimal 0–511; for example, 420 means octal 0644. Zero means no explicit chmod, not chmod 0000. Invalid base64/encoding/mode fails before writing.

Creates or truncates the remote file and returns `{"bytes":6,"path":"/workspace/hello.txt"}` for a six-byte transfer. There is no application inline-upload limit tied to `output_bytes`; transport limits may still apply. Upload is not atomic. Copy failures report bytes written and warn that the remote file may be partial; close/chmod failures also return errors.

### `sftp-download-file`

Input: required remote `path` and MCP-server `local_path`, optional `profile`. Streams without the inline download limit. A temporary sibling file is renamed over the local destination only after successful transfer and close. Failures preserve the existing destination and remove the temporary file. Returns `bytes` and the local destination `path`; does not preserve remote permissions on the new local file.

### `sftp-upload-file`

Input: required remote destination `path`, optional `profile`, and exactly one source: `local_path` on the MCP server or ChatGPT `file`. `file` declares required `download_url` and `file_id`, plus optional `mime_type` and `file_name`; tool metadata advertises `openai/fileParams: ["file"]`. ChatGPT supplies the attachment object without placing file bytes in model context. File metadata never determines the destination path.

Local-path uploads preserve the existing streaming behavior: source must be regular, no inline limit, remote writes may be partial on failure. Attachment downloads allow HTTPS on port 443 only, with no userinfo/fragment; approved host families are `oaiusercontent.com`, `openai.com`, `chatgpt.com`, `oaistatic.com`, `amazonaws.com`, `blob.core.windows.net`, and `googleapis.com`. Redirects are validated (maximum five), proxies are bypassed, public DNS results are pinned for connection, and Referer is removed. Signed URLs and response bodies are omitted from errors. Expired links require a fresh ChatGPT file.

Attachments stage to a private mode-0600 temporary file with a 512 MiB limit (including unknown-length responses) and a five-minute download deadline. Empty files are supported. After a complete download, SFTP writes a private sibling file and publishes with POSIX rename; unsupported servers fail rather than weakening atomicity. Success replaces the destination with mode 0600, returning `bytes` and remote `path`. Failure preserves the original destination. Local staging is removed; remote temporary-file cleanup is best effort if SSH has already disconnected. See the [attachment guide](attachments.md).

## Native desktop tools

`desktop-screenshot`, `desktop-pointer`, `desktop-key` and `desktop-type` use the same arguments, result shapes and input bounds as their VNC counterparts below. They require `desktop.enabled = true` and neither read VNC settings nor connect to VNC. Each operation runs target-side `xwd` or `xdotool` over SSH. Capture accepts only 24-bit TrueColor or identity-mapped DirectColor XWD with 24/32-bit storage and returns an unscaled PNG. Native support is Linux X11, not native Wayland/macOS/Windows desktops; external targets need display access, the utilities and C.UTF-8. Errors remain tool errors, possibly after partial input; no automatic VNC fallback or retries. Both paths share an operation queue within one MCP process to prevent interleaved gestures and bound image memory, while configuration, target, timeout and service lifecycle remain independent. See the [native desktop guide](desktop.md).

## VNC desktop tools

All four tools accept optional `profile`, require `vnc.enabled = true`, and use a separate shared VNC connection through SSH forwarding. Failures return `isError: true`; input may be partially delivered and is never retried automatically. The default 15-second timeout includes queueing and connection; cancellation and manager shutdown release connections. See the [VNC guide](vnc.md) for workflows and key names.

| Tool | Required arguments | Successful result |
|---|---|---|
| `vnc-screenshot` | None | `content[0]` is an `image/png` image; `content[1].text` contains JSON with `width`, `height`, `name` |
| `vnc-pointer` | `events`: 1–256 objects with integer `x`, `y`, `buttons` | Text JSON: `desktop` (width/height/name) and `message` |
| `vnc-key` | `keys`: 1–16 strings representing a key or chord | Same as above |
| `vnc-type` | `text`: 1–4096 Unicode characters | Same as above |

Screenshots return native PNG only after full-frame coverage: at most 8192 per side, 8,388,608 total pixels and 16 MiB encoded PNG. Pointer coordinates start at the top left; the entire batch is validated against the current desktop before sending. Button bits are left 1/middle 2/right 4/up-scroll 8/down-scroll 16, zero releases, and a final release is automatic. Chords press in order and release in reverse. Text uses key events, treats CRLF as one Return and supports CR/LF and Tab; invalid UTF-8 and other controls fail. Clipboard is not used. Input results acknowledge writes only; capture again to confirm application state. Only screenshots are annotated read-only; input tools are not.

Screenshot clients should display the image in `content[0]` and parse dimensions from `content[1].text`. Screenshot tools do not return `structuredContent`. SFTP downloads return both text JSON and typed structured results.
