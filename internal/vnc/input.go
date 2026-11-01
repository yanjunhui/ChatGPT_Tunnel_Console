// Created by Yanjunhui
package vnc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"strings"
)

// Pointer sends one bounded gesture, then sends a release at its last position.
// Network errors may mean partial delivery; input is never retried automatically.
// 中文：Pointer 发送有界手势并在末位置发送释放；网络错误可能表示部分发送，不自动重试。
func (c *Client) Pointer(events []desktop.PointerEvent) (err error) {
	if err = desktop.ValidatePointer(events); err != nil {
		return err
	}
	for _, event := range events {
		if event.X >= c.info.Width || event.Y >= c.info.Height {
			return fmt.Errorf("pointer outside current %dx%d desktop; request a new screenshot", c.info.Width, c.info.Height)
		}
	}
	last := events[0]
	defer func() {
		last.Buttons = 0
		err = errors.Join(err, c.pointer(last))
	}()
	for _, event := range events {
		last = event
		if err = c.pointer(event); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) pointer(event desktop.PointerEvent) error {
	packet := []byte{5, byte(event.Buttons), 0, 0, 0, 0}
	binary.BigEndian.PutUint16(packet[2:], uint16(event.X))
	binary.BigEndian.PutUint16(packet[4:], uint16(event.Y))
	return c.write(packet)
}

// Key presses a chord in order and releases it in reverse order before returning.
// A failed transport returns an error even if a best-effort release also fails.
// 中文：Key 按顺序按下组合键、逆序释放；传输或尽力释放失败均返回错误。
func (c *Client) Key(keys []string) (err error) {
	symbols, err := desktop.ValidateKeys(keys)
	if err != nil {
		return err
	}
	pressed := []uint32{}
	defer func() {
		for i := len(pressed) - 1; i >= 0; i-- {
			err = errors.Join(err, c.key(pressed[i], false))
		}
	}()
	for _, symbol := range symbols {
		pressed = append(pressed, symbol)
		if err = c.key(symbol, true); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) key(symbol uint32, down bool) error {
	packet := make([]byte, 8)
	packet[0] = 4
	if down {
		packet[1] = 1
	}
	binary.BigEndian.PutUint32(packet[4:], symbol)
	return c.write(packet)
}

// Type sends Unicode key press/release pairs, with CRLF treated as one Return.
// It does not use or change the clipboard; target Unicode support can vary.
// 中文：Type 发送 Unicode 按下／释放对，CRLF 作为一次回车；不使用剪贴板，目标 Unicode 支持可能不同。
func (c *Client) Type(text string) error {
	if err := desktop.ValidateText(text); err != nil {
		return err
	}
	for _, r := range strings.ReplaceAll(text, "\r\n", "\n") {
		symbol := desktop.RuneKeysym(r)
		pressErr := c.key(symbol, true)
		releaseErr := c.key(symbol, false)
		if err := errors.Join(pressErr, releaseErr); err != nil {
			return err
		}
	}
	return nil
}
