<!-- Created by Yanjunhui -->
# Go OpenAI Tunnel SSH MCP 实现计划

[English](2026-11-01-go-tunnel-ssh.md) | 简体中文

历史实施计划。文中验收状态以当时阶段为准；当前部署状态见 [部署记录](../live-deployment.zh-CN.md)。

**目标：** 独立用 Go 实现仅经 OpenAI 隧道对外使用的 SSH MCP，附带 Docker Linux 开发环境。

**架构：** 官方 tunnel-client 监督 Go MCP stdio 子进程；Go 服务通过 SSH/SFTP 操作目标机。没有 MCP HTTP 端口、独立 Bearer/OAuth、命令策略、审批、角色、路径黑白名单或每日配额。

**技术栈：** Go、官方 MCP Go SDK、x/crypto/ssh、pkg/sftp、TOML、官方 tunnel-client、Docker Compose。

## 待办

- [x] 配置与隧道启动：`internal/config`、`internal/tunnel`、`cmd/tunnel-ssh-mcp`；校验错误配置、参数转义、密钥不进入命令行。
- [x] SSH/SFTP：`internal/remote`；实现执行、超时、取消、PTY、后台任务、输入输出和文件传输，不重试可能已执行的命令。
- [x] MCP：`internal/server`；注册工具并通过 SDK 客户端验证发现、参数和错误结果。
- [x] 部署：`Dockerfile`、`compose.yaml`、`deploy`；提供本地开发环境、外部主机配置和官方隧道安装。
- [x] 验证：27 项测试通过，含竞态检查、真实 OpenSSH Docker 验收；静态检查、原生和 Linux 双架构构建通过。线上隧道待配置凭据后验收，见 `docs/verification.md`。

## 设计约定

- 不复制或套用两个参考项目的实现，不链接 npm ssh-mcp。
- OpenAI 隧道验证 MCP 调用方；SSH 密钥/密码仅用于目标主机的 SSH 登录。
- 持久会话使用原生 SSH PTY 和显式输入/输出工具，不通过 shell 哨兵伪装命令完成。
- 每次短命令/传输建立独立连接；每个持久会话拥有自己的 SSH 连接，避免一项取消中断其它操作。
- 多行命令、管道、重定向和 sudo 直接交由目标 shell 和操作系统处理。
- 输出环形缓存、超时用于运行稳定性，丢弃字节和超时状态必须显式报告。
- 不部署到现有线上隧道，不读取或复用其它项目凭据。线上端到端验收需单独的隧道配置。
