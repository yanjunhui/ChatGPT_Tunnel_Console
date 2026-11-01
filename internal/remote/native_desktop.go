// Created by Yanjunhui
package remote

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"golang.org/x/crypto/ssh"
)

// DesktopScreenshot captures an X11 framebuffer over SSH without a VNC service.
// Returns a native PNG or configuration, transport, process or decoding errors.
// 中文：DesktopScreenshot 经 SSH 直接获取 X11 原尺寸 PNG，不依赖 VNC；返回配置、传输、进程或解码错误。
func (m *Manager) DesktopScreenshot(ctx context.Context, in DesktopInput) ([]byte, desktop.Info, error) {
	var data []byte
	var info desktop.Info
	err := m.withX11(ctx, in.Profile, func(client *ssh.Client, display string) error {
		return runX11(client, display, "xwd -silent -root", "", func(reader io.Reader) error {
			var err error
			data, info, err = desktop.XWD(reader)
			return err
		})
	})
	return data, info, err
}

// DesktopPointer sends one validated gesture directly through X11 and releases it.
// Failure may follow partial input; callers must inspect a new screenshot.
// 中文：DesktopPointer 通过 X11 发送已校验手势并释放；失败可能发生在部分输入后，应重新截图。
func (m *Manager) DesktopPointer(ctx context.Context, in PointerInput) (InputResult, error) {
	if err := desktop.ValidatePointer(in.Events); err != nil {
		return InputResult{}, err
	}
	return m.x11Input(ctx, in.Profile, "", func(info desktop.Info) (string, error) {
		for _, event := range in.Events {
			if event.X >= info.Width || event.Y >= info.Height {
				return "", fmt.Errorf("pointer outside current desktop; take a new screenshot")
			}
		}
		return pointerCommand(in.Events), nil
	})
}

// DesktopKey sends and releases a validated chord through X11 without using VNC.
// 中文：DesktopKey 经 X11 按下并释放已校验组合键，不使用 VNC。
func (m *Manager) DesktopKey(ctx context.Context, in KeyInput) (InputResult, error) {
	symbols, err := desktop.ValidateKeys(in.Keys)
	if err != nil {
		return InputResult{}, err
	}
	names := make([]string, len(symbols))
	for i, symbol := range symbols {
		names[i] = fmt.Sprintf("0x%x", symbol)
	}
	command := "xdotool key --clearmodifiers --delay 12 " + shellQuote(strings.Join(names, "+"))
	return m.x11Input(ctx, in.Profile, "", func(desktop.Info) (string, error) { return command, nil })
}

// DesktopType passes validated UTF-8 on stdin, keeping text out of process arguments.
// CRLF/CR become newline; no clipboard is read or modified.
// 中文：DesktopType 通过 stdin 发送已校验 UTF-8，避免文本进入进程参数；CRLF/CR 转为换行，不使用剪贴板。
func (m *Manager) DesktopType(ctx context.Context, in TextInput) (InputResult, error) {
	if err := desktop.ValidateText(in.Text); err != nil {
		return InputResult{}, err
	}
	text := strings.ReplaceAll(strings.ReplaceAll(in.Text, "\r\n", "\n"), "\r", "\n")
	// UTF-8 and normal key pacing let X11 clients consume temporary Unicode mappings.
	// 中文：UTF-8 和正常按键间隔让 X11 应用及时读取临时 Unicode 键位映射。
	return m.x11Input(ctx, in.Profile, text, func(desktop.Info) (string, error) {
		return "LC_ALL=C.UTF-8 xdotool type --clearmodifiers --delay 12 --file -", nil
	})
}

func (m *Manager) withX11(ctx context.Context, profile string, action func(*ssh.Client, string) error) error {
	_, p, err := m.config.Profile(profile)
	if err != nil {
		return err
	}
	if p.Desktop == nil || !p.Desktop.Enabled {
		return fmt.Errorf("native desktop is not enabled for this SSH profile")
	}
	duration, _ := time.ParseDuration(p.Desktop.Timeout)
	ctx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	stop := context.AfterFunc(m.ctx, cancel)
	defer stop()
	select {
	case m.desktop <- struct{}{}:
		defer func() { <-m.desktop }()
	case <-ctx.Done():
		return ctx.Err()
	}
	client, err := dial(ctx, p)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	defer client.Close()
	stopConnection := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stopConnection()
	err = action(client, p.Desktop.Display)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func runX11(client *ssh.Client, display, command, input string, consume func(io.Reader) error) error {
	session, err := client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.Stdin = strings.NewReader(input)
	session.Stderr = io.Discard
	output, err := session.StdoutPipe()
	if err != nil {
		return err
	}
	if err = session.Start("export DISPLAY=" + shellQuote(display) + "\n" + command); err != nil {
		return err
	}
	if err = consume(output); err != nil {
		_ = session.Close()
		return fmt.Errorf("read X11 result: %w", err)
	}
	if err = session.Wait(); err != nil {
		return fmt.Errorf("X11 operation failed; check display access and xwd/xdotool installation: %w", err)
	}
	return nil
}

func (m *Manager) x11Input(ctx context.Context, profile, input string, command func(desktop.Info) (string, error)) (InputResult, error) {
	var result InputResult
	err := m.withX11(ctx, profile, func(client *ssh.Client, display string) error {
		info, err := x11Geometry(client, display)
		if err != nil {
			return err
		}
		script, err := command(info)
		if err != nil {
			return err
		}
		if err = runX11(client, display, script, input, func(r io.Reader) error { _, err := io.Copy(io.Discard, r); return err }); err != nil {
			return fmt.Errorf("desktop input may be partially delivered; inspect before retrying: %w", err)
		}
		result = InputResult{Info: info, Message: "Input events sent; take a new screenshot to verify application state."}
		return nil
	})
	return result, err
}

func x11Geometry(client *ssh.Client, display string) (desktop.Info, error) {
	var info desktop.Info
	err := runX11(client, display, "xdotool getdisplaygeometry", "", func(r io.Reader) error {
		data, err := io.ReadAll(io.LimitReader(r, 129))
		if err != nil {
			return err
		}
		fields := strings.Fields(string(data))
		if len(data) > 128 || len(fields) != 2 {
			return fmt.Errorf("invalid X11 geometry")
		}
		w, e1 := strconv.Atoi(fields[0])
		h, e2 := strconv.Atoi(fields[1])
		if e1 != nil || e2 != nil || w < 1 || h < 1 || w > 8192 || h > 8192 || w*h > 8388608 {
			return fmt.Errorf("invalid X11 desktop dimensions")
		}
		info = desktop.Info{Width: w, Height: h, Name: "X11 desktop"}
		return nil
	})
	return info, err
}

func pointerCommand(events []desktop.PointerEvent) string {
	var script strings.Builder
	script.WriteString("set -e\ntrap 'xdotool mouseup 1 mouseup 2 mouseup 3 mouseup 4 mouseup 5 >/dev/null 2>&1 || true' EXIT\ntrap 'exit 129' HUP INT TERM\nxdotool")
	buttons := 0
	for _, event := range events {
		// --sync waits for movement and stalls when another backend already moved here.
		// 中文：--sync 等待坐标变化，另一后端已移到此位置时会阻塞。
		fmt.Fprintf(&script, " mousemove %d %d", event.X, event.Y)
		for button := 1; button <= 5; button++ {
			mask := 1 << (button - 1)
			if buttons&mask == event.Buttons&mask {
				continue
			}
			action := "mouseup"
			if event.Buttons&mask != 0 {
				action = "mousedown"
			}
			fmt.Fprintf(&script, " %s %d", action, button)
		}
		buttons = event.Buttons
	}
	return script.String()
}
