<!-- Created by Yanjunhui -->
# MCP 工具参考

[English](tool-reference.md) | 简体中文 | [README](../README.zh-CN.md)

适用于 0.3.3。工具注册位于 `internal/server/server.go`，参数、返回类型和实现位于 `internal/remote`。下文 JSON 名称均为实际公共契约；配置默认值见[配置参考](configuration.zh-CN.md)。

## 通用约定

- 除另有说明，`profile` 是可选字符串，用于选择 SSH 配置；省略或空值使用 `default_profile`，未知配置名报错。`list-sessions` 只查询一个配置，不查询全部配置。
- 必填字段即使允许空字符串也须提供，例如上传内容或只关闭 stdin 时的输入。远端 `path` 不得为空。会话名称不得为空白，在同一配置内唯一，已退出但仍保留的会话也占用名称。
- `path` 属于 SSH 主机，相对路径以配置的 `workdir` 为基准。`local_path` 属于 MCP 服务端，不属于 ChatGPT 客户端；相对本地路径以 MCP 进程工作目录为基准。建议流式传输使用绝对路径。不存在传输根目录限制。
- 除截图外，工具成功时在 MCP `content[0].text` 中返回 JSON。`sftp-download` 额外声明 `outputSchema` 并返回 `structuredContent`；两种截图工具返回图片和文本元数据（见下文）。其他工具需解析文本。
- 参数、连接和操作失败返回 MCP 工具错误（`isError: true`）。独立命令执行失败时仍在 JSON 中保留 stdout/stderr 和状态；`error` 非空时设为工具错误。非零退出即为失败，不能因存在输出而视为成功。
- 只读注解是给客户端的行为说明，不是强制策略或授权。任意命令、上传及写入服务端文件的下载均非只读。

## 连接和独立命令

### `list-connections`

输入 `{}`。返回按名称排序的数组，各项含 `name`、`host`、`port`、`user`、`workdir`、`default`、`sessions`、`status`、`vnc_enabled` 和 `desktop_enabled`。会话数包含已退出但保留的会话。状态为 `configured; connect on use`：该调用不进行 SSH 登录或连通性探测，也不暴露凭据。

### `run-command`

| 字段 | 必填 | 默认值／含义 |
|---|---|---|
| `profile` | 否 | 默认 SSH 配置 |
| `command` | 是 | 非空白 shell 命令或多行脚本 |
| `stdin` | 否 | 空字符串；发送完提供的数据后关闭 stdin |
| `tty` | 否 | False；true 分配 40×120、关闭回显的 `xterm-256color` PTY |
| `timeout_seconds` | 否 | 使用配置超时；显式 `0` 禁用期限；负数或溢出值报错 |

每次调用在配置工作目录中使用独立 SSH 连接和 shell。切换目录失败时，先退出再执行任何脚本行。shell 语法、sudo 和权限由目标机决定。返回示例：

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

执行失败时出现 `error`，没有可用退出码时 `exit_code` 为 null。超时或取消会关闭 SSH 并清空退出码，但脱离会话的子进程仍可能运行。PTY 通常把 stderr 合并到 stdout。每份输出独立有界；仅 stderr 非空不代表执行失败。`duration_ms` 记录执行时间，不包含建连；示例数值不代表耗时保证。

### `signal-process`

输入必填正整数 `pid`、可选 `profile` 和 `signal`（大写 `INT`、默认 `TERM` 或 `KILL`）。经独立命令路径执行 `kill -SIGNAL PID`，返回相同结果结构，受目标权限和配置工作目录影响。成功仅说明 kill 命令的退出状态，不证明所有相关进程已结束。

## 持久会话

### `open-session`

| 字段 | 必填 | 默认值／含义 |
|---|---|---|
| `profile` | 否 | 默认 SSH 配置 |
| `name` | 是 | 配置内唯一的非空白名称 |
| `type` | 否 | `interactive`，也可为 `background` |
| `command` | 仅后台模式 | 后台模式必填且非空白；交互模式须省略或为空 |
| `rows`、`cols` | 否 | 40 和 120；`0` 使用默认值；显式范围 1–65535 |

交互模式在 PTY 中启动 `${SHELL:-/bin/sh} -i`；后台模式无 PTY 执行命令。两者均在创建请求结束后保留专用连接和合并的 stdout/stderr，不使用短命令超时。重名、参数无效或 SSH 启动失败返回错误。服务重启丢失会话状态，不隐式恢复。

返回会话快照：`name`、`profile`、`type`、`status`（`running` 或 `exited`）、`started`（UTC 时间）、`exit_code`（整数或 null）以及可选 `error`。进程可能在创建后立即退出。

### `list-sessions`

输入可选 `profile`。返回上述会话快照，按名称排序；已退出会话保留到显式关闭，无会话时返回 `[]`。此操作不新建 SSH 连接。

### `write-session-input`

输入必填 `name` 和字符串 `data`，可选 `profile` 和布尔 `eof`（默认 false）。发送原始输入，提交 shell 命令时附加 `\n`。空 `data` 配合 `eof: true` 可只关闭 stdin。返回会话快照，只确认送达；未知／已退出会话和写入失败均报错，请求取消会断开该会话。

### `read-session-output`

输入必填 `name`、可选 `profile` 和整数 `cursor`（默认 0）。返回：

```json
{
  "session": {"name":"work","profile":"local-dev","type":"interactive","status":"running","started":"2026-11-01T09:00:00Z","exit_code":null},
  "output": {"text":"hello\n","next_cursor":6,"dropped_bytes":0}
}
```

游标是绝对字节位置，不是字符位置。读取不消费内容，也不等待执行完成；用 `output.next_cursor` 继续读取。请求游标早于保留缓存时，`dropped_bytes` 报告缺失字节数；负数或超前游标报错。PTY 输出含提示符、回显和控制码，应核对真实预期结果或进程退出，不把写入成功视为执行完成。

### `close-session`

输入必填 `name`、可选 `profile`。请求 TERM，最多等待 500 毫秒或请求取消，然后断开连接并移除保留记录。返回 `session`、`process_exit_observed` 和 `note`。只有捕获到可用退出码时 observed 才为 true；false 不代表断开失败。随后列表为空仅证明管理器已移除记录，不证明脱离会话的子进程终止。关闭未知或已移除会话会报错。

调用顺序示例，每行是独立工具调用：

```text
open-session         {"name":"work","type":"interactive"}
write-session-input  {"name":"work","data":"cd /workspace\nexport DEMO=hello\n"}
write-session-input  {"name":"work","data":"printf 'RESULT:%s:%s\\n' \"$DEMO\" \"$PWD\"\n"}
read-session-output  {"name":"work","cursor":0}
close-session        {"name":"work"}
list-sessions        {}
```

## SFTP 操作

每次操作使用独立 SSH/SFTP 连接。远端和本地父目录必须已存在；上下文取消会关闭连接，不自动重试已开始的上传。

### `sftp-list`

输入必填 `path`、可选 `profile`。目录项含 `name`、`size`（字节）、`mode`（可读权限字符串）、`modified`（时间）和 `directory`（布尔）。空目录返回 `[]`，路径和 SFTP 失败仍为错误。

### `sftp-download`

输入必填 `path`、可选 `profile` 和 `encoding`（默认 `utf8` 或 `base64`）。最多读取 `output_bytes` 个原始字节；更大文件报错并提示使用 `sftp-download-file`。非 UTF-8 内容需 base64 或流式工具。成功的结构化结果和文本结果包含相同三个必填字段：

```json
{"content":"Hello\n","encoding":"utf8","bytes":6}
```

`bytes` 统计原始字节，不是 base64 字符串长度。空文件返回 `{"content":"","encoding":"utf8","bytes":0}`。文件缺失、编码无效、UTF-8 无效或文件超限时，返回错误且不含成功结构化结果。

### `sftp-upload`

输入必填 `path` 和字符串 `content`，可选 `profile`、`encoding`（默认 `utf8` 或 `base64`）和整数 `mode`（默认 0）。内容允许为空。权限使用十进制 0–511，例如 420 表示八进制 0644；零表示不主动 chmod，不代表 chmod 0000。无效 base64／编码／权限在写入前报错。

创建或截断远端文件，六字节传输示例返回 `{"bytes":6,"path":"/workspace/hello.txt"}`。应用没有按 `output_bytes` 限制内联上传，但传输层仍可能限制大小。上传不具备原子性；复制失败报告已写字节并说明文件可能不完整，关闭或 chmod 失败也会返回错误。

### `sftp-download-file`

输入必填远端 `path`、MCP 服务端 `local_path` 和可选 `profile`。不受内联大小限制。先写同目录临时文件，传输及关闭成功后才重命名替换目标；失败保留已有文件并移除临时文件。返回 `bytes` 及本地目标 `path`，新文件不保留远端权限。

### `sftp-upload-file`

输入必填远端目标 `path`、可选 `profile`，并且在 MCP 服务端 `local_path` 与 ChatGPT `file` 中选择且仅选择一个。`file` 声明必填 `download_url`、`file_id` 和可选 `mime_type`、`file_name`；工具元数据声明 `openai/fileParams: ["file"]`。ChatGPT 自动提供附件对象，文件字节不进入模型上下文；附件元数据不用于选择目标路径。

本地路径保留原有流式行为：来源必须为普通文件，不受内联上限限制，失败时远端可能部分写入。附件仅允许 HTTPS 443、无用户信息或片段的 URL；允许域名及其子域为 `oaiusercontent.com`、`openai.com`、`chatgpt.com`、`oaistatic.com`、`amazonaws.com`、`blob.core.windows.net`、`googleapis.com`。最多五次重定向且逐次校验；不使用代理，连接固定到已验证公网 DNS 地址，并移除 Referer。错误不包含签名 URL 或响应正文，链接过期时需重新提供 ChatGPT 文件。

附件先暂存到 0600 私有临时文件，大小上限 512 MiB（含未知长度响应），下载期限五分钟，支持空文件。完整下载后，经 SFTP 写入同目录私有临时文件并以 POSIX rename 发布；不支持此扩展的服务器直接报错，不降低原子性。成功以 0600 文件替换目标，返回 `bytes` 和远端 `path`；失败保留原目标。本地暂存会删除；若 SSH 已断开，远端临时文件仅能尽力清理。详见[附件指南](attachments.zh-CN.md)。

## 原生桌面工具

`desktop-screenshot`、`desktop-pointer`、`desktop-key` 和 `desktop-type` 使用与下表 VNC 工具相同的参数、返回结构及输入边界；需要 `desktop.enabled = true`，不读取 VNC 配置或连接 VNC。每次操作通过 SSH 调用目标机的 `xwd` 或 `xdotool`；截图只接受 24 位 TrueColor 或恒等映射 DirectColor、24/32 位存储的 XWD 并转换为原尺寸 PNG。只支持 Linux X11，不支持原生 Wayland/macOS/Windows 桌面；外部目标须具备显示权限、工具及 C.UTF-8。失败会返回工具错误，部分输入可能已发生，不自动切换 VNC 或重试。两条路径在同一 MCP 进程内共用操作队列，以防手势交错并限制图片内存；配置开关、目标、超时及服务生命周期独立。详见[原生桌面指南](desktop.zh-CN.md)。

## VNC 桌面工具

四项工具均接受可选 `profile`，要求该配置的 `vnc.enabled = true`，经 SSH 转发使用独立共享 VNC 连接。错误返回 `isError: true`；输入可能部分发送，不自动重试。默认 15 秒超时涵盖排队与连接；请求取消和管理器关闭均会释放连接。完整操作和键名见 [VNC 指南](vnc.zh-CN.md)。

| 工具 | 必填参数 | 成功结果 |
|---|---|---|
| `vnc-screenshot` | 无 | `content[0]` 为 `image/png` 图片；`content[1].text` 包含 `width`、`height`、`name` 的 JSON |
| `vnc-pointer` | `events`：1–256 个含整数 `x`、`y`、`buttons` 的对象 | 文本 JSON：`desktop`（width/height/name）和 `message` |
| `vnc-key` | `keys`：1–16 个字符串，表示单键或组合键 | 同上 |
| `vnc-type` | `text`：1–4096 个 Unicode 字符 | 同上 |

截图仅在全帧覆盖后返回原尺寸 PNG：单边最多 8192、总像素最多 8,388,608、PNG 最多 16 MiB。鼠标坐标从左上角开始，发送前验证整批事件位于当前桌面；按钮位左1／中2／右4／上滚8／下滚16，0释放，结束时自动释放。组合键依次按下、逆序释放。文本通过按键发送，CRLF 视为一次 Return，CR/LF 和 Tab 受支持；无效 UTF-8 和其他控制字符拒绝，不使用剪贴板。输入结果只确认事件写入，须重新截图确认应用状态。仅截图标注只读；键鼠工具非只读。

从 0.3.1 起，两种截图工具省略 `outputSchema` 和仅含元数据的 `structuredContent`，因为真实 ChatGPT 在同时存在该结构化输出时只显示尺寸、遮蔽图片。原有图片及文本内容块保持不变；此前依赖结构化截图尺寸的客户端应改为解析 `content[1].text`。SFTP 下载仍保留类型化结构化结果。
