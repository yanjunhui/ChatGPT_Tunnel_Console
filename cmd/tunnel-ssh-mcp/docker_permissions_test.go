// Created by Yanjunhui
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Linux bind mounts preserve numeric owners, unlike Docker Desktop's file sharing.
// Restore only this test's temporary tree before Go removes its private files.
// 中文：Linux 挂载保留数字用户身份，与 Docker Desktop 文件共享不同；
// Go 清理私有文件前，只恢复本测试临时目录的归属。
func prepareDockerFixture(t *testing.T, root, workspace string) {
	t.Helper()
	t.Cleanup(func() { setDockerFixtureOwner(t, root, os.Getuid(), os.Getgid()) })
	setDockerFixtureOwner(t, workspace, 1000, 1000)
}

func setDockerFixtureOwner(t *testing.T, path string, uid, gid int) {
	t.Helper()
	image := os.Getenv("TUNNEL_SSH_MCP_IMAGE")
	if image == "" {
		image = "go-tunnel-ssh-mcp:latest"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--user", "0:0",
		"--mount", "type=bind,src="+path+",dst=/fixture", "--entrypoint", "chown",
		image, "-R", fmt.Sprintf("%d:%d", uid, gid), "/fixture")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("test fixture ownership: %v %s", err, out)
	}
}
