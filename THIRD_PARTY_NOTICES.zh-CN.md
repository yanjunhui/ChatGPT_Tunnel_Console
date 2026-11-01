<!-- Created by Yanjunhui -->
# 第三方声明

[English](THIRD_PARTY_NOTICES.md) | 简体中文

项目原创代码采用 [MIT](LICENSE) 许可证。依赖及外部工具保留各自上游许可证；本仓库许可证不会重新许可这些组件。本源码仓库不内置第三方源码，也不发布已编译的二进制或镜像。

## 直接 Go 依赖

版本锁定于 `go.mod`，校验和锁定于 `go.sum`。

| 组件 | 版本 | 上游许可证 |
|---|---|---|
| BurntSushi/toml | v1.6.0 | [MIT](https://github.com/BurntSushi/toml/blob/v1.6.0/COPYING) |
| creack/pty | v1.1.24 | [MIT](https://github.com/creack/pty/blob/v1.1.24/LICENSE) |
| modelcontextprotocol/go-sdk | v1.8.0 | [Apache-2.0 及历史 MIT 贡献](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/LICENSE)；上游文档采用 CC-BY-4.0 |
| pkg/sftp | v1.13.12 | [BSD-2-Clause](https://github.com/pkg/sftp/blob/v1.13.12/LICENSE) |
| golang.org/x/crypto | v0.58.0 | [BSD-3-Clause](https://github.com/golang/crypto/blob/v0.58.0/LICENSE) |
| golang.org/x/sys | v0.49.0 | [BSD-3-Clause](https://github.com/golang/sys/blob/v0.49.0/LICENSE) |
| golang.org/x/term | v0.47.0 | [BSD-3-Clause](https://github.com/golang/term/blob/v0.47.0/LICENSE) |

MCP SDK 的许可证声明说明其正在迁移许可：未取得重新许可同意的贡献仍适用 MIT。不得将整个 SDK 描述为仅适用 MIT 或仅适用 Apache-2.0；具体以对应版本的 LICENSE 为准。

间接 Go 依赖列于 `go.mod`，可通过 `go list -m all` 枚举；许可证文件保留在下载的模块源码中。分发二进制时，应收集实际链接依赖的完整声明及许可文本，而不只使用这份直接依赖清单。

## 外部运行及开发工具

- [OpenAI tunnel-client v0.0.19](https://github.com/openai/tunnel-client/blob/v0.0.19/LICENSE) 采用 Apache-2.0，从官方发布下载；其发布包内的 [Cloudflare cloudflared](https://github.com/cloudflare/cloudflared/blob/master/LICENSE) 等组件保留各自许可条款。
- Dockerfile 使用官方 Go、Node.js 和 Debian 镜像并安装 Debian 软件包。相关许可及源码提供要求继续适用，包括复制到开发镜像中的组件。
- 可选桌面镜像安装 Xvfb、Openbox、xterm、xwd、xdotool 和字体，独立可选 VNC 镜像安装 x11vnc 和 X11 工具，均保留各自软件包许可；本项目 MIT 不重新许可这些组件。准确的许可证与版权清单位于构建镜像的 `/usr/share/doc/<package>/copyright`，重分发前须检查相关源码及声明义务。Go RFB 客户端由本项目独立实现，未内置第三方 VNC 客户端库。
- govulncheck 与 Gitleaks 是单独下载的开发检查工具，版本固定于 Makefile，不属于 MCP 运行时。

重新分发构建后的镜像或可执行文件前，应保留上游要求的版权、许可证及 NOTICE 文件，并满足适用的源码分发义务。本文是本次源码发布的清单，不能替代上游许可文本或完整二进制分发声明包。

OpenAI、ChatGPT、Codex 等产品名称用于标识兼容服务或依赖。本项目为独立项目，不是 OpenAI 官方产品，也不声称存在关联或背书。
