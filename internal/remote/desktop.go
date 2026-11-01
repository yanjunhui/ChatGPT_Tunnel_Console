// Created by Yanjunhui
package remote

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/vnc"
)

// DesktopInput selects an SSH profile with the requested desktop backend enabled.
// 中文：DesktopInput 选择启用所请求桌面后端的 SSH 配置；空名称使用默认配置。
type DesktopInput struct {
	Profile string `json:"profile,omitempty"`
}

// PointerInput groups one move/click/drag/scroll gesture in native image pixels.
// 中文：PointerInput 按原图像素坐标描述一次移动、点击、拖动或滚动手势。
type PointerInput struct {
	DesktopInput
	Events []desktop.PointerEvent `json:"events" jsonschema:"1..256 events; x/y are native screenshot pixels; buttons: left 1, middle 2, right 4, wheel-up 8, wheel-down 16. A final release is sent automatically."`
}

// KeyInput presses one chord, then releases it; keys are not held across calls.
// 中文：KeyInput 按下并释放一组组合键，不跨请求保持按键。
type KeyInput struct {
	DesktopInput
	Keys []string `json:"keys" jsonschema:"1..16 keys in press order, for example Control_L and l, or Return. Released in reverse order."`
}

// TextInput sends Unicode key events without using the clipboard.
// 中文：TextInput 通过 Unicode 按键事件输入文本，不使用剪贴板。
type TextInput struct {
	DesktopInput
	Text string `json:"text" jsonschema:"1..4096 Unicode characters. Newline sends Return; tab sends Tab. Target Unicode support may vary."`
}

// InputResult acknowledges transport delivery, not completion in the application.
// 中文：InputResult 仅确认传输已发送，不表示目标应用已完成操作。
type InputResult struct {
	Info    desktop.Info `json:"desktop"`
	Message string       `json:"message"`
}

// Screenshot returns a native PNG plus metadata, or a profile/SSH/RFB error.
// The connection is closed on completion, cancellation or the profile deadline.
// 中文：Screenshot 返回原尺寸 PNG 及元数据，或配置／SSH／RFB 错误；
// 操作完成、取消或达到配置期限时关闭连接。
func (m *Manager) Screenshot(ctx context.Context, in DesktopInput) ([]byte, desktop.Info, error) {
	var data []byte
	var info desktop.Info
	err := m.withDesktop(ctx, in.Profile, func(client *vnc.Client) error {
		var err error
		data, err = client.Screenshot()
		info = client.Info()
		return err
	})
	return data, info, err
}

// Pointer validates the gesture, forwards it over SSH and releases its buttons.
// Partial delivery is possible on transport failure; callers must inspect again.
// 中文：Pointer 校验并通过 SSH 发送手势及释放事件；传输失败可能部分发送，须重新查看。
func (m *Manager) Pointer(ctx context.Context, in PointerInput) (InputResult, error) {
	if err := desktop.ValidatePointer(in.Events); err != nil {
		return InputResult{}, err
	}
	return m.desktopInput(ctx, in.Profile, func(c *vnc.Client) error { return c.Pointer(in.Events) })
}

// Key validates and sends a chord with releases, returning delivery or I/O errors.
// 中文：Key 校验并发送组合键和释放事件，返回发送结果或 I/O 错误。
func (m *Manager) Key(ctx context.Context, in KeyInput) (InputResult, error) {
	if _, err := desktop.ValidateKeys(in.Keys); err != nil {
		return InputResult{}, err
	}
	return m.desktopInput(ctx, in.Profile, func(c *vnc.Client) error { return c.Key(in.Keys) })
}

// Type validates the entire text before connecting and sends Unicode key pairs.
// 中文：Type 在连接前验证全部文本，然后发送 Unicode 按下／释放事件对。
func (m *Manager) Type(ctx context.Context, in TextInput) (InputResult, error) {
	if err := desktop.ValidateText(in.Text); err != nil {
		return InputResult{}, err
	}
	return m.desktopInput(ctx, in.Profile, func(c *vnc.Client) error { return c.Type(in.Text) })
}

func (m *Manager) desktopInput(ctx context.Context, profile string, action func(*vnc.Client) error) (InputResult, error) {
	var result InputResult
	err := m.withDesktop(ctx, profile, func(client *vnc.Client) error {
		if err := action(client); err != nil {
			return fmt.Errorf("VNC input may be partially delivered; do not blindly retry: %w", err)
		}
		result = InputResult{Info: client.Info(), Message: "Input and release events sent; request a new screenshot to verify application state."}
		return nil
	})
	return result, err
}

func (m *Manager) withDesktop(ctx context.Context, profile string, action func(*vnc.Client) error) error {
	_, p, err := m.config.Profile(profile)
	if err != nil {
		return err
	}
	if p.VNC == nil || !p.VNC.Enabled {
		return fmt.Errorf("VNC is not enabled for this SSH profile")
	}
	timeout, _ := time.ParseDuration(p.VNC.Timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stop := context.AfterFunc(m.ctx, cancel)
	defer stop()
	// Serialize gestures and bound framebuffer memory across this MCP process.
	// 中文：在当前 MCP 进程内串行操作，避免手势交错并限制帧缓冲内存。
	select {
	case m.desktop <- struct{}{}:
		defer func() { <-m.desktop }()
	case <-ctx.Done():
		return ctx.Err()
	}
	client, cleanup, err := desktopConnection(ctx, p)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return err
	}
	defer cleanup()
	err = action(client)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func desktopConnection(ctx context.Context, p config.Profile) (*vnc.Client, func(), error) {
	password, err := config.Secret(p.VNC.Password)
	if err != nil {
		return nil, nil, fmt.Errorf("VNC credential: %w", err)
	}
	sshClient, err := dial(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = sshClient.Close() })
	cleanup := func() { stop(); _ = sshClient.Close() }
	address := net.JoinHostPort(p.VNC.Host, strconv.Itoa(p.VNC.Port))
	conn, err := sshClient.DialContext(ctx, "tcp", address)
	if err != nil {
		cleanup()
		return nil, nil, fmt.Errorf("SSH forward to VNC: %w", err)
	}
	client, err := vnc.New(conn, password)
	if err != nil {
		_ = conn.Close()
		cleanup()
		return nil, nil, fmt.Errorf("VNC handshake: %w", err)
	}
	return client, func() { _ = conn.Close(); cleanup() }, nil
}
