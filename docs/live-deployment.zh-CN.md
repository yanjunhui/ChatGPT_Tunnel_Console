<!-- Created by Yanjunhui -->
# 本机 Docker 真实部署与隧道验收

[English](live-deployment.md) | 简体中文

日期：2026-11-01（Asia/Shanghai）。项目：`ChatGPT_Tunnel_Console`。

## 当前部署：0.3.2

正式 MCP 于 2026-11-01 20:16:44 升级至 **0.3.2**。自带原生 X11 桌面及独立管理的 VNC 服务沿用已验证的 0.3.0 镜像；0.3.1 截图与 0.3.2 Unicode 修复仅替换 MCP。原隧道凭据、SSH 身份、工作区及开发主目录均保留，没有宿主机端口映射。

| 检查 | 实际结果 |
|---|---|
| 运行版本 | `tunnel-console-mcp version` 返回 `0.3.2` |
| 运行镜像 | `go-tunnel-console-mcp:0.3.2`、`go-tunnel-console-desktop:0.3.0`、`go-tunnel-console-vnc:0.3.0` |
| 可执行文件 SHA-256 | `1fe712315dc2c3faff3b16443760e4da7653b1d16a60a2efc6c8df604458603c` |
| 隧道健康 | 要求控制面轮询成功；live/ready HTTP 200，结果 `ok` |
| 真实 ChatGPT SSH | 返回 `developer`、`/workspace`、退出码 0；配置发现另确认两种桌面能力已启用 |
| 真实截图 | 两条路径均返回可见的 1280×800 PNG；ChatGPT 分别读出未在提示中提供的 `VISUAL_PROOF_CEDAR_5873`，没有经 SSH/SFTP 读取 |
| 原生输入 | 鼠标、Control+u、文本和回车生成 `NATIVE_032:中文汉字αβ日文あいう`，精确回读通过，鼠标 `(407,663)` |
| VNC 输入 | 鼠标、组合键、文本和回车生成 `VNC_031:中文`，精确回读通过，鼠标 `(410,203)` |
| 停止 VNC | VNC 截图返回 `Connection refused`；原生截图及全部输入工具仍可用，并生成 `NATIVE_WITH_VNC_STOPPED:中文` |
| 服务独立性 | 停止 VNC 未重启 MCP 或 dev；dev 启动时间保持 19:54:33、MCP 保持 20:16:44 |
| 恢复 VNC | 恢复 `healthy`；真实 ChatGPT 再次调用 VNC 截图／鼠标／按键／文本，精确回读 `VNC_RESTORED_032:中文` |
| 清理 | 五个已知测试文件核对哈希、私有归档后移除；专用目录已删除，并独立确认不存在；ChatGPT `list-sessions` 返回 `[]` |
| 源码 CI | [0.3.2 源码提交](https://github.com/yanjunhui/ChatGPT_Tunnel_Console/actions/runs/41305716797)四项任务全部通过，含 Linux amd64 Docker 集成 |

截图 A/B 对照及最初 Unicode 失败保留于[验证记录](verification.zh-CN.md)。这些是真实失败并触发修复，未计为验收成功；输入送达确认不作为应用处理成功证据。结果覆盖自带 Linux X11 桌面，不包含原生 macOS/Windows/Wayland 桌面、宿主机重启或断网恢复。

### 当前操作与回退

本地 `.env` 选择三个 Compose 文件及明确镜像标签。使用 `docker compose stop vnc` / `docker compose start vnc` 仅管理可选服务，`desktop-*` 独立于它。两条后端仍依赖 SSH，并共享 dev 提供的 X 显示服务；仅原生配置及外部 VNC 目标见[桌面指南](desktop.zh-CN.md)。

私有备份及证据位于 `.cache/upgrade-0.3.0/`，包括升级前 data／工作区归档、开发主目录／SSH 身份卷归档、原环境文件、容器快照、部署日志、健康结果及 ChatGPT 证据。0.2.0/0.3.0/0.3.1 镜像继续保留。仅回退 MCP 时，应先停止 MCP，在 `.env` 恢复所选镜像标签，再以 `--no-deps --no-build --pull never` 仅重建 MCP；此过程会结束内存会话。旧版也会恢复其已记录的截图／Unicode 限制。完整回退至 0.2.0 还需对应配置代次及双文件 Compose 设置。本次成功升级后没有执行回退演练。

以下全部为历史部署记录，不代表当前容器清单或当前回退步骤。

## 历史 0.2.0 部署

正式环境于 2026-11-01 19:30:42 升级至 **0.2.0**，要求控制面轮询成功的健康检查及真实 ChatGPT SSH 通过；该阶段缓存工具列表未暴露 VNC 工具，因此没有宣称截图／输入验收完成。私有备份及证据保留于 `.cache/upgrade-0.2.0/`。

## 历史 0.1.1 部署

正式 MCP 容器已于 2026-11-01 17:25:35 升级至 **0.1.1**。真实 ChatGPT → OpenAI 隧道 → Go MCP → SSH/SFTP 链路已验收通过，包括中文下载、流式文件往返及跨调用保留变量和目录的交互终端。专用测试文件及会话已清理。

开发容器没有重建或重启。现有隧道、配置、凭据、工作区及具名卷保持沿用；本轮没有修改其他 cloud-dev 或 Mac 开发机隧道。

| 检查项 | 结果 |
|---|---|
| MCP 版本 | 运行容器内 `tunnel-ssh-mcp version` 返回 `0.1.1` |
| MCP 容器 | `go-tunnel-ssh-mcp-1`，运行中，重启次数 0 |
| 开发容器 | `go-tunnel-ssh-dev-1`，原容器 ID 不变，`healthy`，重启次数 0 |
| 持久化及端口 | 升级前后挂载一致；配置及 SSH 身份对 MCP 只读；无宿主机端口映射 |
| 重启策略 | 两个服务均为 `unless-stopped` |
| 官方健康检查 | `healthz` HTTP 200 / `live`；`readyz` HTTP 200 / `ready`；要求控制面成功轮询的检查返回 `result: ok` |
| 真实隧道调用 | 截至 17:35:41，18 次命令转发及 18 次响应提交 HTTP 200；成功轮询 33 次，含获取命令 HTTP 200 与空轮询 HTTP 204 |
| 官方 doctor | 独立一次性容器执行，`RESULT ok`；不将 doctor 单独作为在线证明 |
| ChatGPT SSH | 实际返回 `developer`、`/workspace`、`E2E_V011_20261101_B1`，退出码 0 |
| ChatGPT 下载 | `sftp-download` 直接返回中文正文、`encoding: utf8`、`bytes: 33`，本次未出现空对象 |
| SFTP 往返 | 内联上传、流式下载、流式上传完成；三个 33 字节文件 SHA-256 一致 |
| 持久 PTY | 实际读到 `PTY:V011_PTY_B1:/workspace/.e2e-go-tunnel-011-b1:END` |
| 会话及文件清理 | 关闭后 `list-sessions` 返回 `[]`；清理命令退出码 0；宿主机确认新旧验收目录均不存在；开发容器无残留 SSH 会话进程 |

关闭终端时工具返回 `process_exit_observed: false`，表示当时尚未观察到远端退出，不能将其改写为同步退出成功。后续开发容器进程检查仅剩监听 sshd，确认没有残留测试会话。

## 版本与部署证据

- MCP 镜像标签：`go-tunnel-ssh-mcp:latest`、`go-tunnel-ssh-mcp:0.1.1`。
- MCP 镜像：`sha256:5c85dfb439f629a05b1a9f3fd90e477e1527f3696b74e5ff216543cae8043a81`。
- 运行中 Go 二进制 SHA-256：`873dd5feb19b90dc4b7b19a7e7a19d629a3309938aad1e49d54abf4b6e3f5ebe`。
- 开发镜像：`sha256:de4f8528ea967944a30097fb1732771d80aa807889ba17624de5ba8e73b0b114`。

升级前确认没有活跃 SSH 会话，保存回退镜像后先停止旧 MCP，再单独重建 MCP；没有同时运行同一 tunnel ID 的两个隧道实例。实际部署使用 `docker compose up -d --no-deps --no-build --pull never --force-recreate mcp`。

新实例启动时元数据请求出现一次 EOF 警告；首次成功轮询前，要求控制面轮询的健康检查未通过。约 31 秒后首次轮询成功，后续真实命令与响应均成功；不把这次启动元数据请求描述为 HTTP 200。

## 真实 ChatGPT 验收

验收复用了已有专用隧道和账号中已安装的连接。私人账号、隧道／插件标识和聊天链接仅保留在本地证据归档中，不纳入公开记录。

验收聊天完成了连接列表、SSH 命令、SFTP 下载／上传／流式往返、持久终端及清理。网页曾暂时无法加载回复，改用可用模型后完成续测，期间容器和隧道轮询持续正常。

本次下载直接返回：

```json
{"content":"E2E_V011_B1：中文回读测试\n","encoding":"utf8","bytes":33}
```

`source.txt`、`stream-copy.txt`、`upload-copy.txt` 的 SHA-256 均为 `06b3cf70759145885b30f24cec4db0c3ff907453cc291bbc80dce3d3b77ebb48`。通过后只删除这三个文件并移除 `.e2e-go-tunnel-011-b1`；旧 `.e2e-go-tunnel-20261101-a1` 夹具在再次核对文件名和历史哈希后也已清理。

0.1.0 的直接 stdio 响应已包含完整正文，但缺少 `outputSchema` 和 `structuredContent`。0.1.1 使用 MCP Go SDK 类型化输出补齐这两项，并保留原文本 JSON。升级后真实 ChatGPT 下载不再出现空对象，兼容性验收通过；这仍不能证明旧网页现象只有这一种原因。

## 正式容器内的独立 SSH 验收

另通过 `docker compose exec` 启动独立 stdio MCP 进程，验证 0.1.1、13 个工具及下载输出契约；真实 SSH 返回 `developer`、`/workspace`、`aarch64`、sudo UID 0 和唯一标记。覆盖中文、空文件、二进制、缺失文件错误、流式往返、结构化/文本结果一致性、PTY 状态与关闭清理，全部通过。

这组验收直接连接正式容器中的 SSH 后端，**不经过 OpenAI**；与上一节真实 ChatGPT 调用分开记录。没有再次运行与部署无关的全量源码测试；该镜像构建前的测试结果见 [验证记录](verification.zh-CN.md)。

## 回退保留

旧镜像索引已丢失，直接打标签及 `docker commit` 均因缺失 manifest 失败。停止旧容器前，已导出其根文件系统并导入，恢复原 ENTRYPOINT、CMD、USER、WORKDIR 和 ENV：

- 回退标签：`go-tunnel-ssh-mcp:rollback-0.1.0-20261101`。
- 回退镜像：`sha256:477934c79009e06cc3f4ed5363bf1cd750516e0df0db8103095387ce56df0030`。
- 离线启动该镜像执行 `version`，返回 `0.1.0`；旧二进制哈希核对一致。未执行线上回退演练。
- 导出归档不含配置、SSH 身份或工作区的挂载数据；这也符合 [Docker export 的挂载卷语义](https://docs.docker.com/reference/cli/docker/container/export/)。原持久化目录和卷继续保留。

需要回退时执行下列命令；会结束当前内存会话，应先处理现存任务。本轮未执行：

```sh
docker compose stop mcp
docker image tag go-tunnel-ssh-mcp:rollback-0.1.0-20261101 go-tunnel-ssh-mcp:latest
docker compose up -d --no-deps --no-build --pull never --force-recreate mcp
docker compose exec -T mcp tunnel-ssh-mcp version
docker compose exec -T mcp tunnel-client health --port 8080 --require-control-plane-poll --json
```

首次成功轮询前最后一项可能暂时失败，需结合启动日志和后续轮询确认。重新部署 0.1.1 时，以 `go-tunnel-ssh-mcp:0.1.1` 替换上述回退标签，仍先停止旧实例再启动。

## 证据位置

本轮证据位于 `.cache/upgrade-0.1.1/`，已被忽略规则排除：

- `before.json`、`after.json`、`final-state.json`：容器、镜像、挂载与重启状态。
- `health-final.json`、`metrics-final.txt`、`startup.log`、`doctor-isolated.log`：正式隧道健康和诊断。
- `chatgpt-final.txt`、`chatgpt-verified.png`、`chatgpt-forwarded-events.json`：网页结果、截图和 18 次实际命令转发。
- `chatgpt-file-hashes.json`、`cleanup.json`、`dev-processes-final.txt`：真实文件对照及清理核验。
- `ssh-verification.json`、`stdio-events.json`、`verify_runtime.py`：独立 stdio/SSH 验收。
- `rollback-rootfs.tar`、`rollback-export.json`、`rollback-image.json`：回退材料。

真实配置及凭据仍位于忽略提交的 `data/`，导入材料位于 `init-input/`；本文及验收截图不包含密钥。官方接入依据：[Secure MCP Tunnel](https://developers.openai.com/api/docs/guides/secure-mcp-tunnels)。
