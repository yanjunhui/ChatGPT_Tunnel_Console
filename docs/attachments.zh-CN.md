<!-- Created by Yanjunhui -->
# ChatGPT 附件直传

[English](attachments.md) | 简体中文 | [文档中心](README.zh-CN.md)

通过 `sftp-upload-file`，将 ChatGPT 消息中的附件直接上传到 SSH 主机。ChatGPT 会把文件提供给工具，无需在对话中粘贴文件内容或下载链接。

## 上传文件

1. 在 ChatGPT 消息中附加文件，并启用已连接的插件。
2. 指定 SSH 连接和目标，例如：“将这个文件上传到 local-dev 的 `/workspace/import/input.csv`。”
3. 如果目标父目录不存在，先创建目录。工具使用指定的完整目标路径，不根据附件原文件名决定保存位置。
4. 核对返回的目标路径和字节数。需要严格确认内容一致时，进一步比较目标校验和。

上传会覆盖已有目标。附件支持空文件和任意二进制内容。完成后的文件权限为 0600，其他账户需要访问时，另行调整权限。

升级后无法选择附件输入时，[刷新已安装插件的工具](troubleshooting.zh-CN.md#工具或参数仍然是旧版)，确认 `sftp-upload-file` 包含 `file` 参数。临时下载链接由客户端提供，不应复制到对话或日志。

## 传输限制

附件下载器允许最多 512 MiB，下载期限为五分钟，与 `output_bytes` 独立。ChatGPT／客户端的上传限制和请求期限可能更低。链接过期后需要重新提供附件。

文件先完整下载到私有临时存储，再开始远端写入。工具将内容上传到目标同目录的临时文件，最后原子重命名为目标。SFTP 服务必须支持 `posix-rename@openssh.com`，OpenSSH SFTP 支持此扩展。

下载或 SFTP 发布失败会保留原目标。请求正常结束时会删除本地暂存文件。强制终止进程可能留下操作系统临时文件；SSH 连接中断可能导致远端 `.tunnel-console-upload-*` 文件未能清理。删除遗留文件前应确认没有正在进行的传输。

## 已在 MCP 服务端的文件

源文件已位于运行 MCP 的机器或容器时，使用 `local_path`：

```json
{"path":"/workspace/import/input.csv","local_path":"/workspace/staging/input.csv"}
```

`file` 与 `local_path` 必须且只能提供一个。本地路径上传直接流式写入远端，失败可能留下部分文件。两种方式都会覆盖目标。默认 Docker 配置中，MCP 与自带 SSH 主机共享 `/workspace`，源与目标应选择不同路径。

## 客户端集成

工具遵循 [OpenAI 文件输入协议](https://developers.openai.com/plugins/reference#define-file-inputs)。`_meta["openai/fileParams"]` 标记 `file`，其中 `download_url`、`file_id` 必填，`mime_type`、`file_name` 可选。附件字节在模型文本上下文之外传输。

下载器验证 HTTPS 域名、重定向和公网 DNS 地址，拒绝内网／回环目标、嵌入凭据及非 443 端口。错误不包含签名 URL 或 HTTP 响应正文。允许域名和准确的返回／错误格式见[工具参考](tool-reference.zh-CN.md)，主机与网络出口策略仍然适用。
