<!-- Created by Yanjunhui -->
# 更新记录

[English](CHANGELOG.md) | 简体中文

## 0.2.0 — 2026-11-01

- 项目／仓库更名为 ChatGPT_Tunnel_Console，Go 模块改为 `github.com/yanjunhui/ChatGPT_Tunnel_Console`，主命令改为 `tunnel-console-mcp`。
- 新增按配置启用的 VNC 截图、鼠标手势、组合键和 Unicode 输入四项 MCP 工具，经 SSH 转发访问；现有 SSH/SFTP 契约继续可用。
- 无新增 Go 依赖，实现有界 RFB 3.3/3.7/3.8 Raw 截图、桌面尺寸变化、None/VncAuth 鉴权及取消清理。
- 增加可选 TigerVNC/Openbox Docker 桌面、初始化桌面选项、真实桌面验收及完整双语配置／工具文档。
- 保留原部署路径、Compose／卷标识、环境配置和旧命令别名；已有配置关闭 VNC，本次源码发布不升级正在运行的 0.1.1 服务。

## 0.1.2 — 2026-11-01

- 修复 run、serve、check、doctor、install-tunnel 的 `--help` 错误退出；增加覆盖这些命令及 init 的回归测试。
- 根据漏洞审查，将最低 Go 工具链及 Docker 构建基线提升至 1.26.10。
- 将 Go 模块路径统一为 `github.com/yanjunhui/ChatGPT_Tunnel_SSH`。
- 支持通过 Compose 镜像标签区分验收构建和正式部署。
- Docker 构建前先创建项目临时目录，支持首次检出的干净环境。
- 使 Docker 测试夹具兼容不同 Linux 宿主机用户编号，说明挂载目录归属要求，并保留私有配置权限。
- 完善公开源码发布：MIT 许可、中英贡献／安全／第三方文档、CI、依赖更新、文档检查和敏感信息扫描；公开部署记录已去除私人标识。

MCP 工具契约与 0.1.1 兼容。本次源码发布不会自动升级已有部署；记录中的正式部署仍为 0.1.1。

## 0.1.1 — 2026-11-01

- 为内联 SFTP 下载增加 `outputSchema` 和类型化 `structuredContent`，保留原有文本 JSON 响应，明确返回空文件字段。
- 通过真实隧道、SSH 和 SFTP 调用验收正式容器升级，详见[验证记录](docs/verification.zh-CN.md)。
- 完成英文主文档及完整简体中文配套，并补充双语源码注释。

## 0.1.0 — 2026-11-01

- 实现独立 Go MCP 服务，提供 13 项 SSH、PTY／会话及 SFTP 工具。
- 接入官方 OpenAI 出站隧道，提供原生系统服务安装和 Docker 一次性初始化。
- 增加本地协议测试及隔离 Docker OpenSSH 验收测试。
