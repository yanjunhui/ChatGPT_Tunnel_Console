<!-- Created by Yanjunhui -->
# ChatGPT Tunnel Console

[English](README.md) | 简体中文

通过 OpenAI Tunnel，在 ChatGPT 或 Codex 中执行命令、传输文件并操作 Linux 桌面。

ChatGPT Tunnel Console 将助手连接到 Linux 开发工作区或已有 SSH 主机。它是一个用 Go 编写的 MCP 服务，提供持久终端、SFTP 文件传输和桌面控制。自带 Docker 环境包含构建和运行软件所需的工具，也可通过 SSH 连接配置接入自己的主机。

## 主要功能

| 功能 | 常见用途 |
|---|---|
| 远程命令 | 查看文件、执行脚本、构建项目并读取退出状态 |
| 持久终端 | 在多次调用间保留 shell 状态、与程序交互、跟踪后台任务 |
| 文件传输 | 浏览目录、读写文件、上传 ChatGPT 附件及流式传输较大文件 |
| Linux 桌面 | 在对话中截取 X11 桌面并操作鼠标与键盘 |
| 多主机连接 | 每次操作选择具名 SSH 连接配置 |

原生桌面通过 SSH 和 X11 操作。可选 VNC 工具既能连接自带桌面，也能接入兼容的现有 VNC 服务。设置方式与支持环境见[桌面指南](docs/desktop.zh-CN.md)。

## 快速开始

需要安装 Docker 和 Compose v2，并准备 OpenAI 隧道 ID，以及具有该隧道 **Read + Use** 权限的运行时 API Key。在 [OpenAI Platform 设置](https://platform.openai.com/settings/organization/tunnels)中创建隧道；账号条件与设置方法见[安装指南](docs/installation.zh-CN.md)。

```sh
git clone https://github.com/yanjunhui/ChatGPT_Tunnel_Console.git
cd ChatGPT_Tunnel_Console
mkdir -p data init-input workspace
docker compose -f compose.init.yaml run --rm --build init
```

输入隧道 ID 和运行密钥，选择自带命令行开发环境。初始化器将配置和凭据保存到 `data/`，随后启动服务：

```sh
docker compose up -d --build
```

在 ChatGPT 中创建自定义 MCP 插件，连接类型选择 **Tunnel**，使用相同的隧道 ID，认证方式选择 **No authentication**。安装并启用插件后，可以要求：

> 列出我的 SSH 连接，然后在 local-dev 上查看当前用户和工作目录。

自带环境使用 `developer` 账户，工作目录为 `/workspace`，对应检出目录中的 `workspace/`。环境包含 Go、Python、Node.js、Git 和常用构建工具。完整连接步骤见[连接 ChatGPT](docs/installation.zh-CN.md#连接-chatgpt)。

### 启用图形桌面

初始化时选择 **3：自带图形桌面**，可选 VNC 保持 **n**，使用原生桌面控制。随后带上桌面叠加配置启动：

```sh
docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

已有安装须先按[桌面指南](docs/desktop.zh-CN.md)更新配置，再启用叠加文件。这会启动一个 1280×800 的 Linux 桌面，包含 Openbox 和 xterm。先要求截图，再通过鼠标和键盘操作。需要 VNC 时，参阅 [VNC 设置](docs/vnc.zh-CN.md)。

### 连接自己的 SSH 主机

Docker 初始化器也支持外部 SSH 主机。还可以直接在 macOS 或 Linux 上安装原生服务：

```sh
make build
./bin/tunnel-console-mcp init
```

原生构建需要 Go 1.26.10，或允许自动下载所需工具链。初始化器配置 SSH 凭据并注册系统服务。支持平台、服务路径和手工运行方式见[原生安装](docs/installation.zh-CN.md#原生运行连接现有-ssh-主机)。

## 日常使用

启用插件后，可以让助手执行以下操作：

- “在 `/workspace/my-project` 运行项目测试，列出失败项。”
- “在后台会话中开始构建，并检查输出。”
- “把附加的 CSV 上传到 `/workspace/import/input.csv`。”
- “截取桌面，选中终端，然后输入这条命令。”

操作前选择已有项目，或先创建目标目录。ChatGPT 会将附件直接提供给上传工具，无需粘贴文件内容或临时下载链接。参数和示例见[命令与会话](docs/usage.zh-CN.md)、[附件直传](docs/attachments.zh-CN.md)和[工具参考](docs/tool-reference.zh-CN.md)。

## 连接方式

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

服务主动建立出站隧道连接，不开放 MCP HTTP 端口。默认 Docker 配置也不发布 SSH 或 MCP 端口。SSH 连接配置决定目标主机、账户和工作目录。

调用者拥有 SSH 账户的权限，包括已配置的 sudo 权限。同一部署的调用者共享连接配置、会话和桌面。当前**不校验 SSH 主机公钥**。请阅读[安全说明](SECURITY.zh-CN.md)，为不同信任范围使用独立部署。

## 文档

| 指南 | 内容 |
|---|---|
| [安装与升级](docs/installation.zh-CN.md) | Docker、原生服务、ChatGPT 连接、备份与升级 |
| [命令、会话与文件](docs/usage.zh-CN.md) | 常用操作与工作目录行为 |
| [ChatGPT 附件](docs/attachments.zh-CN.md) | 上传方法、目标路径和传输限制 |
| [原生桌面](docs/desktop.zh-CN.md) · [VNC](docs/vnc.zh-CN.md) | 桌面配置、截图与输入 |
| [配置参考](docs/configuration.zh-CN.md) · [工具参考](docs/tool-reference.zh-CN.md) | 命令行选项、TOML 配置和全部 21 个 MCP 工具 |
| [故障排查](docs/troubleshooting.zh-CN.md) | 连接、工具发现、文件传输和桌面问题 |

全部指南见[文档中心](docs/README.zh-CN.md)。开发环境与检查方法见[贡献指南](CONTRIBUTING.zh-CN.md)，版本变化见[更新记录](CHANGELOG.zh-CN.md)。

## 许可证

[MIT](LICENSE)。本项目为独立项目，使用[官方 OpenAI tunnel-client](https://github.com/openai/tunnel-client) 和 [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)。依赖许可证与致谢见[第三方声明](THIRD_PARTY_NOTICES.zh-CN.md)。
