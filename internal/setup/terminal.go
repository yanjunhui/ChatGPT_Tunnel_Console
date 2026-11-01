// Created by Yanjunhui
package setup

import (
	"context"
	"errors"
	"io"
	"unicode/utf8"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

// terminalLine uses raw mode for hidden input and restores terminal state after
// errors, Ctrl+C or EOF.
// 中文：terminalLine 对隐藏输入使用原始模式，错误、Ctrl+C 或 EOF 后均恢复终端状态。
func terminalLine(ctx context.Context, fd int, hidden bool) (string, error) {
	if hidden {
		state, err := term.MakeRaw(fd)
		if err != nil {
			return "", err
		}
		defer term.Restore(fd, state)
	}
	var value []byte
	for {
		b, err := terminalByte(ctx, fd)
		if err != nil {
			return "", err
		}
		switch b {
		case '\n', '\r':
			return string(value), nil
		case 3:
			return "", context.Canceled
		case 4:
			return "", io.EOF
		case 8, 127:
			if len(value) > 0 {
				_, size := utf8.DecodeLastRune(value)
				value = value[:len(value)-size]
			}
		default:
			value = append(value, b)
		}
	}
}

func terminalByte(ctx context.Context, fd int) (byte, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		fds := []unix.PollFd{{Fd: int32(fd), Events: unix.POLLIN}}
		n, err := unix.Poll(fds, 100)
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			continue
		}
		var b [1]byte
		n, err = unix.Read(fd, b[:])
		if errors.Is(err, unix.EINTR) {
			continue
		}
		if err != nil {
			return 0, err
		}
		if n == 0 {
			return 0, io.EOF
		}
		return b[0], nil
	}
}
