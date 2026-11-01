<!-- Created by Yanjunhui -->
# 验证记录

[English](verification.md) | 简体中文

日期：2026-11-01（Asia/Shanghai）。正式运行状态以 live-deployment.zh-CN.md 为准；下方早期快照对应 0.1.0，最新升级验收记录在文末。

前半部分保留初版交付快照；原生 `init`、Docker 初始化和下载返回兼容性的增量验证分别记录在文末。各节的未验收范围只描述该阶段；真实隧道的后续进度以 [部署记录](live-deployment.zh-CN.md) 为准。

## 已通过

| 检查 | 结果 |
|---|---|
| 完整测试 `go test -race -count=1 -timeout=120s -json ./...`，启用 `TUNNEL_SSH_DOCKER_TEST=1` | 27 项通过，0 失败，0 跳过；包含一个 stdio 子进程测试入口 |
| `go vet ./...` | 通过 |
| macOS arm64 原生构建 | 通过 |
| Linux arm64 / amd64 静态交叉编译 | 通过 |
| 官方 tunnel-client 下载和 SHA-256 校验 | v0.0.19，通过 |
| 官方二进制版本与 CLI 参数核对 | `0.0.19+01d67bb4da0d8359b808b4761f8db171ab682cd7`；支持使用的 stdio 和凭据引用参数 |
| Docker MCP 和开发镜像构建 | linux/arm64，通过 |
| Docker 中 MCP → OpenSSH → 开发环境 | 通过：身份、目录、sudo、Go/Python/Node/npm/fd、持久 PTY、SFTP 上传/下载/流式下载 |
| 测试资源清理 | 测试项目的容器和卷均已清理，无运行中的测试隧道 |
| 配置模板解析 | 原生和 Docker 两份模板均通过 |
| 源码长度检查 | Go 函数均不超过 50 行，单文件非空行均不超过 999 行 |

命令/SFTP 测试使用真实 SSH 协议连接临时回环服务器；容器测试额外使用真实 OpenSSH 服务。MCP 测试覆盖 SDK 内存传输和独立 stdio 子进程，发现 13 个工具并完成真实 SSH 调用。

覆盖的边界包括：空命令/文件、无效配置及未知字段、端口和缓存上界、负数/超大超时、零超时、输出截断和游标、缺失工作目录、非零退出、命令取消、并发会话/重复名称、会话退出、无效 base64/UTF-8、大文件流式传输、失败下载保留已有文件、任意绝对路径上传。

已修复并验证：工作目录切换失败后多行脚本继续执行的问题；Docker OpenSSH 会话默认 PATH 找不到 Go 的问题。开发镜像构建遇到的 Debian HTTP 下载失败已通过 HTTPS 源及官方 Node/npm 二进制依赖解决。

## 初版镜像快照

```text
go-tunnel-ssh-mcp:latest
sha256:74f67985961ba53351771fa8fc7c20dc9b412bd24c73deaeed9aa9a638f4c3a4

go-tunnel-ssh-dev:latest
sha256:b30b7cae40a042215e86431727316dde947febcec84e522bc5d5c51c42154e68
```

容器验收实际返回：`developer`、`/workspace`、sudo 后 UID `0`、Go `1.26.10 linux/arm64`、Python `3.11.2`、Node `24.22.2`、npm `11.19.1`、fd `8.6.0`。MCP 镜像内服务版本读取为 `0.1.0`，官方隧道版本读取为 `0.0.19`。

可复查的本地构建日志位于项目 `.cache/docker-build.log`、`.cache/build-checks.log`；测试事件位于 `.cache/test-results.jsonl`。这些是本次验证的生成文件，已加入忽略规则。

## 初版交付时尚未验收的范围

未配置或读取任何真实 OpenAI 隧道凭据，未启动线上隧道，未修改既有部署。尚未验证真实 ChatGPT/Codex → OpenAI 隧道 → 本服务的线上调用、断线重连及服务端鉴权结果。配置自己的 tunnel ID 和运行时 API Key 后，按 README 执行线上接入步骤；容器/本地测试通过不替代这项验收。

Linux amd64 已编译，但没有在 amd64 实机运行 Docker 验收；本次 Docker 实测架构为 arm64。

## init 初始化与系统服务增量验证

日期：2026-11-01（Asia/Shanghai）。

- 完整 `go test -race -count=1 -timeout=120s -json ./...`：44 项通过、0 失败、1 项跳过。跳过的是需要显式启用的旧 Docker OpenSSH 验收，本次未重跑该项目。
- `go vet`、macOS arm64 原生构建、Linux arm64/amd64 构建全部通过。
- 真实 macOS `plutil -lint` 校验生成的 LaunchDaemon plist，通过。
- 隔离 Linux 容器中执行真实 `systemd-analyze verify`，校验生成的 unit，通过；含空格和百分号的路径也通过。
- 初始化测试覆盖交互字段、端口错误、EOF、空凭据、加密私钥/错误口令、SSH agent 的登录生命周期限制、环境变量凭据落盘、权限 0600、无密钥输出、首次服务失败清理、更新失败恢复、重复更新、并发初始化互斥。
- 使用真实 PTY 验证密钥输入编辑、Ctrl+C、超时取消和终端模式恢复。
- 服务管理器调用由测试记录器验证停止/注册/启用/启动/回滚顺序。没有使用真实隧道凭据，未向本机 `/Library/LaunchDaemons` 或 Linux 宿主机 `/etc/systemd/system` 安装常驻服务，未进行重启机器后的自启实测。

系统服务管理器确认启动与 OpenAI 鉴权上线仍是不同验收项。Linux 单元使用 `Type=exec`，让执行文件缺失等启动错误由管理器报告；规则参考 [systemd.service 官方文档](https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html)。macOS 定义依据本机 `launchd.plist(5)` 和 `launchctl(1)` 手册核对。

本次日志：`.cache/init-test-results.jsonl`、`.cache/init-build-checks.log`；隔离服务文件位于 `.cache/service-check/`。安装器向导与文件操作实现位于 `internal/setup/`，系统服务定义和控制位于 `internal/service/`。

## Docker 初始化增量验证

日期：2026-11-01（Asia/Shanghai）。测试使用当前 Go 源码重建的 linux/arm64 产品镜像。

| 检查 | 结果 |
|---|---|
| 完整 `go test -race -count=1 -timeout=120s -json ./...` | 54 项通过、0 失败；2 项 Docker 测试在下一步单独启用 |
| `TUNNEL_SSH_DOCKER_TEST=1 go test -race -count=1 -timeout=240s -json -run '^TestDocker' ./cmd/tunnel-ssh-mcp` | 2 项通过、0 失败、0 跳过 |
| 首次未提供配置的非交互 init | 返回错误，未生成当前配置 |
| 真正的 `compose.init.yaml` 非交互导入 | 成功；初始化时开发环境及其 SSH 私钥尚未创建 |
| 输入文件移除后重复 init | 成功；凭据已持久化，不依赖导入文件 |
| Docker 真实 PTY 交互重配 | 隧道 ID/API Key/默认开发环境向导完成，密钥不回显，一次性容器正常退出 |
| 正式 Compose 读取配置 | UID/GID 1000 能读取配置和密钥；文件所有者 1000:1000、权限 0600；配置挂载禁止写入 |
| 生成配置驱动 MCP → OpenSSH | 首次导入、重复初始化和交互重配后均通过：发现 13 个工具、命令执行、sudo、工具链、SFTP 往返、持久 PTY |
| 失败与边界 | 空/缺失凭据、非法目标、EOF、混用参数、非法 UID/GID、发布失败、取消及并发 init 均有覆盖；失败不替换旧配置 |
| 原生初始化回归 | 保留原有 systemd/launchd 安装、凭据复制、失败恢复和 PTY 测试，全部通过 |
| 静态检查及构建 | `go vet`、macOS arm64、Linux arm64/amd64 构建通过；函数不超过 50 行、文件非空行不超过 999 行 |
| Compose 定义 | 独立 init 服务无现有凭据依赖、端口或服务依赖；运行服务只读挂载配置，使用 `unless-stopped` |
| 测试清理 | 隔离项目的容器及具名卷已清理；未启动线上隧道 |

当前镜像：

```text
go-tunnel-ssh-mcp:latest
sha256:9ed327134941a2253fd8e43c7ce0a42013043932d8efa4ad0860edd6a3c3ca32

go-tunnel-ssh-dev:latest
sha256:de4f8528ea967944a30097fb1732771d80aa807889ba17624de5ba8e73b0b114
```

测试通过正式 Compose 启动隔离的开发容器，并将 MCP 命令覆盖为 `serve --config /etc/tunnel-ssh/current/config.toml`，因此验证了生成配置到真实 SSH 的链路，没有使用虚拟凭据连接 OpenAI。正式镜像默认命令已核对为 `run --config /etc/tunnel-ssh/current/config.toml`。

本次没有真实 OpenAI 隧道凭据，未进行 ChatGPT/Codex 在线接入验收，也未测试宿主机重启后的 Docker 自动启动。Linux amd64 完成构建，实际容器验收仍为 arm64。原生系统服务没有安装到本机。

日志：`.cache/docker-init-unit-results.jsonl`、`.cache/docker-init-integration-results.jsonl`、`.cache/docker-init-build.log`、`.cache/docker-init-build-checks.log`。解析后的 Compose 定义位于 `.cache/docker-init-compose.json` 与 `.cache/docker-runtime-compose.json`；均为忽略提交的验证产物。

## 0.1.1 下载返回兼容性验证

日期：2026-11-01（Asia/Shanghai）。本节记录升级前的本地修复阶段，当时用户限定本地验证，未更新正式容器；后续授权部署结果见下一节。

旧版直接 stdio 响应含完整文件文本，三个实际传输文件的内容与哈希一致；ChatGPT 显示空对象的原因尚未确定。可以确认的本地差异是下载工具没有 `outputSchema` 和 `structuredContent`。新契约测试在旧实现上失败；0.1.1 使用 SDK 类型化结果补齐这两项，并保留原有文本 JSON。该变更不是线上根因已确认的证明。

| 检查 | 结果 |
|---|---|
| 完整 `go test -race -count=1 -timeout=120s -json ./...` | 55 个顶层测试通过（计入子测试为 69 个通过事件）、0 失败；2 个 Docker 测试随后单独启用 |
| `TUNNEL_SSH_DOCKER_TEST=1 go test -race -count=1 -timeout=240s -json -run '^TestDocker' ./cmd/tunnel-ssh-mcp` | 2 项通过、0 失败、0 跳过 |
| 下载结果契约 | 发现接口包含三个必填字段及其类型；结构化对象与原文本 JSON 解码后完全一致 |
| 成功边界 | 空文件、中文及换行、内容本身为 JSON、含 NUL 和非 UTF-8 字节的 base64、恰好达到内联上限 |
| 失败边界 | 缺少参数、空路径、文件不存在、未知 profile、无效编码、无效 UTF-8、超出内联上限；均返回工具错误且无成功结构化结果 |
| 隔离 Docker / OpenSSH | 13 个工具可发现；短命令、sudo、工具链、中文内联下载、流式下载和流式上传后回读通过；首次初始化、重复初始化和真实 PTY 向导重配后均通过 |
| 持久终端 | 真实 OpenSSH PTY 在不同工具调用之间保留环境变量与目录，读取到预期标记并关闭；这属于本地容器验收 |
| 静态检查和构建 | `go vet`、macOS arm64 原生构建、Linux arm64/amd64 静态构建通过；Go 函数不超过 50 行、文件非空行不超过 999 行 |
| 版本核对 | 本机二进制及新 Docker 镜像均返回 0.1.1；正式容器仍返回 0.1.0，容器 ID、镜像 ID 与重启次数保持原值 |
| 测试清理 | 隔离测试容器、网络和具名卷由测试清理完成；回读未发现测试容器或测试卷，仅保留正式容器及其持久化卷 |

本地 MCP 镜像 `go-tunnel-ssh-mcp:latest`：`sha256:5c85dfb439f629a05b1a9f3fd90e477e1527f3696b74e5ff216543cae8043a81`，实际容器测试架构为 Linux arm64。未重建开发环境镜像，仍使用 Docker 初始化阶段的开发镜像。Linux amd64 仅完成交叉编译。

日志及计数：`.cache/download-result/before-tests.log`、`unit-results.jsonl`、`docker-results.jsonl`、`test-summary.json`、`build-checks.log`、`docker-build.log`。该目录的 `before/` 保留本轮修改前的目标文件快照，便于在尚无 Git 仓库的工作区内核对改动。

输出约定参考 [OpenAI 官方 MCP 结果说明](https://developers.openai.com/plugins/build/mcp-server#return-useful-results-without-ui)；具体编码及错误行为同时核对本地锁定的 MCP Go SDK v1.8.0 源码。该阶段尚未进行的线上下载与 ChatGPT PTY 验收，已在下述正式升级阶段完成。

## 0.1.1 正式升级与真实隧道验收

日期：2026-11-01（Asia/Shanghai），17:25:35 升级正式 MCP，17:35:41 完成证据和清理核验。用户已明确授权正式升级及隧道、SSH 验收。本阶段未修改产品源码，部署上节已通过测试的 0.1.1 镜像。

| 检查 | 结果 |
|---|---|
| 实际运行版本 | 正式容器内返回 0.1.1；镜像及二进制 SHA-256 与本次部署记录一致 |
| 部署范围 | 仅重建 MCP；开发容器 ID/启动时间不变；挂载、凭据引用、无端口映射和重启策略保留 |
| 官方健康检查 | `health --port 8080 --require-control-plane-poll --json` 返回 ok，live/ready 均 HTTP 200 |
| 真实控制面 | 33 次成功轮询，18 次转发，18 次响应提交 HTTP 200；最后成功轮询时间推进 |
| 正式容器 stdio → SSH | 13 工具、版本、输出契约、身份/目录/sudo、中文/空/二进制/缺失文件、流式往返和持久 PTY 全部通过；此组不经过 OpenAI |
| ChatGPT → 隧道 → SSH | 实际命令返回 developer、/workspace、专用标记及退出码 0 |
| ChatGPT SFTP | 下载直接返回 `E2E_V011_B1：中文回读测试\n`、utf8、33 bytes；内联上传及流式往返后三个文件哈希一致 |
| ChatGPT PTY | 跨两次输入调用保留变量和目录；真实输出 `PTY:V011_PTY_B1:/workspace/.e2e-go-tunnel-011-b1:END` |
| 会话清理 | close 返回尚未观察到退出，随后 list 为 `[]`；容器进程检查无残留 SSH 会话 |
| 文件清理 | 新旧专用验收目录及独立 stdio 测试目录均已清理，未删除其他工作文件 |
| 回退材料 | 保留 0.1.0 根文件系统回退镜像，恢复启动配置并离线确认 version 为 0.1.0；未线上回退 |

真实 ChatGPT 下载空对象现象本次未复现，兼容性验收通过；不把这一结果扩大为旧问题唯一根因已确认。中途 ChatGPT 页面回复加载失败且模型菜单提示限额，切换可用模型后续测完成；隧道持续正常。

完整运行标识、原始证据路径、启动瞬态 EOF、回退命令和验收聊天见 [正式部署记录](live-deployment.zh-CN.md)。本阶段没有验证故障断网重连、宿主机重启恢复或 amd64 实机运行，这些不属于本次正常功能验收结果。

## 英文主文档与双语注释验证

日期：2026-11-01（Asia/Shanghai），在正式升级完成之后。本轮只更新文档及源码／配置注释，应用版本仍为 0.1.1；运行时提示和 MCP 工具描述保持不变。

| 检查 | 结果 |
|---|---|
| 源码等价性 | 45 个 Go 文件去除注释后的词法内容在修改前后完全一致 |
| 注释覆盖 | 生产代码导出声明均有说明；解释性 Go 注释包含英文和中文；函数长度符合要求 |
| 配置等价性 | 三份 TOML 模板和三份解析后的 Compose 定义完全一致；Dockerfile、Makefile、shell 和 SSH 配置的非注释内容不变 |
| 文档 | 18 个 Markdown 文件组成 9 组中英文对应文档；本地链接、对应代码示例、SHA-256、文件头、代码围栏和文件长度检查通过 |
| 本地回归 | `go test -race -count=1 -timeout=120s -json ./...`：55 个顶层测试通过，含子测试为 69 个通过事件，0 失败；2 个需显式启用的 Docker 测试跳过 |
| 静态检查／构建 | `make vet build` 和 shell 语法检查通过 |
| 部署范围 | 未重建 Docker 镜像或重启正式服务；本阶段未重复执行真实 ChatGPT 验收或 Docker 集成测试 |

证据位于忽略目录 `.cache/docs-bilingual/`：`before/`、`test-results.jsonl`、`test-summary.json`、`build-checks.log`、`deployment-config-checks.json`、`document-checks.json` 及源码／文档检查辅助文件。上文历史线上结果仍对应其原验收阶段。

## 0.1.2 公开源码发布审查

日期：2026-11-01（Asia/Shanghai）。本阶段准备 MIT 公开源码仓库，不部署 0.1.2：正式 MCP 仍为 0.1.1，容器 ID、镜像、启动时间和重启次数均未改变。

| 检查 | 结果 |
|---|---|
| CLI 问题复现 | 五个子命令原先对 `--help` 返回退出码 1；回归测试在修复前失败、修复后通过。六个带参数的子命令现均返回退出码 0。 |
| 本地测试与编译 | macOS arm64、Go 1.26.10 下，全量竞态测试、`go vet`、本机编译、Linux arm64/amd64 交叉编译及模块整理通过。 |
| Docker 验收 | 使用独立 `public-0.1.2` 镜像标签，两项可选测试通过：初始化、向导、持久化／权限、真实 OpenSSH、PTY 状态及 SFTP 往返。测试容器与卷已清理。 |
| 文档与源码检查 | 13 对、26 份双语文档的链接／示例／哈希检查通过；47 个 Go 文件的公开声明注释、双语注释及函数长度检查通过。 |
| 干净检出与 Linux 归属 | CI 先复现构建临时目录缺失，再复现原生 Linux 测试夹具归属问题。现先准备构建目录；Docker 夹具授予 UID 1000 工作目录访问权限，清理时恢复宿主机归属，配置仍保持私有权限。 |
| 可达漏洞扫描 | 工具链从 1.26.0 提升至 1.26.10 后，govulncheck v1.9.0 未发现代码可达漏洞或导入包漏洞。 |
| 仅模块级告警 | GO-2026-5932 涉及已停止维护的 `golang.org/x/crypto/openpgp` 包。本项目未导入它，公告无已修复版本；保留记录，不屏蔽告警。 |
| 公开内容 | Gitleaks v8.30.1 对暂存源码导出未发现敏感凭据。公开记录已移除私人部署标识；凭据、运行数据、二进制及原始日志均排除在外。 |

本地证据保存在 `.cache/public-release/`，不公开上传。GitHub Actions 独立执行 Linux/macOS 检查及 Linux Docker 集成测试，具体提交状态以仓库 Actions 页面为准。本阶段未安装原生开机服务，也未重新进行 ChatGPT 端到端调用；此前 0.1.1 真实验收保留为独立历史记录。

## 0.2.0 Console 更名与 VNC 验收

日期：2026-11-01（Asia/Shanghai）。本阶段为源码发布，使用独立候选 Docker 镜像标签。正式 0.1.1 容器的 ID、镜像、启动时间及零重启次数保持不变；现存监督进程要求控制面成功轮询的健康检查通过。

| 检查 | 结果 |
|---|---|
| 配置和初始化 | 旧配置关闭 VNC；默认值、无效地址／超时／引用、桌面向导选项及 VNC 凭据持久化通过。删除原导入凭据后仍能正常使用。 |
| MCP 契约 | 可发现 17 个工具，含四项 VNC 工具；截图返回原生 PNG 图片及结构化 width/height/name；桌面工具中仅截图标注只读。 |
| RFB 协议 | 3.3/3.7/3.8、None/VncAuth、独立 DES 挑战向量、认证拒绝／不降级、错误矩形、重叠／不完整帧、尺寸变化及全像素覆盖均通过。 |
| 输入及生命周期 | 鼠标／组合键／文本边界、按下释放顺序、CRLF、真实 SSH 转发中的取消、期限和管理器关闭均通过。 |
| 真实桌面 | MCP stdio → OpenSSH → TigerVNC 返回 1280×800 PNG；真实鼠标到达 (100,100)；MCP 键盘输入在 xterm 执行命令，生成文件回读为 `VNC_KEYBOARD_OK\n`，并已目视检查返回的截图。 |
| 认证 | 第二个真实 TigerVNC 服务使用正确的持久化密码返回 64×48 PNG，错误密码被拒绝，结果不回显密码值。 |
| Docker 部署 | Linux arm64 下基础 SSH/SFTP/PTY/init 回归及桌面 Compose 叠加配置通过；桌面健康检查覆盖 SSH/X 显示服务，VNC/SSH/MCP 均无宿主机端口映射，临时资源已清理。 |
| 本地检查 | 全量竞态、vet、本机 macOS arm64 及 Linux arm64/amd64 交叉编译通过；模块整理无新增依赖。30 份文档组成 15 对双语检查，63 个 Go 文件通过公开声明／注释／函数长度检查。 |
| 模糊测试 | RFB 握手连续 10 秒执行 8,291,917 次，无崩溃；这是有时限的模糊测试证据，不代表协议穷尽验证。 |
| 安全检查 | 未发现可达或导入包 Go 漏洞；保留此前记录的仅模块级 OpenPGP 公告。发布验证包含脱敏历史与待发布源码扫描。 |

最初按窗口标题查找失败，原因是终端 Shell 会重写标题；验收现按可见 XTerm 类查找。随后一次 Docker Desktop 配置权限断言无诊断失败，测试现会在失败时输出不含敏感信息的归属／权限，后续全量运行通过。没有放宽权限或添加重试规避，该次瞬态失败原因尚未确认。

证据私有保存在 `.cache/console-vnc/`，包括返回的桌面 PNG 及测试／构建／扫描日志。公开仓库现为 `ChatGPT_Tunnel_Console`，保留部署标识以兼容升级；GitHub Actions 按提交独立检查 Linux/macOS 和 Docker。本阶段未执行真实 ChatGPT VNC 调用、正式 0.2.0 升级、外部主机桌面安装、重启恢复或断网重连；此前 0.1.1 真实隧道验收仍为独立历史记录。

## 0.3.0 独立桌面验收

日期：2026-11-01（Asia/Shanghai）。原生后端经 SSH 使用 X11，可选 VNC 后端保持独立配置，未新增 Go 依赖。

| 检查 | 实际结果 |
|---|---|
| 本地回归 | 全量竞态测试、vet、macOS arm64 构建及 Linux arm64/amd64 交叉编译通过。 |
| 解码与配置 | XWD 24/32 位存储、两种字节序、行填充、原始像素、非法头／尺寸／颜色表及所有截断位置通过；原生显示／超时校验和导入复制通过。 |
| 生命周期 | 真实 SSH 取消、管理器关闭及队列等待期限均释放连接和桌面互斥。 |
| 真实桌面独立性 | dev 未安装 VNC 程序且 VNC 服务未启动时，原生截图、鼠标、组合键、中文输入和回车通过；随后双路径并用通过；停止 VNC 后，VNC 调用报错，原生操作仍通过。 |
| 可观察输入效果 | 回读鼠标坐标为 (100,100)；xterm 分别执行命令生成四个含中文的精确标记，经 SSH 回读文件核对；原始 1280×800 MCP PNG 已解码并目视检查。 |
| 仅 VNC 配置 | 不含原生设置的独立 profile 成功连接带密码 x11vnc；错误密码被拒绝且不泄露。 |
| Docker 回归 | arm64 下初始化、权限、真实 OpenSSH/SFTP/PTY 和工具链检查通过；临时容器与卷已清理。 |
| 模糊测试 | XWD 连续 10 秒执行 6,762,395 次，无崩溃；仅为有界证据，不代表穷尽证明。 |
| 文档与源码 | 32 份／16 对双语文档通过；72 个 Go 文件通过公开声明／双语注释及函数长度检查。 |
| 安全 | 无可达或导入包 Go 漏洞，保留现有仅模块级 OpenPGP 公告；历史与待发布源码扫描使用脱敏输出。 |

验收中复现并修复：SSH 的 ASCII locale 拒绝中文；xdotool 的 1 毫秒输入间隔可能漏收临时 Unicode 键位（12 毫秒恢复回读）；`mousemove --sync` 在坐标不变时阻塞（一秒复现返回 124）；跨容器 MIT-SHM 访问失败，独立 VNC 因此使用 `-noshm` 的 socket 截图，并关闭 IPv6 监听。

证据私有保存在 `.cache/independent-desktop/`。本地检查不替代[正式部署记录](live-deployment.zh-CN.md)中单列的真实 ChatGPT 调用。本阶段未验证宿主机重启恢复、断网恢复或外部原生桌面安装。
