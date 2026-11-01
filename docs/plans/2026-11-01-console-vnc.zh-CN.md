<!-- Created by Yanjunhui -->
# Console 更名与 VNC 实现计划

[English](2026-11-01-console-vnc.md) | 简体中文

**目标：** 将公开项目更名为 ChatGPT_Tunnel_Console，使 ChatGPT/Codex 通过 MCP 截图及输入工具查看和操作 VNC 桌面。

**架构：** 保留官方 OpenAI MCP 隧道。SSH 配置可选提供 VNC 目标，通过 SSH 直接 TCP 转发访问。具有资源边界的 RFB 客户端使用共享、短时连接，不新增公网 VNC 监听或浏览器查看器，现有 SSH/SFTP 接口保持兼容。

**技术栈：** Go、现有 MCP SDK 与 SSH 库、RFB 3.3/3.7/3.8 Raw 编码、PNG MCP 图片、可选 TigerVNC/Openbox Docker 环境。

## 任务

1. 更名 Go 模块、CLI 源码包、项目显示名称、仓库链接和 GitHub 仓库。已有安装路径、Compose 项目／数据卷、环境变量名称作为部署兼容标识保留并说明，不重启已有部署。
2. 新增 `internal/config/vnc.go`、可选 VNC 字段及默认值、无效端口／超时、凭据引用测试。扩展 `internal/setup/configuration.go`，包括内置开发配置在内均能持久化 VNC 凭据。
3. 新增 `internal/vnc`：有界 RFB 握手、None/VNC 鉴权、Raw 帧缓冲解码、尺寸变化和输入事件。测试协议版本、挑战响应、错误／截断数据、尺寸限制、鉴权失败及关闭连接取消操作。
4. 新增 `internal/remote/desktop.go`，实现 SSH 转发和有界串行操作；新增 `internal/server/desktop.go`，注册 `vnc-screenshot`、`vnc-pointer`、`vnc-key`、`vnc-type`。连接前验证全部输入，在请求内释放按键／鼠标按钮，返回原生 MCP 图片和明确的输入发送结果。
5. 新增可选桌面 Docker 构建目标及 `compose.desktop.yaml`。VNC 绑定目标机回环地址，仅通过 SSH 访问。扩展隔离 Docker 测试，验证实际截图解码、鼠标移动、键盘输入及无公开端口。
6. 同步两种 README、命令／配置／工具参考、安全说明、第三方声明、更新与验收记录。执行格式、竞态测试、vet、交叉编译、漏洞／敏感信息扫描和真实 Docker 测试，查看实际桌面截图；更名并推送 GitHub 仓库，核对发布提交的 CI。

## 验收与边界

- 旧配置在 VNC 禁用时继续正常工作；桌面功能须显式配置并存在可用 VNC 服务。
- 支持标准 RFB Raw 更新和通过 SSH 的传统 VNC 密码认证；拒绝不支持的安全／编码类型，不静默降级已配置密码的认证。
- 截图保留原始像素尺寸，限制内存／输出并设置请求期限；输入结果只确认已发送，不等于应用操作完成。
- 不包含跨请求保持按键、剪贴板同步、视频流、浏览器查看器或自动为外部主机安装桌面。
- 保留正式 0.1.1 容器；本地／CI 验收与后续真实 ChatGPT 部署分别记录。

## 进度

- [x] 确认 MCP 截图和键鼠交互范围。
- [x] 实现更名、协议、配置及 MCP 工具。
- [x] 通过本地和真实 Docker 桌面验证。
- [x] 同步双语文档并发布已验证源码。
