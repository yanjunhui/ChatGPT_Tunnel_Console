<!-- Created by Yanjunhui -->
# 原生 Linux 桌面

[English](desktop.md) | 简体中文 | [文档中心](README.zh-CN.md)

使用 `desktop-*` 工具，在 ChatGPT 或 Codex 中截取 Linux X11 桌面并操作鼠标和键盘。原生后端通过 SSH 工作，无需 VNC 服务或独立查看器。可选 `vnc-*` 工具可连接同一桌面或另一个兼容的 VNC 服务。

| 能力 | 原生桌面 | 可选 VNC |
|---|---|---|
| 传输 | MCP 隧道 → Go → SSH → xwd/xdotool | MCP 隧道 → Go → SSH 转发 → VNC |
| 配置 | `profiles.NAME.desktop` | `profiles.NAME.vnc` |
| 目标要求 | 可访问的 Linux X11 显示服务、xwd、xdotool、C.UTF-8 | 已运行且兼容的 RFB 服务 |
| 自带服务 | `dev`：Xvfb、Openbox、xterm 和 SSH | 独立 `vnc`：x11vnc |
| 另一后端关闭／停用 | 继续运行 | 显示服务可用时继续运行 |

两条路径可以共享同一桌面并互补。独立指后端配置、协议和服务生命周期，而不是两个桌面会话：它们仍然需要 SSH，自带 VNC 还需要 `dev` 中的 X 显示服务。停止 `dev` 会使两条路径都无法访问该桌面；禁用原生工具配置不会停止 Xvfb 或 VNC。单个 MCP 进程内共用操作队列，但不锁定外部查看器。

## 仅启动原生桌面

运行初始化向导，选择 **3**，随后可选 VNC 回答 **n**（默认）：

```sh
mkdir -p data init-input workspace
docker compose -f compose.init.yaml run --rm --build init
docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

该镜像未安装 VNC 服务。它创建 1280×800、24 位深度的 Xvfb `:1` 显示服务，以及 Openbox 窗口管理器和 xterm 终端；X 服务不监听 TCP。原生截图经 SSH exec 调用 `xwd`，由 Go 校验并编码原始帧，不缩放。输入由 `xdotool` 通过 XTEST 发送，文本经 stdin 传输并使用 UTF-8 locale，不进入进程参数或剪贴板。此后端面向 Linux X11，不支持原生 Wayland、macOS 或 Windows 桌面截图。

已有部署先备份配置、工作区和持久卷，停止 MCP 后导入完整配置，保留已有连接与凭据。相关新增段为：

```toml
[profiles.local-dev.desktop]
enabled = true
display = ":1"
timeout = "15s"
```

使用 `compose.init.yaml` 加 `--non-interactive --config /input/config.toml` 导入，再用桌面叠加配置启动。已有配置不会隐式启用任一后端。外部 Linux 目标需安装工具，并确保 SSH 账户已有所选 X 显示服务及其认证环境的访问权限；保留显示访问控制，仅向目标账户授予访问权限。

## 添加或停止补充 VNC

初始化向导的可选 VNC 问题选择 **y**，或额外导入：

```toml
[profiles.local-dev.vnc]
enabled = true
host = "127.0.0.1"
port = 5901
timeout = "15s"
```

使用三个配置文件管理同一部署：

```sh
docker compose -f compose.yaml -f compose.desktop.yaml -f compose.vnc.yaml up -d --build
docker compose -f compose.yaml -f compose.desktop.yaml -f compose.vnc.yaml stop vnc
docker compose -f compose.yaml -f compose.desktop.yaml -f compose.vnc.yaml start vnc
```

停止 `vnc` 不会停止 SSH、X 显示服务和原生工具。其健康检查独立，`dev` 和 `mcp` 均不依赖 VNC 就绪。可选服务共享 X11 socket 和 `dev` 网络命名空间，监听回环 5901，不映射宿主机端口。重建 `dev` 时，使用带三个文件的 `up -d`，使 Compose 同时更新 VNC 网络关联。标准 VNC 查看器可经显式配置的 SSH 转发连接，但 MCP 操作不需要查看器。详见 [VNC 认证与协议限制](vnc.zh-CN.md)。

为避免后续漏写叠加配置，macOS/Linux 的本地 `.env` 可选写入 `COMPOSE_FILE=compose.yaml:compose.desktop.yaml:compose.vnc.yaml`。初始化与运行保持相同镜像标签，相关变量见 `.env.example`。桌面 socket 卷属于运行状态，不是用户数据；应保存 `developer_home`、`ssh_identity`、`data/` 和 `workspace/`，需要保留数据时不得使用 `down -v`。

## 通过 MCP 操作

每行是一次调用，先查看截图，再按原图像素选择坐标：

```text
list-connections    {}
desktop-screenshot  {"profile":"local-dev"}
desktop-pointer     {"profile":"local-dev","events":[{"x":100,"y":100,"buttons":1}]}
desktop-key         {"profile":"local-dev","keys":["Control_L","u"]}
desktop-type        {"profile":"local-dev","text":"printf 'DESKTOP_OK\\n'"}
desktop-key         {"profile":"local-dev","keys":["Return"]}
desktop-screenshot  {"profile":"local-dev"}
```

`list-connections` 分别返回 `desktop_enabled` 和 `vnc_enabled`；两者是配置开关，不是实时探测结果。截图返回 MCP PNG 图片及文本 JSON 形式的宽、高、名称；输入仅确认事件送达，应重新截图核对应用效果，焦点错误可能把输入发送到其他位置。主动选择另一条路径时，用相同参数结构调用 `vnc-*`。仅操作任务已经授权的桌面与应用。

两条路径均限制鼠标事件 1–256、组合键 1–16、Unicode 字符 1–4096、截图单边 8192 像素、总像素 8,388,608、PNG 16 MiB，默认操作期限 15 秒（最多 120 秒）。键名、按钮位及错误语义见[工具参考](tool-reference.zh-CN.md)。原生截图要求 24 位 TrueColor 或恒等映射 DirectColor、24/32 位存储，非法或截断帧报错，不返回部分图片。Unicode 呈现仍受目标应用和输入法影响。

升级后缺少工具时，参阅[工具发现排查](troubleshooting.zh-CN.md#工具或参数仍然是旧版)。

原生输入对 ASCII 使用 12 毫秒间隔；调用包含非 ASCII 文本或按键时使用 100 毫秒，为应用处理临时 Unicode 映射留出时间。此举改善应用短暂延迟时的送达，不保证无响应应用能接收输入；应验证实际结果，不盲目重试。非 ASCII 文本应拆为短段（默认期限建议每段约 100 字），或增大 `desktop.timeout` 至最多 120 秒。4096 字符是参数上限，不保证单次调用在期限内完成。
