<!-- Created by Yanjunhui -->
# 安装与升级

[English](installation.md) | 简体中文 | [文档中心](README.zh-CN.md)

需要自带 Linux 工作区时选择 Docker；连接已有 SSH 主机时，也可在 macOS/Linux 上安装原生服务。两种方式均通过出站连接接入 OpenAI Tunnel，需要准备隧道 ID 和运行时 API Key。

## 准备 OpenAI 隧道

1. 打开 [OpenAI Platform 隧道设置](https://platform.openai.com/settings/organization/tunnels)，选择与 ChatGPT 工作区或个人账号关联的组织。
2. 创建隧道。创建需要 Read + Manage 权限；本服务使用的运行密钥只需该隧道的 Read + Use 权限。
3. 保存隧道 ID 和运行密钥，初始化时使用。凭据应保存在 Git 之外，通过向导或凭据文件引用导入。

服务所在主机需要通过 HTTPS 访问 OpenAI Tunnel，并能够连接 SSH 目标。Docker 构建和原生隧道客户端安装还需访问软件源与发行包下载地址，无需开放入站 MCP 端口。账号条件和权限说明见 [OpenAI Tunnel 设置指南](https://developers.openai.com/api/docs/guides/secure-mcp-tunnels)。

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

4. 按[连接 ChatGPT](#连接-chatgpt)完成设置，然后要求运行 `id -un; pwd; go version`。内置环境应返回 `developer`、`/workspace` 和 Go 版本。

`dev` 容器包含 Go、Python、Node.js/npm、Git、C/C++、CMake、Ninja、tmux、rg、fd-find、jq、tree。`developer` 在容器内可使用无密码 sudo。容器内权限不等于宿主机权限；对 `/workspace` 的操作会修改项目下 `workspace/` 的同一批文件。

`mcp` 容器仅运行 Go MCP 和官方隧道组件。两个容器之间使用自动生成的 SSH 密钥，SSH 和 MCP 都没有宿主机端口映射。`workspace/` 是共享工作目录，也可作为 MCP 的流式文件传输目录。SSH 身份和开发用户主目录保存在具名卷中。

`compose.init.yaml` 独立于运行时 Compose，不依赖已有配置、`.env` 凭据或运行中的开发容器。它以容器内 root 写入配置并设置运行账户 UID/GID 1000；运行时 `mcp` 使用该非 root 账户，只读挂载配置。初始化容器不注册宿主机 systemd/launchd，不需要 Docker socket，也不启动 OpenAI 隧道。

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

可通过 `TUNNEL_SSH_MCP_IMAGE` 和 `TUNNEL_SSH_DEV_IMAGE` 指定自定义镜像标签；默认仍为 `go-tunnel-ssh-mcp:latest` 和 `go-tunnel-ssh-dev:latest`。初始化与运行须使用相同 MCP 标签。

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

将 [导入模板](../deploy/config.init.toml) 复制到 `init-input/config.toml`，修改隧道 ID，并将运行时 API Key 保存到 `init-input/openai-tunnel-key`，文件只包含密钥。模板默认连接自带开发环境，也可按 [原生模板](../config.example.toml) 配置多个外部 SSH profile；凭据使用 `/input/` 下的文件引用。

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

安装后按[连接 ChatGPT](#连接-chatgpt)完成设置，并执行一条命令，确认隧道和 SSH 连通。macOS 开启 FileVault 时，冷启动需要先解锁系统磁盘。

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

不带 `--container` 的 `init` 用于原生系统服务。Docker 部署使用前面的独立初始化 Compose，它会调用 `init --container`，随后由运行时 Compose 管理启动。

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

OpenAI Tunnel 验证 MCP 调用方身份，SSH 凭据用于登录目标主机。当前服务不校验 SSH 主机公钥，连接外部主机前请阅读[安全说明](../SECURITY.zh-CN.md)。

## 连接 ChatGPT

1. 在 ChatGPT 中打开 **Plugins**，选择 **+ → Add custom MCP server**，连接方式选择 **Tunnel**。
2. 选择或输入初始化时配置的隧道 ID。MCP 认证方式选择 **No authentication**，调用方身份由 OpenAI Tunnel 验证。
3. 创建并安装插件，在对话中启用。如果 Codex 账号可以访问已连接的插件，也可在 Codex 中启用。
4. 要求列出 SSH 连接，再在所选连接上运行 `pwd`。`list-connections` 显示配置，执行命令用于确认 SSH 可用。

隧道须属于目标账号或工作区关联的 Platform 组织。找不到隧道或插件时，先检查账号关联。工具发现和连接问题见[故障排查](troubleshooting.zh-CN.md)。

## 备份与升级

升级前记录当前部署的 Git 提交，保留旧镜像或程序，并备份配置和用户数据。Docker 部署需要保留整个 `data/`、`workspace/`，以及 `developer_home` 和 `ssh_identity` 卷。所用 Compose 叠加文件和本地 `.env` 也应纳入私有备份。备份须保留文件归属、权限和符号链接；配置备份包含凭据。

### Docker

在现有检出目录停止 MCP，更新到所需版本，再构建和重建服务。默认命令行环境使用：

```sh
docker compose stop mcp
git pull --ff-only
docker compose up -d --build
```

桌面部署的管理命令须继续带上相同的桌面／VNC 叠加文件，或在 `COMPOSE_FILE` 中保留它们。保持检出路径、Compose 项目名和卷标识稳定，继续使用原有数据。仅升级源码不需要重新初始化。修改配置见[重新初始化和外部 SSH](#重新初始化和外部-ssh)；需要保留多个连接时，导入完整 TOML。

### 原生服务

更新源码、重新构建并再次运行安装器：

```sh
git pull --ff-only
make build
./bin/tunnel-console-mcp init
```

安装器复用已有配置并保留旧安装代次。服务使用自定义账户时，继续传入 `--service-user USER`。升级后查看服务日志，并通过插件执行一条命令。

### 工具发现与兼容性

升级涉及工具或参数变化时，在已安装插件的管理页刷新工具。核对对话实际使用的插件和账号；刷新浏览器不会更新另一个账号的工具目录。操作说明见[刷新工具元数据](https://developers.openai.com/plugins/deploy/connect-chatgpt#refresh-metadata)。

主命令为 `tunnel-console-mcp`，`tunnel-ssh-mcp` 保留为别名。原生服务路径、Compose 项目名 `go-tunnel-ssh`、持久卷名称和 `TUNNEL_SSH_*` 变量沿用原名称，以兼容已有安装。现有检出目录无需更名。桌面和 VNC 均需显式配置启用，更新程序不会自动开启。

同一个隧道 ID 只运行一个 stdio 隧道客户端，回退时也一样。先停止新实例，再恢复保存的源码版本并构建，或恢复旧镜像／程序和配置。重启 MCP 会清空内存会话。版本变化见[更新记录](../CHANGELOG.zh-CN.md)，服务或工具列表异常时参阅[故障排查](troubleshooting.zh-CN.md)。
