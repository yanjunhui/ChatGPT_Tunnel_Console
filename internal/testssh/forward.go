// Created by Yanjunhui
package testssh

import (
	"io"
	"net"
	"strconv"
	"time"

	"golang.org/x/crypto/ssh"
)

func forward(next ssh.NewChannel) {
	var target struct {
		Host       string
		Port       uint32
		Origin     string
		OriginPort uint32
	}
	if ssh.Unmarshal(next.ExtraData(), &target) != nil || !net.ParseIP(target.Host).IsLoopback() || target.Port > 65535 {
		_ = next.Reject(ssh.Prohibited, "test forwarding only supports loopback")
		return
	}
	remote, err := net.DialTimeout("tcp", net.JoinHostPort(target.Host, strconv.Itoa(int(target.Port))), time.Second)
	if err != nil {
		_ = next.Reject(ssh.ConnectionFailed, "test target unavailable")
		return
	}
	defer remote.Close()
	channel, requests, err := next.Accept()
	if err != nil {
		return
	}
	defer channel.Close()
	go ssh.DiscardRequests(requests)
	done := make(chan struct{})
	go func() { defer close(done); _, _ = io.Copy(channel, remote); _ = channel.Close() }()
	_, _ = io.Copy(remote, channel)
	_ = remote.Close()
	<-done
}
