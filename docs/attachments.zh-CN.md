<!-- Created by Yanjunhui -->
# ChatGPT 附件直传

[English](attachments.md) | 简体中文 | [README](../README.zh-CN.md)

0.3.3 为 `sftp-upload-file` 增加附件直传。升级后刷新 ChatGPT 应用的工具列表。描述符遵循 [OpenAI 官方文件参数协议](https://developers.openai.com/plugins/reference#define-file-inputs)：`_meta["openai/fileParams"]` 指定 `file`，其中 `download_url` 和 `file_id` 必填，`mime_type` 和 `file_name` 可选。

在 ChatGPT 附加文件，要求上传到明确目标，例如 `/workspace/import/input.csv`；目标父目录须已存在。客户端自动提供临时下载链接，不要把签名链接粘贴到聊天，也不要把附件字节复制进工具参数。成功返回字节数及目标路径；需要字节一致性时，核对目标哈希。

已在 MCP 服务端的文件仍采用原有参数：

```json
{"path":"/workspace/import/input.csv","local_path":"/workspace/staging/input.csv"}
```

`file` 与 `local_path` 必须且只能提供一个。两种来源都会覆盖目标。附件先完整下载到私有临时文件，再写入 SFTP 同目录私有临时文件，最后原子重命名发布。SSH 服务必须支持 `posix-rename@openssh.com`，完成后的附件权限为 0600；本地路径仍保留原来的非原子流式行为。

附件下载上限为 512 MiB、期限五分钟，与 `output_bytes` 独立，支持空文件和任意二进制。会验证 HTTPS 域名、重定向及公网 DNS 地址，拒绝内网／回环目标、嵌入凭据及非 443 端口。错误不包含临时 URL 或 HTTP 正文。允许域名和精确失败行为见[工具参考](tool-reference.zh-CN.md)。这些应用限制不能替代主机和网络出口策略。

链接过期后应重新提供附件，不要反复重试旧链接。下载失败不会开始远端写入；SFTP 发布失败保留原目标。SSH 断开可能导致私有 `.tunnel-console-upload-*` 临时文件无法清理。正常请求结束始终删除本地暂存；进程被直接终止时可能留下操作系统临时文件。

自动验证覆盖协议元数据／Schema、来源二选一、旧本地路径兼容、空／Unicode／二进制内容、字节上限、重定向、取消、签名链接脱敏及真实 SFTP 原子发布。这些检查与真实 ChatGPT 附件直传不同，部署记录必须明确后者是否实际完成。
