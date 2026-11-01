// Created by Yanjunhui
package testssh

import (
	"context"
	"errors"
	"io"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"golang.org/x/crypto/ssh"
)

func run(ctx context.Context, ch ssh.Channel, command, root string, terminal bool) {
	defer ch.Close()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Dir = root
	cmd.Env = []string{"PATH=/usr/bin:/bin:/usr/sbin:/sbin", "HOME=" + root, "SHELL=/bin/sh", "TERM=xterm"}
	cmd.WaitDelay = time.Second
	cmd.Cancel = func() error {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return nil
		}
		return err
	}
	var err error
	if terminal {
		err = runTerminal(cmd, ch)
	} else {
		err = runPipes(cmd, ch)
	}
	status := uint32(0)
	if err != nil {
		status = 1
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() >= 0 {
			status = uint32(exit.ExitCode())
		}
	}
	_, _ = ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{status}))
}

func runTerminal(cmd *exec.Cmd, ch ssh.Channel) error {
	f, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer f.Close()
	var output sync.WaitGroup
	output.Add(1)
	go func() { defer output.Done(); _, _ = io.Copy(ch, f) }()
	go func() { _, _ = io.Copy(f, ch) }()
	err = cmd.Wait()
	output.Wait()
	return err
}

func runPipes(cmd *exec.Cmd, ch ssh.Channel) error {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	cmd.Stdout, cmd.Stderr = ch, ch.Stderr()
	if err = cmd.Start(); err != nil {
		_ = stdin.Close()
		return err
	}
	go func() { _, _ = io.Copy(stdin, ch); _ = stdin.Close() }()
	return cmd.Wait()
}
