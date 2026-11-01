// Created by Yanjunhui
package main

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/creack/pty"
)

func (r dockerInitRig) checkWizard(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	cmd := r.command(ctx, true, "run", "--rm", "init", "--reconfigure")
	terminal, err := pty.Start(cmd)
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()
	defer cmd.Process.Kill()
	chunks := readTerminal(ctx, terminal)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var transcript strings.Builder
	for _, step := range []struct{ prompt, answer string }{
		{"OpenAI 隧道 ID: ", "tunnel_interactive\r"},
		{"OpenAI 运行时 API Key（不回显）: ", "docker-interactive-secret\r"},
		{"SSH 目标：1 自带 Docker 开发环境 / 2 外部 SSH 主机 / 3 自带图形桌面 [1]: ", "\r"},
	} {
		awaitPrompt(t, ctx, chunks, &transcript, step.prompt)
		time.Sleep(50 * time.Millisecond)
		if _, err = terminal.Write([]byte(step.answer)); err != nil {
			t.Fatal(err)
		}
	}
	awaitPrompt(t, ctx, chunks, &transcript, "容器初始化完成")
	select {
	case err = <-done:
		if err != nil {
			t.Fatal(err)
		}
	case <-ctx.Done():
		t.Fatal("interactive init did not exit")
	}
	if strings.Contains(transcript.String(), "docker-interactive-secret") {
		t.Fatal("Docker TTY echoed API key")
	}
}

func readTerminal(ctx context.Context, file *os.File) <-chan string {
	chunks := make(chan string, 16)
	go func() {
		defer close(chunks)
		buffer := make([]byte, 4096)
		for {
			n, err := file.Read(buffer)
			if n > 0 {
				select {
				case chunks <- string(buffer[:n]):
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return chunks
}

func awaitPrompt(t *testing.T, ctx context.Context, chunks <-chan string, transcript *strings.Builder, prompt string) {
	t.Helper()
	for !strings.Contains(transcript.String(), prompt) {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				t.Fatalf("TTY closed before prompt %q", prompt)
			}
			transcript.WriteString(chunk)
		case <-ctx.Done():
			t.Fatalf("timed out waiting for prompt %q", prompt)
		}
	}
}
