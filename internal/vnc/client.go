// Created by Yanjunhui
// Package vnc implements bounded, short-lived RFB operations over an owned tunnel.
// 中文：vnc 包在调用者管理的隧道连接上实现有资源边界的短时 RFB 操作。
package vnc

import (
	"encoding/binary"
	"fmt"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"image"
	"io"
)

const maxPixels = 8 * 1024 * 1024

// Client is not concurrency-safe. The caller closes its transport on cancellation.
// Framebuffers are allocated only for screenshots, not for keyboard/mouse calls.
// 中文：Client 不支持并发使用，调用者须在取消时关闭传输；仅截图分配帧缓冲。
type Client struct {
	rw      io.ReadWriter
	info    desktop.Info
	frame   *image.RGBA
	covered int
}

// New performs a shared RFB 3.3/3.7/3.8 handshake with None or VNC authentication.
// Passwords are optional printable ASCII, at most eight bytes; no silent truncation.
// The caller owns transport cleanup on both success and failure.
// 中文：New 执行共享模式握手，支持无密码及传统 VNC 鉴权；密码最多八个可打印
// ASCII 字节，不静默截断。成功或失败时的传输清理由调用者负责。
func New(rw io.ReadWriter, password string) (*Client, error) {
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	c := &Client{rw: rw}
	version, err := c.read(12)
	if err != nil {
		return nil, fmt.Errorf("read RFB version: %w", err)
	}
	if string(version) != "RFB 003.003\n" && string(version) != "RFB 003.007\n" && string(version) != "RFB 003.008\n" {
		return nil, fmt.Errorf("unsupported RFB version; expected 3.3, 3.7 or 3.8")
	}
	if err = c.write(version); err != nil {
		return nil, err
	}
	if err = c.authenticate(string(version), password); err != nil {
		return nil, err
	}
	if err = c.initialize(); err != nil {
		return nil, err
	}
	return c, nil
}

// Info returns metadata without another network request.
// 中文：Info 返回当前元数据，不另行发送网络请求。
func (c *Client) Info() desktop.Info { return c.info }

func (c *Client) initialize() error {
	if err := c.write([]byte{1}); err != nil {
		return err
	}
	header, err := c.read(24)
	if err != nil {
		return fmt.Errorf("read RFB server initialization: %w", err)
	}
	if err = c.resize(int(binary.BigEndian.Uint16(header)), int(binary.BigEndian.Uint16(header[2:]))); err != nil {
		return err
	}
	n := binary.BigEndian.Uint32(header[20:])
	if n > 4096 {
		return fmt.Errorf("RFB desktop name exceeds 4096 bytes")
	}
	name, err := c.read(int(n))
	c.info.Name = string(name)
	return err
}

func (c *Client) resize(width, height int) error {
	if width < 1 || height < 1 || width > 8192 || height > 8192 || width*height > maxPixels {
		return fmt.Errorf("VNC desktop exceeds 8192 per side or 8388608 pixels, or is empty")
	}
	c.info.Width, c.info.Height = width, height
	c.covered = 0
	if c.frame != nil {
		c.frame = image.NewRGBA(image.Rect(0, 0, width, height))
	}
	return nil
}

func (c *Client) read(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := io.ReadFull(c.rw, b)
	return b, err
}

func (c *Client) write(b []byte) error {
	for len(b) > 0 {
		n, err := c.rw.Write(b)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		b = b[n:]
	}
	return nil
}
