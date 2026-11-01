// Created by Yanjunhui
package setup

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// Console separates prompts from secret input so tests can avoid a real terminal.
// ReadSecret must not echo sensitive input; Context governs terminal cancellation.
// 中文：Console 分离普通输入和密钥输入，便于无终端测试；ReadSecret 不应回显密钥，
// Context 控制终端输入取消。
type Console struct {
	Context    context.Context
	Input      io.Reader
	Output     io.Writer
	ReadSecret func() (string, error)
}

// Terminal creates a console whose hidden input requires a TTY and restores its mode.
// 中文：Terminal 创建终端控制台，隐藏输入必须来自 TTY，结束时恢复终端模式。
func Terminal(ctx context.Context, input *os.File, output io.Writer) *Console {
	return &Console{Context: ctx, Input: input, Output: output, ReadSecret: func() (string, error) {
		if !term.IsTerminal(int(input.Fd())) {
			return "", fmt.Errorf("初始化向导需要终端；脚本请使用 init --non-interactive --config 文件")
		}
		data, err := terminalLine(ctx, int(input.Fd()), true)
		fmt.Fprintln(output)
		return data, err
	}}
}

// Ask trims a line, uses fallback for empty input and returns read/cancellation errors.
// 中文：Ask 去除输入两端空白，空输入采用 fallback，读取或取消失败时返回错误。
func (c *Console) Ask(label, fallback string) (string, error) {
	if fallback != "" {
		fmt.Fprintf(c.Output, "%s [%s]: ", label, fallback)
	} else {
		fmt.Fprintf(c.Output, "%s: ", label)
	}
	line, err := c.readLine()
	if err != nil {
		return "", fmt.Errorf("读取%s: %w", label, err)
	}
	value := strings.TrimSpace(line)
	if value == "" {
		value = fallback
	}
	return value, nil
}

func (c *Console) readLine() (string, error) {
	if input, ok := c.Input.(*os.File); ok && term.IsTerminal(int(input.Fd())) {
		ctx := c.Context
		if ctx == nil {
			ctx = context.Background()
		}
		return terminalLine(ctx, int(input.Fd()), false)
	}
	return line(c.Input)
}

// Secret reads without trimming; required rejects an empty value, not whitespace.
// 中文：Secret 保留输入空白；required 仅拒绝空字符串，不把纯空白视为空值。
func (c *Console) Secret(label string, required bool) (string, error) {
	fmt.Fprintf(c.Output, "%s（不回显）: ", label)
	value, err := c.ReadSecret()
	if err != nil {
		return "", err
	}
	if required && value == "" {
		return "", fmt.Errorf("%s不能为空", label)
	}
	return value, nil
}

// Read a byte at a time so terminal prompts do not buffer subsequent secret input.
// 中文：逐字节读取，避免普通提示提前缓存后续密钥输入。
func line(reader io.Reader) (string, error) {
	var b strings.Builder
	var data [1]byte
	for {
		n, err := reader.Read(data[:])
		if n > 0 {
			if data[0] == '\n' {
				return strings.TrimSuffix(b.String(), "\r"), nil
			}
			b.WriteByte(data[0])
		}
		if err != nil {
			return "", err
		}
	}
}
