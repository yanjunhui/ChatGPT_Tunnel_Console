// Created by Yanjunhui
// Package desktop holds shared image and input contracts, independent of transport.
// 中文：desktop 包定义独立于传输方式的图片及输入契约。
package desktop

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PointerEvent uses native screenshot coordinates and RFB button bits:
// left=1, middle=2, right=4, wheel-up=8, wheel-down=16; zero releases buttons.
// 中文：PointerEvent 使用原图坐标和 RFB 按钮位；左1、中2、右4、上滚8、下滚16，零释放。
type PointerEvent struct {
	X       int `json:"x"`
	Y       int `json:"y"`
	Buttons int `json:"buttons"`
}

// ValidatePointer checks batch size and protocol ranges before a connection opens.
// Desktop-specific coordinate limits are checked by Pointer before any input.
// 中文：ValidatePointer 在连接前检查批量大小和协议范围；实际桌面边界由 Pointer 检查。
func ValidatePointer(events []PointerEvent) error {
	if len(events) == 0 || len(events) > 256 {
		return fmt.Errorf("pointer events must contain 1..256 items")
	}
	for _, event := range events {
		if event.X < 0 || event.Y < 0 || event.X > 8191 || event.Y > 8191 || event.Buttons < 0 || event.Buttons > 31 {
			return fmt.Errorf("pointer coordinates must be 0..8191 and buttons 0..31")
		}
	}
	return nil
}

// ValidateKeys resolves a 1..16-key chord or returns an unsupported-key error.
// Names follow common X11 keysyms; one printable Unicode character is also valid.
// 中文：ValidateKeys 解析 1..16 键组合；接受常见 X11 键名或单个可打印 Unicode 字符。
func ValidateKeys(keys []string) ([]uint32, error) {
	if len(keys) == 0 || len(keys) > 16 {
		return nil, fmt.Errorf("keys must contain 1..16 items")
	}
	result := make([]uint32, len(keys))
	for i, key := range keys {
		var err error
		result[i], err = keysym(key)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

// ValidateText accepts 1..4096 Unicode characters, including tab and line endings.
// Other control characters and invalid UTF-8 are rejected before sending anything.
// 中文：ValidateText 接受 1..4096 个 Unicode 字符（含制表与换行），拒绝其他控制字符及非法 UTF-8。
func ValidateText(text string) error {
	if !utf8.ValidString(text) || utf8.RuneCountInString(text) == 0 || utf8.RuneCountInString(text) > 4096 {
		return fmt.Errorf("text must contain 1..4096 valid Unicode characters")
	}
	for _, r := range text {
		if !unicode.IsPrint(r) && r != '\n' && r != '\r' && r != '\t' {
			return fmt.Errorf("text contains an unsupported control character")
		}
	}
	return nil
}

// RuneKeysym maps an accepted Unicode rune to an X11 keysym.
// 中文：RuneKeysym 将已校验 Unicode 字符映射为 X11 键符。
func RuneKeysym(r rune) uint32 {
	switch r {
	case '\n', '\r':
		return 0xff0d
	case '\t':
		return 0xff09
	}
	if r <= 255 {
		return uint32(r)
	}
	return 0x01000000 | uint32(r)
}

func keysym(key string) (uint32, error) {
	if utf8.ValidString(key) && utf8.RuneCountInString(key) == 1 {
		r, _ := utf8.DecodeRuneInString(key)
		if unicode.IsPrint(r) {
			return RuneKeysym(r), nil
		}
	}
	name := strings.ToLower(key)
	if symbol, ok := namedKeys[name]; ok {
		return symbol, nil
	}
	if strings.HasPrefix(name, "f") {
		if n, err := strconv.Atoi(name[1:]); err == nil && n >= 1 && n <= 12 {
			return uint32(0xffbd + n), nil
		}
	}
	return 0, fmt.Errorf("unsupported key name; use a documented keysym or one printable character")
}

var namedKeys = map[string]uint32{
	"ctrl": 0xffe3, "control_l": 0xffe3, "control_r": 0xffe4,
	"shift": 0xffe1, "shift_l": 0xffe1, "shift_r": 0xffe2,
	"alt": 0xffe9, "alt_l": 0xffe9, "alt_r": 0xffea,
	"super": 0xffeb, "super_l": 0xffeb, "super_r": 0xffec,
	"enter": 0xff0d, "return": 0xff0d, "tab": 0xff09,
	"escape": 0xff1b, "esc": 0xff1b, "backspace": 0xff08, "delete": 0xffff,
	"home": 0xff50, "end": 0xff57, "insert": 0xff63,
	"left": 0xff51, "up": 0xff52, "right": 0xff53, "down": 0xff54,
	"page_up": 0xff55, "page_down": 0xff56, "space": 0x20,
}
