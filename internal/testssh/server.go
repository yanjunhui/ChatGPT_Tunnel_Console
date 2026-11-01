// Created by Yanjunhui
// Package testssh provides a real SSH/SFTP protocol endpoint for local tests.
// 中文：testssh 包为本地测试提供真实 SSH/SFTP 协议端点。
package testssh

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"

	"github.com/pkg/sftp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"
	"golang.org/x/crypto/ssh"
)

// Server exposes a temporary workdir and matching client config for local tests.
// 中文：Server 提供临时工作目录及配套客户端配置，仅用于本地测试。
type Server struct {
	Config      *config.Config
	Root        string
	listener    net.Listener
	mu          sync.Mutex
	connections map[net.Conn]bool
	closed      bool
	wg          sync.WaitGroup
}

// Start opens an ephemeral loopback port and registers all cleanup with t.
// Setup failures fail the test; no existing SSH host or tunnel is contacted.
// 中文：Start 监听临时回环端口并注册测试清理；准备失败直接使测试失败，
// 不连接已有 SSH 主机或隧道。
func Start(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	key := makeKey(t, root)
	cfg := &ssh.ServerConfig{PublicKeyCallback: func(_ ssh.ConnMetadata, pub ssh.PublicKey) (*ssh.Permissions, error) {
		if !bytes.Equal(pub.Marshal(), key.PublicKey().Marshal()) {
			return nil, fmt.Errorf("wrong key")
		}
		return nil, nil
	}}
	cfg.AddHostKey(key)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	host, port, _ := net.SplitHostPort(listener.Addr().String())
	portNumber, _ := strconv.Atoi(port)
	s := &Server{Root: root, listener: listener, connections: make(map[net.Conn]bool)}
	s.Config = &config.Config{DefaultProfile: "test", OutputBytes: 4096, Profiles: map[string]config.Profile{"test": {
		Host: host, Port: portNumber, User: "tester", PrivateKey: filepath.Join(root, "client-key"), Workdir: root, ConnectTimeout: "2s", CommandTimeout: "5s",
	}}}
	s.wg.Add(1)
	go s.accept(cfg)
	t.Cleanup(s.close)
	return s
}

func makeKey(t *testing.T, root string) ssh.Signer {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	block, err := ssh.MarshalPrivateKey(private, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "client-key"), pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(private)
	if err != nil {
		t.Fatal(err)
	}
	return signer
}

func (s *Server) accept(cfg *ssh.ServerConfig) {
	defer s.wg.Done()
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		if s.closed {
			s.mu.Unlock()
			_ = conn.Close()
			return
		}
		s.connections[conn] = true
		s.wg.Add(1)
		s.mu.Unlock()
		go s.connection(conn, cfg)
	}
}

func (s *Server) connection(conn net.Conn, cfg *ssh.ServerConfig) {
	defer s.wg.Done()
	defer func() { _ = conn.Close(); s.mu.Lock(); delete(s.connections, conn); s.mu.Unlock() }()
	server, channels, requests, err := ssh.NewServerConn(conn, cfg)
	if err != nil {
		return
	}
	defer server.Close()
	go ssh.DiscardRequests(requests)
	var channelsWG sync.WaitGroup
	defer channelsWG.Wait()
	for next := range channels {
		if next.ChannelType() == "direct-tcpip" {
			channelsWG.Add(1)
			go func() { defer channelsWG.Done(); forward(next) }()
			continue
		}
		if next.ChannelType() != "session" {
			_ = next.Reject(ssh.UnknownChannelType, "only session")
			continue
		}
		ch, reqs, err := next.Accept()
		if err != nil {
			continue
		}
		channelsWG.Add(1)
		go func() { defer channelsWG.Done(); s.channel(ch, reqs) }()
	}
}

func (s *Server) channel(ch ssh.Channel, requests <-chan *ssh.Request) {
	defer ch.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	terminal := false
	for req := range requests {
		switch req.Type {
		case "pty-req":
			terminal = true
			_ = req.Reply(true, nil)
		case "exec":
			var payload struct{ Command string }
			if ssh.Unmarshal(req.Payload, &payload) != nil {
				_ = req.Reply(false, nil)
				continue
			}
			_ = req.Reply(true, nil)
			go run(ctx, ch, payload.Command, s.Root, terminal)
		case "subsystem":
			var payload struct{ Name string }
			if ssh.Unmarshal(req.Payload, &payload) != nil || payload.Name != "sftp" {
				_ = req.Reply(false, nil)
				continue
			}
			_ = req.Reply(true, nil)
			go func() {
				srv, err := sftp.NewServer(ch, sftp.WithServerWorkingDirectory(s.Root))
				if err == nil {
					_ = srv.Serve()
					_ = srv.Close()
				}
			}()
		case "signal":
			cancel()
			_ = req.Reply(true, nil)
		default:
			_ = req.Reply(false, nil)
		}
	}
}

func (s *Server) close() {
	s.mu.Lock()
	s.closed = true
	_ = s.listener.Close()
	for conn := range s.connections {
		_ = conn.Close()
	}
	s.mu.Unlock()
	s.wg.Wait()
}
