<!-- Created by Yanjunhui -->
# 命令、会话与文件

[English](usage.md) | 简体中文 | [文档中心](README.zh-CN.md)

[连接插件](installation.zh-CN.md#连接-chatgpt)后，在对话中描述操作，助手会选择对应的 MCP 工具。下文同时给出供客户端和集成使用的准确工具参数；调用序列中的每行表示一次独立调用。

## 选择主机和工作目录

调用 `list-connections` 查看已配置的连接、工作目录及桌面是否启用。传入 `profile` 选择主机，省略时使用 `default_profile`。

```text
list-connections {}
run-command {"profile":"local-dev","command":"id -un; pwd"}
```

自带连接使用 `/workspace`，与检出目录下的 `workspace/` 共享。外部主机使用其配置的 `workdir`。远端相对文件路径以该目录为基准，命令也从该目录开始执行；某次 `run-command` 中切换目录不会影响后续调用。

## 执行命令

使用 `run-command` 执行独立命令或多行脚本：

```json
{"profile":"local-dev","command":"cd /workspace/my-project\nmake test","timeout_seconds":120}
```

将示例路径和命令替换为实际项目所用内容。结果包含标准输出、标准错误和退出状态。即使工具成功返回结果，非零退出状态仍表示远端命令失败。通过 `stdin` 传入程序输入；需要终端时使用 `tty: true`，PTY 模式会合并标准输出和标准错误。

命令默认期限为 60 秒，可用 `timeout_seconds` 按次覆盖，`0` 关闭命令期限。调用客户端仍可能取消请求。需要在初始调用返回后继续执行的工作，使用后台会话。

## 保留交互终端

交互会话在多次调用之间保留 shell、当前目录和环境变量：

```text
open-session         {"name":"work","type":"interactive","profile":"local-dev"}
write-session-input  {"name":"work","profile":"local-dev","data":"cd /workspace\nexport DEMO=hello\n"}
write-session-input  {"name":"work","profile":"local-dev","data":"printf '%s\\n' \"$DEMO\"\n"}
read-session-output  {"name":"work","profile":"local-dev","cursor":0}
```

提交 shell 命令时须包含换行。`write-session-input` 确认输入送达，读取输出才能查看结果。终端输出包含提示符、回显和控制码。后续读取应使用上次返回的 `output.next_cursor`，它是绝对字节位置。

发送 Ctrl+C 时，向会话写入 `\u0003`。关闭标准输入时，传入空 `data` 和 `eof: true`。两者效果取决于远端程序：Ctrl+C 是终端输入，EOF 则关闭输入流。

## 跟踪后台任务

每个保留中的任务使用不同会话名：

```text
open-session        {"name":"build","type":"background","profile":"local-dev","command":"cd /workspace/my-project && make build"}
read-session-output {"name":"build","profile":"local-dev","cursor":0}
list-sessions       {"profile":"local-dev"}
```

后台会话在初始调用返回后继续保留输出，不受短命令期限限制。定期用 `output.next_cursor` 读取增量输出，直到会话报告 `exited`，再检查 `exit_code` 和错误。已退出会话会继续显示在列表中，直至关闭。

```text
close-session {"name":"build","profile":"local-dev"}
close-session {"name":"work","profile":"local-dev"}
```

关闭会话会请求 TERM、断开连接并移除保留的会话。需要确认进程已退出时，检查 `process_exit_observed`。脱离终端的子进程可能继续运行，必要时使用已知远端 PID 调用 `signal-process`。重启 MCP 会丢失内存中的会话及输出，不会自动恢复。

## 读写文件

通过 SFTP 工具列出目录、读写小型文本或二进制文件：

```text
sftp-list     {"profile":"local-dev","path":"."}
sftp-upload   {"profile":"local-dev","path":"hello.txt","content":"Hello\n"}
sftp-download {"profile":"local-dev","path":"hello.txt","encoding":"utf8"}
```

二进制内容使用 `encoding: "base64"`。内联下载受 `output_bytes` 限制，默认 1 MiB；较大文件使用流式工具。目标父目录须已存在，上传会覆盖目标文件。

### 上传 ChatGPT 附件

在消息中附加文件并给出明确目标，例如：

> 如果 `/workspace/import` 不存在，先创建目录，再将附加的 CSV 上传为 `/workspace/import/input.csv`。

ChatGPT 将文件对象提供给 `sftp-upload-file`。成功结果包含目标路径和传输字节数。对需要严格核对内容的数据集或归档文件，还可比对目标校验和。大小限制和失败行为见[附件直传](attachments.zh-CN.md)。

### 传输 MCP 服务端文件

`local_path` 指运行 MCP 的机器或容器中的路径，不是你正在聊天的电脑上的路径。

```text
sftp-upload-file   {"profile":"local-dev","local_path":"/workspace/staging/input.csv","path":"/workspace/import/input.csv"}
sftp-download-file {"profile":"local-dev","path":"/workspace/result.tar","local_path":"/workspace/result-copy.tar"}
```

对应的源文件和目标父目录须已存在。默认 Docker 配置在 MCP 和自带开发主机间共享 `/workspace`，因此源与目标应使用不同路径。下载会将结果保存到 MCP 服务端，不会自动生成可下载的聊天附件。

下载仅在传输成功后替换本地目标。内联上传和本地路径上传直接写入，传输中断可能留下部分远端文件。ChatGPT 附件上传先暂存，再在兼容的 SFTP 服务上原子替换目标。

## 操作桌面

在 SSH 连接配置中启用桌面，先截图，再根据原始图片选择坐标。输入会发送到当前获得焦点的应用。每次操作后重新截图，检查实际效果。自带桌面从[原生桌面指南](desktop.zh-CN.md)开始；已有 VNC 服务参阅 [VNC 指南](vnc.zh-CN.md)。

## 输出与共享状态

`output_bytes` 控制每个保留输出缓存的容量：0 使用默认 1 MiB，显式配置范围为 1 字节到 64 MiB。超出容量时保留尾部，并以 `dropped_bytes` 报告丢弃量。需要完整输出时，将详细日志写入远端文件。

同一部署的调用者共享 SSH 连接配置、会话名称和桌面状态。不同信任范围使用独立部署。服务不会自动重试已开始的命令或上传；调用开始后发生失败时，重试前先检查目标状态。完整参数和结果定义见[工具参考](tool-reference.zh-CN.md)。
