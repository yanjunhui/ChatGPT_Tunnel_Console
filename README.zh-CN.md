<!-- Created by Yanjunhui -->
# ChatGPT Tunnel Console · Go

[English](README.md) | 简体中文

用 Go 独立实现的 SSH、SFTP、原生 X11 桌面及可选 VNC控制台，以 MCP 工具提供能力。对外入口为 OpenAI 官方隧道，MCP 调用方鉴权完全由隧道负责；服务端不提供第二套 Bearer Token、OAuth、角色、审批、命令分类、命令黑白名单、路径沙箱或每日配额。

参考了 `ssh-mcp-main 2` 的功能范围和 `SSH-MCP-main` 的部署方式。没有复制或包装其 TypeScript/Python 实现，也不依赖 npm 的 ssh-mcp。MCP、SSH、会话、SFTP、VNC、配置和启动逻辑均由本项目 Go 代码实现；隧道链路使用官方 tunnel-client。

```text
ChatGPT / Codex
      │ OpenAI 隧道鉴权
      ▼
OpenAI Tunnel ← 出站连接 ← 官方 tunnel-client
                                  │ stdio
                                  ▼
                             Go Console MCP
                                  │ SSH / SFTP / 原生 X11 / 转发 VNC
                                  ▼
                      Docker 开发环境 / 现有 SSH 主机
```

没有 MCP HTTP 监听端口。`serve` 是由 tunnel-client 启动的本地 stdio 工作进程，也可用于本地测试；它本身不验证调用者身份。部署中不要另行将其通过 HTTP、TCP 或 socat 暴露。官方隧道的诊断界面仅绑定回环地址。

当前源码版本：**0.3.3**。本独立项目采用 [MIT](LICENSE) 许可证。部署前请阅读[安全说明](SECURITY.zh-CN.md)：获授权的调用者拥有 SSH 账户的完整权限，当前不校验 SSH 目标主机公钥。

## 文档导航

- [配置与命令行参考](docs/configuration.zh-CN.md)：默认值、凭据引用、初始化和校验范围。
- [VNC 桌面指南](docs/vnc.zh-CN.md)：配置、截图、键鼠操作和边界。
- [MCP 工具参考](docs/tool-reference.zh-CN.md)：21 个工具的参数、结果和错误语义。
- [验证记录](docs/verification.zh-CN.md)及[正式部署记录](docs/live-deployment.zh-CN.md)：历史测试与真实隧道验收证据。

- [贡献指南](CONTRIBUTING.zh-CN.md)、[安全说明](SECURITY.zh-CN.md)、[更新记录](CHANGELOG.zh-CN.md)及[第三方声明](THIRD_PARTY_NOTICES.zh-CN.md)。

## 已实现功能

| 工具 | 行为 |
|---|---|
| `list-connections` | 列出连接配置和保留的会话数，不返回凭据、不假装已经探测连通性 |
| `run-command` | 执行任意单行/多行 shell 命令，支持 stdin、PTY、超时覆盖，返回 stdout、stderr、退出码 |
| `open-session` | 创建交互式 PTY 或后台任务 |
| `write-session-input` | 写入原始终端输入、换行、控制字符，或关闭 stdin |
| `read-session-output` | 按字节游标读取会话输出，明确报告缓存丢弃的字节数 |
| `list-sessions` | 查询运行中/已退出的会话及退出结果 |
| `close-session` | 请求 TERM 并关闭连接，报告是否已观察到进程退出 |
| `signal-process` | 向目标机正整数 PID 发送 INT / TERM / KILL |
| `sftp-list` | 通过 SFTP 列目录 |
| `sftp-download` | 读取 UTF-8 或 base64 内容 |
| `sftp-upload` | 写入 UTF-8 或 base64 内容，覆盖已有远端文件 |
| `sftp-download-file` | 流式下载到 MCP 进程所在文件系统，成功后替换本地目标文件 |
| `sftp-upload-file` | 将 ChatGPT 附件或 MCP 服务端本地文件上传到 SSH 主机 |
| `desktop-screenshot` | 独立获取原生 X11 桌面 PNG，无需 VNC |
| `desktop-pointer` | 直接移动、点击、拖动或滚动 |
| `desktop-key` | 直接按下并释放组合键 |
| `desktop-type` | 直接输入 Unicode 文本，无需剪贴板 |
| `vnc-screenshot` | 返回原尺寸 PNG 的 MCP 图片内容及桌面元数据 |
| `vnc-pointer` | 按截图像素坐标移动、点击、拖动或滚动 |
| `vnc-key` | 按下并释放单键或组合键 |
| `vnc-type` | 通过按键事件输入 Unicode 文本，不访问剪贴板 |

从 0.1.1 起，`sftp-download` 发布 `outputSchema`，成功时同时返回 MCP `structuredContent` 和原有 `content[0].text` JSON。两者均包含 `content`（文件文本或 base64）、`encoding` 和 `bytes`（原始文件字节数）；空文件明确返回空字符串和 `0`。调用方可直接读取结构化字段，旧的文本 JSON 读取方式继续可用。路径、编码、UTF-8 或内联大小检查失败时返回 `isError: true` 和错误文本，不返回成功数据。

不保留 `read-command`、`privileged-command`、`reload-config` 等旧接口：任意命令统一走 `run-command`，sudo 按目标操作系统的实际权限执行，修改配置后重启。没有声称是只读的任意 shell 工具。

## 独立原生桌面与可选 VNC

在 Docker 初始化向导选择 **3：自带图形桌面**，可选 VNC 问题默认选 **n**，仅使用原生桌面，然后执行：

```sh
docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

此选项增加 1280×800 的 Xvfb/Openbox 桌面和 xterm。四项 `desktop-*` 工具经 SSH 直接截图及操作，无需 VNC 服务或客户端。若需 VNC 补充能力，选择 **y** 并增加 `-f compose.vnc.yaml`。独立管理的 x11vnc 服务连接同一显示环境，监听回环端口 5901；停止它不影响原生工具。不发布宿主机端口。详见[原生桌面指南](docs/desktop.zh-CN.md)和 [VNC 指南](docs/vnc.zh-CN.md)。

### 更名与升级兼容

仓库、Go 模块、MCP 实现及主命令现分别为 `ChatGPT_Tunnel_Console`、`github.com/yanjunhui/ChatGPT_Tunnel_Console`、`chatgpt-tunnel-console`、`tunnel-console-mcp`。本地构建和运行镜像保留旧命令别名。原生安装／服务路径、Compose 项目名 `go-tunnel-ssh`、持久卷标识和 `TUNNEL_SSH_*` 配置保持稳定，使升级继续使用原数据。已有检出目录无需更名，尤其是正在被正式容器绑定挂载的目录；新检出使用新目录名。已有配置默认关闭 VNC，本次源码发布不会自动升级运行服务。

## Docker Linux 开发环境

需要 Docker Engine / Docker Desktop 与 Compose v2。首次构建需要联网。

执行以下命令前，先获取源码：

```sh
git clone https://github.com/yanjunhui/ChatGPT_Tunnel_Console.git
cd ChatGPT_Tunnel_Console
```

1. 在项目根目录运行一次性初始化容器：

   ```sh
   mkdir -p data init-input workspace
   docker compose -f compose.init.yaml run --rm --build init
   ```

2. 按向导输入隧道 ID、运行时 API Key，再选择 SSH 目标。默认选择自带开发环境；API Key 输入不回显。使用该隧道具有 Read + Use 权限的运行时凭据，不使用管理 API Key。向导把配置和凭据保存到 `data/` 后退出，`--rm` 只删除一次性容器。

3. 构建并启动：

   ```sh
   docker compose up -d --build
   docker compose logs --tail=100 mcp
   ```

4. 在 ChatGPT 的自定义应用中连接该 OpenAI 隧道，应用鉴权选择无额外认证。启用应用后先调用 `list-connections`，然后调用 `run-command` 执行 `id -un; pwd; go version`。预期用户为 `developer`，目录为 `/workspace`。这一步才验证 ChatGPT → 隧道 → MCP → SSH 的完整链路。

`dev` 容器包含 Go、Python、Node.js/npm、Git、C/C++、CMake、Ninja、tmux、rg、fd-find、jq、tree。`developer` 可使用无密码 sudo，以满足不添加应用操作策略的设计。容器内权限不等于宿主机权限；对 `/workspace` 的操作会修改项目下 `workspace/` 的同一批文件。

`mcp` 容器仅运行 Go MCP 和官方隧道组件。两个容器之间使用自动生成的 SSH 密钥，SSH 和 MCP 都没有宿主机端口映射。`workspace/` 是共享工作目录，也可作为 MCP 的流式文件传输目录。SSH 身份和开发用户主目录保存在具名卷中。

`compose.init.yaml` 独立于正式 Compose，不依赖已有配置、`.env` 凭据或运行中的开发容器。它以容器内 root 写入配置并设置运行账户 UID/GID 1000；正式 `mcp` 使用该非 root 账户，只读挂载配置。初始化容器不注册宿主机 systemd/launchd，不需要 Docker socket，也不启动 OpenAI 隧道。

| 持久化内容 | 位置 |
|---|---|
| 当前配置 | `data/current/config.toml`，容器内 `/etc/tunnel-ssh/current/config.toml` |
| 各代配置和凭据 | `data/generations/init-*/`，配置与凭据文件权限 0600，代次和凭据目录 0700 |
| 导入文件 | `init-input/`，初始化容器内只读 `/input/`；导入后运行不依赖此目录 |
| 共享工作文件 | `workspace/` |
| 开发用户主目录 / 自动 SSH 身份 | Compose 的 `developer_home` / `ssh_identity` 具名卷 |

上述目录已排除在 Git 和镜像构建上下文之外。`current` 是相对符号链接，凭据引用固定到其代次，重新初始化失败不会替换已有配置。备份配置时保存整个 `data/`，并保留权限和符号链接。旧代次保留，以便运行中的进程继续读取原凭据。

原生 Linux 的挂载保留数字用户归属。需授予运行账户 UID/GID 1000 对专用宿主机工作目录的写权限；对于新建的 `./workspace`，可执行 `sudo chown 1000:1000 ./workspace` 设置归属。若必须保留其他宿主机账户的所有权，应使用适当的 ACL。已有文件保留各自权限。受保护的配置代次可能需要宿主机提权才能备份或删除，不要把凭据改为所有人可读；Docker Desktop 文件共享的行为可能不同。

可选复制 `.env.example` 为 `.env`，通过 `TUNNEL_SSH_CONFIG_DIR`、`TUNNEL_SSH_INIT_INPUT_DIR`、`TUNNEL_SSH_WORKSPACE_DIR` 更改宿主机目录。初始化与运行必须使用同一配置目录，容器内均固定挂载为 `/etc/tunnel-ssh`。不要在 `.env` 中填写密钥。

可通过 `TUNNEL_SSH_MCP_IMAGE` 和 `TUNNEL_SSH_DEV_IMAGE` 为验收构建指定独立镜像标签；默认仍为 `go-tunnel-ssh-mcp:latest` 和 `go-tunnel-ssh-dev:latest`。初始化与运行须使用相同 MCP 标签。

### 重新初始化和外部 SSH

再次执行初始化命令默认复用已保存配置。需要重新填写时：

```sh
docker compose stop mcp
docker compose -f compose.init.yaml run --rm init --reconfigure
docker compose up -d --build
```

初始化失败时旧配置仍在，可用 `docker compose start mcp` 恢复原服务。配置不会热加载，已运行的 MCP 需要重启；重启会结束内存中的交互会话。

选择“外部 SSH 主机”时填写主机、端口、用户、目录及登录方式。使用私钥时，先把私钥放入 `init-input/`，向导填写 `/input/私钥文件名`；口令和密码不回显。初始化会验证并复制私钥及凭据，正式服务不依赖导入文件。仅连接外部主机可以只启动 MCP：

```sh
docker compose up -d --build --no-deps mcp
```

### 非交互初始化

将 [导入模板](deploy/config.init.toml) 复制到 `init-input/config.toml`，修改隧道 ID，并将运行时 API Key 保存到 `init-input/openai-tunnel-key`，文件只包含密钥。模板默认连接自带开发环境，也可按 [原生模板](config.example.toml) 配置多个外部 SSH profile；凭据使用 `/input/` 下的文件引用。

```sh
docker compose -f compose.init.yaml run --rm --build -T init \
  --non-interactive --config /input/config.toml
docker compose up -d --build
```

首次非交互初始化没有导入文件会返回错误；已有配置可省略 `--config`。`--reconfigure` 不能与 `--config` 或 `--non-interactive` 同时使用。初始化成功只表示配置已保存并通过结构、凭据文件及私钥格式校验，不表示隧道鉴权或 SSH 连接已经成功。

### 容器管理

```sh
docker compose exec --user developer dev bash  # 进入开发环境
docker compose logs -f mcp                    # 隧道日志
docker compose stop                          # 停止并保留数据
docker compose start                         # 重新启动
docker compose down                          # 删除容器并保留具名卷
```

不要对已有数据使用 `docker compose down -v`，该命令会删除开发主目录和 SSH 身份卷。

正式服务使用 `restart: unless-stopped`，由 Docker 管理异常退出与引擎重启后的恢复；宿主机需自行启用 Docker Engine/Desktop 自动启动，手工停止的容器不会因该策略自动恢复。参见 [Docker 重启策略](https://docs.docker.com/engine/containers/start-containers-automatically/)。

## 原生运行，连接现有 SSH 主机

支持 macOS/Linux 的 amd64/arm64。模块需要 Go 1.26.10 或更高版本；支持自动工具链的较早 Go 会按 `go.mod` 下载所需版本。Makefile 把构建、模块和临时缓存放在项目 `.cache/`。

### 初始化并自动注册开机服务

```sh
make build
./bin/tunnel-console-mcp init
```

`init` 会引导输入 OpenAI 隧道 ID、运行时 API Key、SSH 主机/端口/用户/工作目录，以及私钥或密码。密钥和密码输入不回显；Ctrl+C 取消时恢复终端状态。安装器先提示系统写入路径，普通账户执行时自动通过 `sudo` 获取注册服务所需权限。

初始化会把 Go 程序、官方隧道客户端、配置和 SSH 凭据复制到固定目录；若未找到可复用的 tunnel-client，则下载并校验官方发行包。随后停止已有服务，切换到新安装，再注册、启用并立即启动服务：

- macOS：系统级 LaunchDaemon，服务名 `com.yanjunhui.tunnel-ssh-mcp`，不要求用户登录。
- Linux：systemd 系统服务 `tunnel-ssh-mcp.service`，启用 `multi-user.target` 开机启动。

服务默认使用 `sudo` 的原调用账户运行；直接以 root 执行时默认使用 root。可用 `--service-user 用户名` 明确指定本机运行账户。服务的 HOME 和工作目录为 `/var/lib/tunnel-ssh-mcp`，无需交互 shell 环境。

| 内容 | 固定位置 |
|---|---|
| 当前程序及配置 | `/usr/local/lib/tunnel-ssh-mcp/current/` |
| 配置文件 | `/usr/local/lib/tunnel-ssh-mcp/current/config.toml` |
| 各代安装和私有凭据 | `/usr/local/lib/tunnel-ssh-mcp/releases/`，配置/凭据文件权限 0600 |
| 服务工作目录 | `/var/lib/tunnel-ssh-mcp` |
| macOS 服务文件 | `/Library/LaunchDaemons/com.yanjunhui.tunnel-ssh-mcp.plist` |
| macOS 日志 | `/var/log/tunnel-ssh-mcp/stdout.log`、`stderr.log` |
| Linux 服务文件 | `/etc/systemd/system/tunnel-ssh-mcp.service` |
| Linux 日志 | `journalctl -u tunnel-ssh-mcp.service` |

也可导入已有配置，支持多个 SSH profile：

```sh
./bin/tunnel-console-mcp init --config ./config.toml
sudo ./bin/tunnel-console-mcp init --non-interactive --config /absolute/path/config.toml
```

导入配置中的 `env:` 凭据会解析并保存为文件引用，使服务在重启后仍能读取。`sudo` 可能清除调用终端的环境变量，因此非交互导入优先使用 `file:` 引用；不会把密钥加入 sudo 参数。开机服务不支持依赖登录会话的 `SSH_AUTH_SOCK`，应改用私钥文件或密码；直接 `run` 模式仍支持 SSH agent。

再次执行 `init` 默认复用已安装配置并更新程序，无需重输凭据；用 `--config` 导入新配置。旧安装代次保留，启动失败时会尝试恢复旧配置和服务，并返回失败；恢复也失败时会明确报告。并发执行两个 `init` 会被安装锁阻止。

`init` 显示成功表示系统服务管理器已完成注册、启用与启动，**不等于 OpenAI 隧道已经鉴权上线或 SSH 主机已经连通**。仍需在 ChatGPT 中实际调用工具完成在线验收。macOS 开启 FileVault 时，冷启动需要先解锁系统磁盘。

查看或重启服务：

```sh
# macOS
sudo launchctl print system/com.yanjunhui.tunnel-ssh-mcp
sudo launchctl kickstart -k system/com.yanjunhui.tunnel-ssh-mcp

# Linux
sudo systemctl status tunnel-ssh-mcp.service
sudo systemctl restart tunnel-ssh-mcp.service
```

停止并取消自动启动：

```sh
# macOS（保留配置和安装文件）
sudo launchctl disable system/com.yanjunhui.tunnel-ssh-mcp
sudo launchctl bootout system/com.yanjunhui.tunnel-ssh-mcp

# Linux（保留配置和安装文件）
sudo systemctl disable --now tunnel-ssh-mcp.service
```

不带 `--container` 的 `init` 用于原生系统服务。Docker 部署使用前面的独立初始化 Compose，它会调用 `init --container`，随后由正式 Compose 管理启动。

### 手工前台运行

```sh
make build
make install-tunnel
cp config.example.toml config.toml
```

编辑 `config.toml` 中 SSH 的 host、user、private_key、workdir，以及隧道 ID/凭据引用。路径相对配置文件所在目录解析，不展开 `~`。密码和密钥口令支持 `env:变量名` 或 `file:/绝对路径`；不接受直接写入 TOML 的明文密码。私钥文件支持 OpenSSH 格式；也可通过 `agent_socket = "env:SSH_AUTH_SOCK"` 使用 SSH agent。

```sh
./bin/tunnel-console-mcp check --config config.toml
./bin/tunnel-console-mcp doctor --config config.toml
./bin/tunnel-console-mcp run --config config.toml
```

`check` 仅验证配置结构，不验证凭据和远端连通性。`doctor` 运行官方诊断，会访问 OpenAI 隧道服务。`run` 将进程交给 tunnel-client，由官方进程监督 Go stdio 子进程；重启后内存中的会话会丢失。

无需提供新的 MCP 访问密码或 Token。SSH 密钥/密码仍是目标机 SSH 登录所需；它们与 MCP 入口鉴权不同。本版本按需求没有单独的 SSH 主机公钥信任策略，不验证目标主机指纹：OpenAI 隧道验证调用方身份，并不验证外部 SSH 主机身份。使用外部主机时应明确这个信任范围。

## 工具调用示例

```json
{"command":"pwd\nprintf 'Multiline script\\n'\nls -la","profile":"local-dev"}
```

持久终端依次调用：

```text
open-session         {"name":"work","type":"interactive"}
write-session-input  {"name":"work","data":"cd /workspace\nexport DEMO=hello\n"}
write-session-input  {"name":"work","data":"printf '%s\\n' \"$DEMO\"\n"}
read-session-output  {"name":"work","cursor":0}
```

后续读取使用上次返回的 `output.next_cursor`。`write-session-input` 只确认输入已发送，不表示输入中的命令已完成。交互输出包含终端提示符、回显和控制码。`run-command` 每次使用独立 shell，不继承命名终端状态。

```text
open-session       {"name":"build","type":"background","command":"make build"}
read-session-output {"name":"build","cursor":0}
sftp-upload         {"path":"hello.txt","content":"Hello"}
sftp-download-file  {"path":"artifact.tar","local_path":"/workspace/artifact-copy.tar"}
```

相对远端路径以配置的 `workdir` 为基准。不会自动创建目标文件的父目录。流式上传中断时远端文件可能已部分写入，错误会明确说明；本地下载采用临时文件并在成功后替换目标。

## 运行行为与边界

- 一个 tunnel ID 只启动一个 stdio tunnel-client；升级时先停止旧实例，再启动新实例。官方明确不支持同一隧道的多个 stdio 实例，详见[官方 stdio 部署约束](https://github.com/openai/tunnel-client/blob/v0.0.19/docs/configuration.md#stdio-deployment-limits)。
- 同一服务的调用方共享连接配置和命名会话，没有用户角色或租户隔离。不同账户需要独立部署和独立 tunnel ID。
- 每个命令默认 60 秒超时。`timeout_seconds: 0` 或配置 `command_timeout = "0s"` 可禁用。后台会话不使用短命令超时。
- `output_bytes` 默认 1 MiB，允许配置 1 字节至 64 MiB，0 使用默认值。它是每个输出缓存的内存界限，不是命令配额。缓存溢出保留尾部并报告 `dropped_bytes`。内联下载超过此值会指向无该限制的流式工具。
- 命令取消/超时会关闭其 SSH 连接。SSH 连接断开、TERM 请求发出和远端进程实际退出是不同事件；脱离终端的子进程可能继续运行。需要时用已知 PID 调用 `signal-process`。
- 目标命令和交互终端面向 POSIX shell；没有 Windows PowerShell/cmd 适配。
- 服务不重试已经开始的命令或上传，以免重复执行产生副作用。会话没有自动重建或隐式状态恢复。

## 开发和验证

```sh
make check   # 格式和双语文档检查
make test    # race + 真实本地 SSH/SFTP + PTY + MCP 内存及 stdio 客户端
make vet
make build
make linux   # Linux arm64 和 amd64
TUNNEL_SSH_MCP_IMAGE=go-tunnel-ssh-mcp:review \
TUNNEL_SSH_DEV_IMAGE=go-tunnel-ssh-dev:review \
make docker-build docker-test # 初始化容器、真实交互向导、持久化/权限及 OpenSSH/PTY/SFTP
```

自动化测试创建临时 SSH 主机和临时文件，不访问现有业务机器或线上隧道。0.3.0 容器检查覆盖 VNC 未启动时的原生截图及输入、双路径并用、停止 VNC 后继续原生操作。各版本测试范围、正式运行状态、真实 ChatGPT 证据和回退方式见[验证记录](docs/verification.zh-CN.md)和[部署记录](docs/live-deployment.zh-CN.md)。

依赖锁定在 `go.mod` / `go.sum`。官方隧道版本固定为 v0.0.19，安装器通过发布的 `SHA256SUMS.txt` 核对下载归档，构建缓存与运行凭据均排除在 Git/Docker 构建上下文之外。开发镜像使用官方 Go 1.26.10、Node 24 镜像中的工具链，其余软件使用 Debian 软件源版本；APT 和基础镜像标签未固定到完整快照，构建不宣称逐字节可重现。

参考接口文档：[官方 MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk/tree/v1.8.0)、[官方 OpenAI tunnel-client](https://github.com/openai/tunnel-client/tree/v0.0.19)。

## 文档与注释约定

英文主文档使用原文件名，完整简体中文版本使用 `.zh-CN.md` 后缀，双方互相链接。修改行为、字段或命令时同步更新两种语言；示例参数、默认值、版本、哈希和验收结论必须一致。源码注释以英文在前、中文在后，说明契约、边界和设计原因，避免重复代码。运行时向导和错误提示的语言不由文档翻译改变。新增文件头保留 `Created by Yanjunhui`。
