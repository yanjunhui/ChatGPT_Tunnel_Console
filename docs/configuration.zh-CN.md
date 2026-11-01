<!-- Created by Yanjunhui -->
# 配置与命令行参考

[English](configuration.md) | 简体中文 | [README](../README.zh-CN.md)

本文描述仓库 0.2.0 的实际实现，对应代码为 `internal/config`、`internal/setup`、`internal/tunnel` 和 `cmd/tunnel-console-mcp`。示例只使用占位值，不包含真实凭据。

## TOML 配置

`check`、`serve`、`run`、`doctor` 均接受 `--config PATH`，默认 `config.toml`。未知 TOML 字段会被拒绝。相对凭据、私钥和二进制路径以配置文件所在目录为基准，不以调用 shell 的目录为基准；不含路径分隔符的二进制名称通过 PATH 查找。TOML 路径不展开 `~`；交互私钥向导会单独按所选账户展开开头的 `~/`。

| 顶层字段 | 默认值／要求 | 含义 |
|---|---|---|
| `default_profile` | 只有一个配置时自动选择，否则必填 | 必须指向已存在的配置 |
| `output_bytes` | 1048576；`0` 采用该默认值 | 每份输出的保留字节数及内联下载原始字节上限；显式范围为 1–67108864 |
| `[profiles.NAME]` | 至少一个 | 命名 SSH 目标；会话名称也按配置划分 |
| `[tunnel]` | `run`、`doctor` 和初始化需要 | 官方监督进程配置；本地 `serve` 不使用隧道 |

| 隧道字段 | 默认值 | 含义 |
|---|---|---|
| `binary` | `tunnel-client` | PATH 名称或文件路径；Docker init 使用 `/usr/local/bin/tunnel-client` |
| `id` | 缺省时使用 `CONTROL_PLANE_TUNNEL_ID` | 启动或初始化隧道时必须提供 |
| `api_key` | 缺省时使用 `env:CONTROL_PLANE_API_KEY` | 凭据引用；密钥内容不进入命令行参数 |
| `health_addr` | `127.0.0.1:0` | 官方诊断接口的回环 IP 和端口；`0` 自动分配可用端口；Docker 导入模板使用 8080 |

| SSH 配置字段 | 默认值／要求 | 含义 |
|---|---|---|
| `host`、`user` | 不得为空白 | SSH 主机和账户 |
| `port` | 22；`0` 采用 22 | 显式范围为 1–65535 |
| `private_key` | 可选路径 | SSH 私钥，初始化时复制 |
| `passphrase` | 可选凭据引用 | 加密私钥的口令 |
| `password` | 可选凭据引用 | SSH 密码 |
| `agent_socket` | 可选路径或 `env:NAME` | 手工运行支持；原生和容器初始化均拒绝依赖 agent 的配置 |
| `workdir` | 空 | 命令／会话目录，以及远端相对文件路径基准；空值保留目标默认目录 |
| `connect_timeout` | `15s` | 非负 Go duration；`0s` 禁用配置层连接期限，但请求取消仍生效 |
| `command_timeout` | `60s` | 独立命令的非负 Go duration；`0s` 禁用期限；持久会话不使用此值 |

`private_key`、`password`、`agent_socket` 至少提供一项，可同时配置多种认证方式。当前实现不校验目标主机公钥；隧道的调用方认证不能证明外部 SSH 主机身份。

### 可选 VNC 配置

`[profiles.NAME.vnc]` 仅对该 SSH 目标启用桌面；省略时完全关闭，已有配置继续有效。

| VNC 字段 | 默认值／要求 | 含义 |
|---|---|---|
| `enabled` | false | 显式允许桌面工具 |
| `host` | `127.0.0.1` | 从 SSH 目标访问的主机名或 IP，不含端口 |
| `port` | 5900；`0` 采用 5900 | 范围 1–65535；自带桌面使用 5901 |
| `password` | 空 | 空值选择 None；否则只允许 `env:NAME` 或 `file:PATH` |
| `timeout` | `15s` | 正值且不超过 `120s`，覆盖排队、连接与操作 |

`check` 验证字段和引用格式，不读取密码或探测 VNC。初始化持久化密码引用，连接时再验证传统 VNC 密码不超过八个可打印 ASCII 字节。完整设置、认证范围和操作边界见 [VNC 指南](vnc.zh-CN.md)。

### 凭据引用

- `env:NAME` 读取环境变量。
- `file:PATH` 读取文件，去除末尾 CR/LF 字节，保留其他空白。
- 未提供的可选引用解析为空；已提供但来源为空或缺失时失败。解析凭据时拒绝直接明文。
- `check` 不解析凭据，仅检查结构和默认值。`run`／`doctor` 先解析隧道凭据，再把引用交给监督进程；SSH 凭据在建立连接时读取。
- 初始化将引用的凭据保存为 0600 文件，引用固定到具体代次；在本地校验导入私钥格式和口令，不尝试 SSH 登录。
- Docker 精确匹配内置 `dev`／`developer` 配置时，可引用尚未存在的 `/var/lib/tunnel-ssh/client_key`：开发容器首次启动时生成。其他导入私钥必须存在。

原生或手工运行使用 [config.example.toml](../config.example.toml)，Docker 导入使用 [deploy/config.init.toml](../deploy/config.init.toml)。`deploy/config.docker.toml` 是本地回归测试夹具，不是正式初始化模板。

## 命令

| 命令 | 作用与校验边界 |
|---|---|
| `version` | 输出 MCP 应用版本 |
| `check --config PATH` | 解析、填充默认值并校验结构，不解析凭据或建立连接 |
| `serve --config PATH` | 通过 stdio 提供 MCP，不认证调用者或启动隧道 |
| `run --config PATH` | 用官方监督进程替换当前进程，并把当前程序作为 stdio 工作进程启动 |
| `doctor --config PATH` | 使用 `--explain` 执行官方诊断；诊断成功不等于端到端工具调用成功 |
| `install-tunnel --dir DIR --version VERSION` | 下载并校验官方发布包；默认 `bin`、`v0.0.19`；支持 macOS/Linux amd64/arm64 |
| `init` | 准备原生安装，注册、启用并启动系统服务 |
| `init --container` | 发布容器持久化配置，不启动服务或安装宿主机二进制 |

所有带参数的子命令均接受 `--help`，成功退出，不加载配置、不安装服务、不访问网络；未知参数仍返回错误。

### 初始化参数

| 参数 | 默认值 | 适用范围／行为 |
|---|---|---|
| `--config PATH` | 空 | 导入 TOML；未指定时复用当前配置，首次则打开向导 |
| `--non-interactive` | False | 必须有导入或现存配置，不打开向导 |
| `--service-user USER` | sudo 调用者或当前账户 | 仅原生模式，要求已有本机用户 |
| `--container` | False | 使用容器初始化模式 |
| `--output DIR` | `/etc/tunnel-ssh` | 仅容器模式，必须是与运行容器共享且不是 `/` 的专用目录 |
| `--owner-uid`、`--owner-gid` | 均为 1000 | 仅容器模式，范围 0–2147483647 |
| `--reconfigure` | False | 仅容器模式，重新打开向导；不能与 `--config` 或 `--non-interactive` 同用 |

Docker 初始化以 root 运行，为非 root 正式账户设置归属。原生安装先提示系统路径，必要时通过 sudo 提权。

原生代次位于 `/usr/local/lib/tunnel-ssh-mcp/releases/`，容器代次位于配置目录的 `generations/`；`current` 选择当前代次，准备完成后才发布。原生激活先停止旧服务，失败时尝试恢复。容器 init 只发布配置，需显式重启 MCP 加载。两种成功提示均不代表隧道或 SSH 已连通。

## Docker 变量

`.env.example` 只包含宿主机路径和镜像标签，不包含凭据：

| 变量 | 默认值 | 挂载／用途 |
|---|---|---|
| `TUNNEL_SSH_CONFIG_DIR` | `./data` | 初始化及运行容器内的 `/etc/tunnel-ssh`，运行时只读 |
| `TUNNEL_SSH_INIT_INPUT_DIR` | `./init-input` | 仅初始化容器内只读的 `/input` |
| `TUNNEL_SSH_WORKSPACE_DIR` | `./workspace` | MCP 及开发容器内的 `/workspace` |
| `TUNNEL_SSH_MCP_IMAGE` | `go-tunnel-ssh-mcp:latest` | 初始化和 MCP 运行共用的镜像标签 |
| `TUNNEL_SSH_DEV_IMAGE` | `go-tunnel-ssh-dev:latest` | 开发镜像标签 |
| `TUNNEL_CONSOLE_DESKTOP_IMAGE` | `go-tunnel-console-desktop:latest` | 可选桌面叠加镜像，替代开发镜像 |

初始化与运行须使用相同配置挂载。备份 `data/` 时保留权限和符号链接。`developer_home` 与 `ssh_identity` 是持久化具名卷，`docker compose down -v` 会删除它们。

## 如何判断就绪

标准 Docker 配置下检查现存监督进程，不另起同一隧道实例：

```sh
docker compose exec -T mcp tunnel-console-mcp version
docker compose exec -T mcp tunnel-client health --port 8080 --require-control-plane-poll --json
```

端口须与配置匹配。ready 状态可能早于首次控制面成功轮询，因此应要求轮询检查并查看启动日志。在运行容器内直接执行 `doctor` 可能与已占用的诊断端口冲突；部署记录使用独立一次性诊断容器。最后仍需在 ChatGPT 中实际调用 MCP 工具；报告中分别记录本地 stdio、隧道健康和端到端结果。
