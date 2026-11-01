<!-- Created by Yanjunhui -->
# Commands, sessions and files

English | [简体中文](usage.zh-CN.md) | [Documentation](README.md)

After [connecting the plugin](installation.md#connect-chatgpt), describe the operation in your conversation. The assistant selects the corresponding MCP tools. The examples below also show exact tool arguments for clients and integrations; each line in a tool sequence is a separate call.

## Select a host and working directory

Call `list-connections` to see configured profiles, their working directories and whether desktop tools are enabled. Pass `profile` to choose a host; omitting it uses `default_profile`.

```text
list-connections {}
run-command {"profile":"local-dev","command":"id -un; pwd"}
```

The bundled profile uses `/workspace`, shared with the checkout's `workspace/` directory. External hosts use their configured `workdir`. Relative remote file paths resolve against that directory. Commands start there as well; changing directories in one `run-command` does not affect later calls.

## Run a command

Use `run-command` for independent commands or multiline scripts:

```json
{"profile":"local-dev","command":"cd /workspace/my-project\nmake test","timeout_seconds":120}
```

Replace the project path and command with ones appropriate to your project. Results include stdout, stderr and exit status. Nonzero exit status means the remote command failed even if the tool successfully delivered its result. Use `stdin` for program input and `tty: true` when a command requires a terminal; PTY output combines stdout and stderr.

Commands default to a 60-second deadline. Override it per call with `timeout_seconds`; `0` disables the command deadline. The calling client can still cancel a request. Use a background session for work that should continue after the opening call returns.

## Keep an interactive terminal

An interactive session keeps its shell, current directory and environment between calls:

```text
open-session         {"name":"work","type":"interactive","profile":"local-dev"}
write-session-input  {"name":"work","profile":"local-dev","data":"cd /workspace\nexport DEMO=hello\n"}
write-session-input  {"name":"work","profile":"local-dev","data":"printf '%s\\n' \"$DEMO\"\n"}
read-session-output  {"name":"work","profile":"local-dev","cursor":0}
```

Include a newline to submit a shell command. `write-session-input` confirms input delivery; read the output to see the result. Terminal output includes prompts, echo and control sequences. Subsequent reads should use the previous `output.next_cursor`, an absolute byte position.

To send Ctrl+C, write `\u0003` to the session. To close stdin, send empty `data` with `eof: true`. These operations depend on the remote program: Ctrl+C is terminal input, while EOF closes the input stream.

## Follow a background task

Use a unique session name for each retained task:

```text
open-session        {"name":"build","type":"background","profile":"local-dev","command":"cd /workspace/my-project && make build"}
read-session-output {"name":"build","profile":"local-dev","cursor":0}
list-sessions       {"profile":"local-dev"}
```

Background sessions retain output after the opening call returns and are not subject to the short-command deadline. Read periodically with `output.next_cursor` until the session reports `exited`, then inspect `exit_code` and any error. Exited sessions remain listed until closed.

```text
close-session {"name":"build","profile":"local-dev"}
close-session {"name":"work","profile":"local-dev"}
```

Closing requests TERM, disconnects and removes the retained session. Check `process_exit_observed` if you need confirmation of process exit. Detached child processes may survive; use `signal-process` with a known remote PID when necessary. Restarting MCP loses in-memory sessions and their output; sessions are not restored automatically.

## Read and write files

Use SFTP tools for directory listings and small text or binary files:

```text
sftp-list     {"profile":"local-dev","path":"."}
sftp-upload   {"profile":"local-dev","path":"hello.txt","content":"Hello\n"}
sftp-download {"profile":"local-dev","path":"hello.txt","encoding":"utf8"}
```

Use `encoding: "base64"` for binary content. Inline downloads are limited by `output_bytes`; the default is 1 MiB. For larger files, use the streaming tools. Parent directories must exist, and uploads overwrite the destination.

### Upload a ChatGPT attachment

Attach the file to your message and give an explicit destination, for example:

> Create `/workspace/import` if needed, then upload the attached CSV as `/workspace/import/input.csv`.

ChatGPT supplies the file object to `sftp-upload-file`. A successful response includes the destination path and transferred byte count. For datasets or archives where exact identity matters, also compare the destination's checksum. See [attachment uploads](attachments.md) for size limits and failure behavior.

### Transfer files on the MCP server

`local_path` refers to the machine or container running MCP. It is not a path on the computer where you are chatting.

```text
sftp-upload-file   {"profile":"local-dev","local_path":"/workspace/staging/input.csv","path":"/workspace/import/input.csv"}
sftp-download-file {"profile":"local-dev","path":"/workspace/result.tar","local_path":"/workspace/result-copy.tar"}
```

The source file and destination parent must already exist where relevant. The default Docker setup shares `/workspace` between MCP and the bundled development host, so use distinct source and destination paths. A download saves the result on the MCP server; it does not automatically create a downloadable chat attachment.

Downloads replace the local destination only after successful transfer. Inline and local-path uploads write directly and can leave a partial remote file if interrupted. ChatGPT attachment uploads stage the file and atomically replace the destination on a compatible SFTP server.

## Work with a desktop

Enable the desktop in the SSH profile, capture a screenshot, then choose coordinates from the original image. Input is sent to the focused application. Capture again after a gesture to inspect the outcome. Start with the [native desktop guide](desktop.md); use the [VNC guide](vnc.md) for an existing VNC server.

## Output and shared state

`output_bytes` controls each retained output buffer: 0 selects the 1 MiB default, and explicit values range from 1 byte to 64 MiB. When output exceeds the limit, the buffer keeps its tail and reports `dropped_bytes`. Save verbose logs to a remote file when you need the complete output.

All callers of a deployment share SSH profiles, session names and desktop state. Use separate deployments for separate trust domains. The service does not automatically retry started commands or uploads. If a call fails after it started, inspect the target state before retrying. Full parameter and result definitions are in the [tool reference](tool-reference.md).
