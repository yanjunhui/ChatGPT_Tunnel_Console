// Created by Yanjunhui
package service

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

const searchPath = "/usr/local/bin:/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin"

// Render builds a launchd plist or systemd unit. Secrets are never embedded.
// Unsupported platforms, empty values and NUL/newline characters return errors.
// 中文：Render 生成不含密钥的 launchd 或 systemd 定义；
// 平台不支持、值为空或含 NUL/换行时返回错误。
func Render(platform string, p Paths, username string) ([]byte, error) {
	for _, value := range []string{p.Install, p.State, p.Log, p.Definition, username} {
		if value == "" || strings.ContainsAny(value, "\x00\r\n") {
			return nil, fmt.Errorf("服务路径和账户不得为空或包含控制字符")
		}
	}
	if platform == "darwin" {
		return []byte(plist(p, username)), nil
	}
	if platform == "linux" {
		return []byte(unit(p, username)), nil
	}
	return nil, fmt.Errorf("不支持的服务平台 %q", platform)
}

func xmlText(s string) string {
	var b bytes.Buffer
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

func plist(p Paths, user string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!-- Created by Yanjunhui -->
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict>
<key>Label</key><string>%s</string>
<key>UserName</key><string>%s</string>
<key>ProgramArguments</key><array><string>%s</string><string>run</string><string>--config</string><string>%s</string></array>
<key>WorkingDirectory</key><string>%s</string>
<key>EnvironmentVariables</key><dict><key>HOME</key><string>%s</string><key>USER</key><string>%s</string><key>PATH</key><string>%s</string></dict>
<key>RunAtLoad</key><true/><key>KeepAlive</key><true/>
<key>ThrottleInterval</key><integer>10</integer>
<key>ExitTimeOut</key><integer>20</integer>
<key>StandardOutPath</key><string>%s</string>
<key>StandardErrorPath</key><string>%s</string>
</dict></plist>
`, Label, xmlText(user), xmlText(filepath.Join(p.Current(), "tunnel-ssh-mcp")), xmlText(p.Config()), xmlText(p.State), xmlText(p.State), xmlText(user), searchPath, xmlText(filepath.Join(p.Log, "stdout.log")), xmlText(filepath.Join(p.Log, "stderr.log")))
}

func systemdQuote(s string) string {
	// systemd expands percent specifiers even inside quoted directive values.
	// 中文：systemd 即使在引号内也会展开百分号指示符，因此需要独立转义。
	r := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "%", "%%", "\t", `\t`)
	return `"` + r.Replace(s) + `"`
}

func execQuote(s string) string { return systemdQuote(strings.ReplaceAll(s, "$", "$$")) }

func unit(p Paths, user string) string {
	return fmt.Sprintf(`# Created by Yanjunhui
[Unit]
Description=OpenAI Tunnel SSH MCP
Wants=network-online.target
After=network-online.target

[Service]
Type=exec
User=%s
WorkingDirectory=%s
Environment=%s
Environment=%s
Environment=%s
ExecStart=%s run --config %s
Restart=always
RestartSec=10
TimeoutStopSec=20
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
`, strings.ReplaceAll(user, "%", "%%"), strings.ReplaceAll(p.State, "%", "%%"), systemdQuote("HOME="+p.State), systemdQuote("USER="+user), systemdQuote("PATH="+searchPath), execQuote(filepath.Join(p.Current(), "tunnel-ssh-mcp")), execQuote(p.Config()))
}
