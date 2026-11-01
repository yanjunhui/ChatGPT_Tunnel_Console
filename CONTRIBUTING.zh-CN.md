<!-- Created by Yanjunhui -->
# 贡献指南

[English](CONTRIBUTING.md) | 简体中文

使用 `go.mod` 指定版本的 Go、用于文档检查的 Python 3，以及用于容器验收的 Docker Compose v2。开发支持 macOS/Linux，远端运行目标为 POSIX SSH 主机。

```sh
git clone https://github.com/yanjunhui/ChatGPT_Tunnel_SSH.git
cd ChatGPT_Tunnel_SSH
make check test vet build linux
```

`make` 将 Go、模块、构建和临时文件保存到 `.cache/`，生成二进制位于 `bin/`，均不得提交。`make test` 执行带竞态检测的真实本地 SSH/SFTP 和 PTY 测试，不连接正式隧道。两个 Docker 测试需要显式启用：

```sh
TUNNEL_SSH_MCP_IMAGE=go-tunnel-ssh-mcp:review \
TUNNEL_SSH_DEV_IMAGE=go-tunnel-ssh-dev:review \
make docker-build docker-test
```

存在正式部署时使用独立镜像标签。测试创建临时容器和卷，使用 stdio 与虚拟凭据，并清理自身资源。不要对需要保留数据的部署运行 `down -v`。

`make vuln` 检查已知且可达的 Go 漏洞，`make secrets` 脱敏扫描 Git 历史。工具版本固定在 Makefile 中，安装及查询数据库需要网络；它们不能替代人工审查。依赖变化后运行 `make tidy` 并提交两个模块文件。CI 包含 macOS/Linux 竞态测试、构建、文档检查、凭据／依赖扫描及 Linux Docker 集成，不使用真实隧道凭据。

## 修改流程

1. 修复前先复现缺陷；有实质行为变化时添加回归测试。
2. 完整理解相关实现，保持范围最小；除非明确说明兼容性变化，否则保留原 MCP 契约。
3. 使用 gofmt，函数不超过 50 行，文件非空行不超过 999 行。新文件用对应注释语法以 `Created by Yanjunhui` 开头。
4. 英文为主，提供完整 `.zh-CN.md` 对应文档。源码注释英文在前、中文在后，说明契约或设计原因；示例、默认值和错误行为同步更新。
5. 执行与改动相关的检查，区分本地测试和真实 ChatGPT 验收，不把输入已发送当作执行成功。
6. PR 聚焦问题或需求，说明结果、验证和兼容性影响。提交使用 Conventional Commits 和中文描述，例如 `fix: 正确处理子命令帮助退出状态`。

普通缺陷使用 issue 表单；凭据泄露或可利用漏洞使用 [SECURITY.md](SECURITY.zh-CN.md) 的私密渠道。不要附带正式配置、私钥、账号标识、隧道 ID 或私人聊天链接，使用虚拟夹具并保留脱敏后的有效错误信息。

## 许可证

贡献按仓库 [MIT 许可证](LICENSE) 接收，只提交你有权贡献的代码和文档。第三方组件保留各自许可证，见[第三方声明](THIRD_PARTY_NOTICES.zh-CN.md)。
