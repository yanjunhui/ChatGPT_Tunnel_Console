<!-- Created by Yanjunhui -->
# 通过 MCP 操作 VNC 桌面

[English](vnc.md) | 简体中文 | [README](../README.zh-CN.md)

0.2.0 为现有 SSH/SFTP 控制台增加桌面截图和输入。链路为 **OpenAI MCP 隧道 → Go MCP 服务 → SSH direct-tcpip → VNC 服务**。官方隧道传输包含 PNG 图片内容的 MCP 消息，不是原生 VNC/TCP 隧道；没有网页查看器或公开 VNC 监听端口。

## 自带 Docker 桌面

新部署运行初始化向导，选择 **3：自带图形桌面**：

```sh
mkdir -p data init-input workspace
docker compose -f compose.init.yaml run --rm --build init
docker compose -f compose.yaml -f compose.desktop.yaml up -d --build
```

可选叠加配置只把开发镜像换成包含 TigerVNC/Openbox 和 xterm 的桌面镜像，继续使用 SSH 身份、开发用户主目录和工作区卷。桌面为 1280×800，显示编号 `:1`，VNC 监听目标容器回环接口的 5901 端口，不发布宿主机端口。桌面健康检查同时验证 SSH 和 X 显示服务。后续构建／启动桌面时使用两个 Compose 文件；已有容器的停止、启动和日志操作可使用基础文件。

已有部署应先备份完整配置和持久卷，停止 MCP，再以 `--reconfigure` 运行初始化向导，选择 3 后用叠加配置启动。重新配置会替换连接选择；若需要保留已有多配置，应导入完整 TOML。初始化本身既不启动也不测试 VNC。同一个隧道 ID 不要运行两个监督进程。原路径、Compose 项目标识和环境变量保留旧名称，见[升级兼容说明](../README.zh-CN.md#更名与升级兼容)。

## 已有 VNC 服务

在导入配置中，为相应 SSH profile 增加：

```toml
[profiles.server.vnc]
enabled = true
host = "127.0.0.1"
port = 5900
password = "file:/absolute/path/to/vnc-password"
timeout = "15s"
```

`host` 从 **SSH 目标主机** 解析和访问，不是从 ChatGPT 或 MCP 容器访问。自带桌面使用端口 5901，并省略 `password`。Docker 导入时，将密码文件放在 `init-input/`，引用 `/input/...`；init 会复制到持久化私有代次。按 SSH 配置相同的初始化方式导入，重启 MCP 后生效。原生交互向导只配置 SSH，外部 VNC 使用 TOML 导入。

SSH 服务须允许到配置地址的 `direct-tcpip` 转发，VNC 服务须已运行并允许共享客户端连接。协议支持 RFB 3.3、3.7、3.8，以及 Raw 像素和 DesktopSize 更新。认证支持 None（没有密码引用）和传统 VncAuth（有密码引用）；未实现 TLS/VeNCrypt、Apple 专用认证或厂商扩展。有密码配置时绝不退回 None。

传统 VNC 密码限制为最多八个可打印 ASCII 字节，过长或非 ASCII 会报错，不静默截断。VNC 认证不提供传输加密；应将监听保持在回环接口并使用 SSH 传输。若 VNC 位于另一台局域网主机，本实现不加密 SSH 主机到 VNC 的网络段。现有 SSH 主机公钥验证限制仍适用，见[安全说明](../SECURITY.zh-CN.md)。

## 截图与输入流程

下面每行是一次独立 MCP 调用：

```text
list-connections {}
vnc-screenshot   {"profile":"local-dev"}
vnc-pointer      {"profile":"local-dev","events":[{"x":100,"y":100,"buttons":1}]}
vnc-key          {"profile":"local-dev","keys":["Control_L","l"]}
vnc-type         {"profile":"local-dev","text":"hello"}
vnc-key          {"profile":"local-dev","keys":["Return"]}
vnc-screenshot   {"profile":"local-dev"}
```

`vnc-screenshot` 的 `content[0]` 为 MCP 图片（`image/png`，JSON 传输时使用 base64），`content[1].text` 为元数据 JSON，`structuredContent` 同样包含 `width`、`height`、`name`。客户端须支持 MCP 图片显示；结果不是本地截图路径，也不是纯文本 base64。坐标基于原始图片尺寸，左上角为原点；即使客户端缩放了预览，也应使用原图像素坐标。

鼠标按钮使用位掩码：左键 1、中键 2、右键 4、向上滚动 8、向下滚动 16；0 释放全部按钮。一批事件表示一次手势，结束时自动释放。例如：

```text
vnc-pointer {"events":[{"x":100,"y":100,"buttons":0}]}
vnc-pointer {"events":[{"x":100,"y":100,"buttons":1},{"x":300,"y":250,"buttons":1}]}
vnc-pointer {"events":[{"x":300,"y":250,"buttons":16},{"x":300,"y":250,"buttons":0}]}
```

`vnc-key` 接受不区分大小写的 X11 风格键名：Control_L/Control_R（ctrl）、Shift_L/Shift_R（shift）、Alt_L/Alt_R（alt）、Super_L/Super_R（super）、Return（enter）、Tab、Escape（esc）、Backspace、Delete、Home、End、Insert、Left/Up/Right/Down、Page_Up/Page_Down、Space 和 F1–F12；也接受单个可打印 Unicode 字符。按数组顺序按下、逆序释放，不能跨调用保持按键。

`vnc-type` 使用按下／释放事件，不通过剪贴板粘贴。CRLF 发送一次 Return，CR/LF 和 Tab 发送对应按键；换行可能执行终端命令或提交表单。目标键盘布局、VNC 实现、应用和输入法会影响 Unicode 输入，应检查实际可见结果，不能假设所有 Unicode 字符都能正确呈现。

输入成功返回含 `desktop` 元数据和 `message` 的文本 JSON，只确认输入／释放事件已写入。应用可能仍在处理，焦点可能不同，其他桌面客户端也可同时操作；应重新截图核对预期结果。网络失败或取消可能发生在部分输入之后，不要盲目重试。

## 边界、生命周期和排错

| 边界 | 数值／行为 |
|---|---|
| 启用 | 配置段缺失或 `enabled = false` 时拒绝桌面工具，SSH/SFTP 仍可使用 |
| 操作超时 | 默认 15 秒，必须为正且不超过 120 秒；包括等待、SSH/VNC 建连和操作 |
| 截图 | 单边最多 8192 像素，总像素最多 8,388,608；PNG 最多 16 MiB，须完整覆盖全帧 |
| 鼠标批次 | 1–256 个事件，发送前检查全部坐标位于当前桌面内，按钮掩码为 0–31 |
| 组合键 | 1–16 个键 |
| 文本 | 1–4096 个 Unicode 字符，拒绝非法 UTF-8 及不支持的控制字符 |
| 连接 | 每次调用新建共享模式 SSH/VNC 连接，成功／失败／取消后关闭 |
| 并发 | 单个 MCP 进程内串行桌面操作，不锁定其他桌面客户端 |

截图边界用于限制服务端内存，客户端／隧道消息限制可能更低，尤其 base64 会增加传输大小；客户端拒绝大截图时，使用较小桌面。没有视频流、图片缩放、剪贴板同步、声音、浏览器界面或跨调用按键保持；截图不包含本地查看器绘制的光标。

VNC 未启用时检查当前配置并重启 MCP。SSH 转发失败时检查 SSH 的 `AllowTcpForwarding`、目标地址和 VNC 监听服务。认证不支持时，使用受保护回环地址上的 None 或 SSH 内的 VncAuth。鼠标坐标越界时重新截图；桌面尺寸可能在两次调用间改变。认证和帧错误返回 MCP `isError: true`，不返回成功图片数据。

本地验收使用真实 TigerVNC、OpenSSH、xterm 和 MCP stdio 客户端，验证截图解码、真实鼠标位置、键盘效果、认证、清理和无公开端口。这本身不证明真实 ChatGPT 应用已经发现或显示新工具；还需升级目标部署后实际经隧道调用。

协议参考：[RFB 规范](https://github.com/rfbproto/rfbproto/blob/master/rfbproto.rst)、[TigerVNC 服务端](https://tigervnc.org/doc/Xvnc.html)、[官方隧道配置](https://github.com/openai/tunnel-client/blob/v0.0.19/docs/configuration.md)。
