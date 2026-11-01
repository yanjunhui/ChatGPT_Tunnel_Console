<!-- Created by Yanjunhui -->
# init 初始化与开机服务

[English](2026-11-01-init-service.md) | 简体中文

> 历史记录：本文描述标题所列版本或阶段的实现与验证情况。当前安装与使用方法请参阅[文档中心](../../README.zh-CN.md)。

历史实施计划。文中验收状态以当时阶段为准；当前部署状态见 [部署记录](../live-deployment.zh-CN.md)。

目标：原生 Go CLI 增加 `init`，首次引导填写隧道/SSH 配置，自动安装并启动 macOS launchd / Linux systemd 系统服务。用户已确认采用原生 Go 服务。

- [x] Go 初始化向导，隐藏密钥输入，支持导入现有 TOML。
- [x] 固定安装目录保存程序、文件型凭据和配置，重跑默认复用配置。
- [x] 注册系统级服务、立即启动、启用开机启动；服务运行账户默认取 sudo 调用者。
- [x] 新安装先准备完毕，停止旧实例后切换；启动失败恢复旧安装。
- [x] 参数、转义、凭据持久化、服务调用顺序、失败回滚测试；静态检查与跨平台编译。
- [x] 更新 CLI 帮助、README 和本次验证边界。

安装布局：`/usr/local/lib/tunnel-ssh-mcp/releases/` 保存每代完整安装，`current` 指向生效版本；`/var/lib/tunnel-ssh-mcp` 为服务工作目录，`/var/log/tunnel-ssh-mcp` 为 macOS 日志目录。macOS 服务文件位于 `/Library/LaunchDaemons`，Linux 位于 `/etc/systemd/system`。服务不依赖交互终端环境或登录会话。

本次只实现初始化能力及隔离验证，不填入虚构凭据去注册真实隧道服务；缺少真实凭据时不声称线上启动或重启验收通过。
