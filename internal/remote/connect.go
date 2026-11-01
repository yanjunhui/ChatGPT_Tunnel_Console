// Created by Yanjunhui
package remote

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func authenticate(p config.Profile) ([]ssh.AuthMethod, func(), error) {
	var methods []ssh.AuthMethod
	cleanup := func() {}
	if p.PrivateKey != "" {
		signer, err := privateKey(p)
		if err != nil {
			return nil, cleanup, err
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}
	if p.Password != "" {
		password, err := config.Secret(p.Password)
		if err != nil {
			return nil, cleanup, err
		}
		methods = append(methods, ssh.Password(password))
	}
	if p.AgentSocket != "" {
		socket := p.AgentSocket
		if len(socket) > 4 && socket[:4] == "env:" {
			socket = os.Getenv(socket[4:])
		}
		conn, err := net.DialTimeout("unix", socket, 5*time.Second)
		if err != nil {
			return nil, cleanup, fmt.Errorf("connect SSH agent: %w", err)
		}
		cleanup = func() { _ = conn.Close() }
		methods = append(methods, ssh.PublicKeysCallback(agent.NewClient(conn).Signers))
	}
	return methods, cleanup, nil
}

func privateKey(p config.Profile) (ssh.Signer, error) {
	data, err := os.ReadFile(p.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("read SSH private key: %w", err)
	}
	passphrase, err := config.Secret(p.Passphrase)
	if err != nil {
		return nil, err
	}
	var signer ssh.Signer
	if passphrase != "" {
		signer, err = ssh.ParsePrivateKeyWithPassphrase(data, []byte(passphrase))
	} else {
		signer, err = ssh.ParsePrivateKey(data)
	}
	if err != nil {
		return nil, fmt.Errorf("cannot parse SSH private key (check format and passphrase)")
	}
	return signer, nil
}

func dial(ctx context.Context, p config.Profile) (*ssh.Client, error) {
	duration, _ := time.ParseDuration(p.ConnectTimeout)
	if duration > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, duration)
		defer cancel()
	}
	methods, cleanup, err := authenticate(p)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", p.Address())
	if err != nil {
		return nil, fmt.Errorf("connect SSH %s: %w", p.Address(), err)
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	// Caller authentication is delegated to OpenAI Tunnel. This product has no
	// separate target host-key policy; SSH login still uses the target credentials.
	// The tunnel does not authenticate the SSH host; its host key is not checked.
	// 中文：隧道验证调用方，SSH 使用目标凭据登录；当前不校验目标主机公钥，
	// 隧道认证不能替代 SSH 主机身份认证。
	cfg := &ssh.ClientConfig{User: p.User, Auth: methods, HostKeyCallback: ssh.InsecureIgnoreHostKey()}
	c, chans, reqs, err := ssh.NewClientConn(conn, p.Address(), cfg)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("SSH handshake %s: %w", p.Address(), err)
	}
	if err = ctx.Err(); err != nil {
		_ = c.Close()
		return nil, err
	}
	_ = conn.SetDeadline(time.Time{})
	return ssh.NewClient(c, chans, reqs), nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func inDirectory(command, directory string) string {
	if directory == "" {
		return command
	}
	// Abort before any script line runs if changing the working directory fails.
	// 中文：切换目录失败时先退出，防止多行脚本继续在错误目录执行。
	return "cd -- " + shellQuote(directory) + " || exit $?\n" + command
}
