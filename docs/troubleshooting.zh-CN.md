<!-- Created by Yanjunhui -->
# 故障排查

[English](troubleshooting.md) | 简体中文 | [文档中心](README.zh-CN.md)

先定位失败阶段：服务启动、隧道发现、SSH 连接，还是某个工具。容器运行表示进程已启动，通过插件实际执行命令才能检查完整连接。

## 查看服务状态和日志

Docker 部署在部署目录执行以下命令。带上所用桌面／VNC 叠加文件，或使用部署中保存的 `COMPOSE_FILE`：

```sh
docker compose ps
docker compose logs --tail=100 mcp
docker compose logs --tail=100 dev
```

原生服务使用：

```sh
# macOS
sudo launchctl print system/com.yanjunhui.tunnel-ssh-mcp
sudo tail -n 100 /var/log/tunnel-ssh-mcp/stderr.log

# Linux
sudo systemctl status tunnel-ssh-mcp.service
sudo journalctl -u tunnel-ssh-mcp.service -n 100 --no-pager
```

记录版本、部署方式、失败工具和相关错误。分享日志前，移除凭据、附件签名链接、隧道 ID 和私有主机／账号信息。反馈方式见[贡献指南](../CONTRIBUTING.zh-CN.md)。

## 找不到隧道或插件

- 检查隧道是否属于当前 ChatGPT 账号或工作区关联的 Platform 组织。
- 确认服务配置与插件选择的是同一个隧道 ID，运行密钥具有 Read + Use 权限。
- 检查服务主机的出站 HTTPS 网络，以及 MCP 日志中的 tunnel-client 错误。
- 每个隧道 ID 仅运行一个 stdio tunnel-client，替换实例前先停止旧实例。

手工安装时，可用 `doctor` 运行官方隧道诊断：

```sh
./bin/tunnel-console-mcp doctor --config config.toml
```

使用实际生效的配置路径。诊断会连接隧道服务，成功结果不代表 SSH 凭据有效。隧道出现后，启用插件并在一个连接配置上执行命令。

## 工具或参数仍然是旧版

1. 确认目标部署已经重新构建并重启。检出源码与运行中的程序可能不是同一版本。
2. 打开实际安装的插件管理页并刷新工具。尤其在多个插件同名时，核对插件身份和当前登录账号。
3. 确认工具目录中出现预期工具或参数，例如 `sftp-upload-file` 的 `file` 输入，再在启用该插件的新对话中调用。

刷新浏览器或新建对话不会刷新另一个插件保存的工具目录。在一个账号刷新插件，也不会更新另一账号中的独立安装。操作说明见 [OpenAI 工具元数据刷新指南](https://developers.openai.com/plugins/deploy/connect-chatgpt#refresh-metadata)。

查看 Docker 中实际运行程序的版本：

```sh
docker compose exec mcp tunnel-console-mcp version
```

## 能列出连接，但命令失败

`list-connections` 只读取配置，不连接目标。检查所选连接的主机、端口、用户、认证方式和 `workdir`。

- 目标须能从 MCP 服务所在环境访问。容器中的 `127.0.0.1` 指容器自身，不是 Docker 宿主机或另一个服务。
- SSH 账户须能使用所配置的私钥或密码来源，并有权进入工作目录。
- 原生开机服务不能依赖登录 shell 的 SSH agent，请导入私钥文件或密码。
- 命令使用 POSIX shell。交互终端可执行而工具失败时，检查 PATH、shell 启动文件、环境变量，以及命令是否需要 PTY。

自带环境可检查 `dev` 容器状态和日志。外部 SSH 应从服务所在网络环境测试连接。当前服务不验证 SSH 主机公钥，详见[安全说明](../SECURITY.zh-CN.md)。

## 修改配置没有生效

配置在启动时加载。编辑或导入运行服务实际使用的配置，再重启 MCP。Docker 通常使用 `data/current/config.toml`，原生服务使用 `/usr/local/lib/tunnel-ssh-mcp/current/config.toml`。

初始化器默认复用已有配置。Docker 初始化可使用 `--reconfigure` 重新打开向导。两种安装方式均可用 `--config` 导入完整替代配置。向导会替换连接选择，需要保留多个连接时使用完整 TOML。初始化和运行须使用同一个配置挂载目录，详见[安装与升级](installation.zh-CN.md)。

`check` 验证 TOML 结构，不测试凭据或连通性。重启会清空内存会话，应先结束或记录进行中的工作。

## 命令超时或输出不完整

独立命令默认 60 秒期限。可增加 `timeout_seconds`，或使用后台会话并在后续调用中读取输出。关闭命令期限不会取消客户端请求限制。

`dropped_bytes` 非零表示输出已超过保留缓存。可更频繁读取、在上限内增大 `output_bytes`，或将完整日志重定向到远端文件。增量读取传入上次的 `output.next_cursor`；负游标和超过已产生输出的位置会失败。

提交输入不等于命令完成，应检查输出和退出状态。调用开始后发生失败时，重复执行前先检查远端进程，避免重复工作。

## 文件传输失败

| 现象 | 检查方法 |
|---|---|
| 目标不存在或没有权限 | 在 SSH 主机创建父目录，并检查 SSH 账户的写权限 |
| 找不到本地源文件 | `local_path` 须存在于 MCP 服务端；容器部署时须位于容器内 |
| 内联下载过大或内容不是 UTF-8 | 大文件使用 `sftp-download-file`，小型二进制使用 `encoding: "base64"` |
| 附件链接过期 | 重新附加文件，让 ChatGPT 提供新的下载链接 |
| 附件来源被拒绝 | 只提供 `file` 或 `local_path` 中的一项，由 ChatGPT 提供文件对象 |
| 不支持附件发布 | SFTP 服务须支持 `posix-rename@openssh.com`，OpenSSH SFTP 支持此扩展 |

附件下载器上限为 512 MiB，下载期限为五分钟；ChatGPT／客户端的上传限制和请求期限可能更低。网络访问限制也可能阻止有效附件下载。不要把临时链接或密钥粘贴到问题报告。

附件发布失败保留原目标。SSH 连接中断可能留下私有 `.tunnel-console-upload-*` 临时文件，删除前应检查并确认没有正在进行的上传。内联上传和本地路径上传可能留下部分目标文件，详见[附件直传](attachments.zh-CN.md)。

## 桌面截图或输入失败

- **工具未启用：**在所选连接配置启用 `desktop` 或 `vnc`，重启 MCP。`list-connections` 显示的是配置标志，不是实时桌面探测结果。
- **原生显示不可用：**检查 Linux X11 显示、`xwd`、`xdotool`、`C.UTF-8`，以及 SSH 账户的显示认证，不要全局关闭 X 访问控制。
- **VNC 连接失败：**从 SSH 主机检查目标地址、VNC 监听、SSH 转发和支持的认证方式。自带桌面须同时使用桌面及 VNC 叠加配置。
- **鼠标坐标越界：**重新截图并使用原始图片坐标，桌面尺寸可能已变化。
- **文本输入位置错误：**检查焦点，重新截图后再决定是否重试。原生非 ASCII 输入宜分成短片段，默认期限下每段约 100 字符；也可将桌面超时调高，最大 120 秒。
- **图片过大：**降低目标桌面分辨率，客户端消息限制可能小于服务端图片上限。

原生后端支持 Linux X11。VNC 认证、键名和协议限制见[桌面](desktop.zh-CN.md)、[VNC](vnc.zh-CN.md)及[工具参考](tool-reference.zh-CN.md)。
