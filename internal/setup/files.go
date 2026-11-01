// Created by Yanjunhui
package setup

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/tunnel"
)

func copyFile(source, destination string, mode os.FileMode) error {
	src, err := os.Open(source)
	if err != nil {
		return err
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("安装源不是普通文件：%s", source)
	}
	dst, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func (i Installer) prepareTunnel(ctx context.Context, source, directory string) error {
	if source == "" || source == "tunnel-client" {
		sibling := filepath.Join(filepath.Dir(i.Executable), "tunnel-client")
		if _, err := os.Stat(sibling); err == nil {
			source = sibling
		}
	}
	resolved, err := exec.LookPath(source)
	if err == nil {
		if err = copyFile(resolved, filepath.Join(directory, "tunnel-client"), 0755); err != nil {
			return err
		}
		cloud := filepath.Join(filepath.Dir(resolved), "cloudflared")
		if _, err = os.Stat(cloud); err == nil {
			return copyFile(cloud, filepath.Join(directory, "cloudflared"), 0755)
		}
		if !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	install := i.InstallTunnel
	if install == nil {
		install = tunnel.Install
	}
	return install(ctx, directory, tunnel.Release)
}

func atomicFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(path), ".tunnel-ssh-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, writeErr := f.Write(data)
	modeErr := f.Chmod(mode)
	closeErr := f.Close()
	if writeErr != nil {
		return writeErr
	}
	if modeErr != nil {
		return modeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), path)
}

// switchCurrent publishes a symlink by rename on the same filesystem; readers
// observe either generation, never a missing link between remove/create steps.
// 中文：switchCurrent 在同一文件系统内重命名符号链接，避免先删后建造成读取空窗。
func switchCurrent(path, target string) error {
	dir, err := os.MkdirTemp(filepath.Dir(path), ".current-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	link := filepath.Join(dir, "link")
	if err = os.Symlink(target, link); err != nil {
		return err
	}
	return os.Rename(link, path)
}
