// Created by Yanjunhui
package setup

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

// lockInstallation holds a nonblocking process lock until the returned function runs.
// The persistent lock file is not proof that another process currently owns the lock.
// 中文：lockInstallation 获取非阻塞进程锁，调用返回函数时释放；
// 锁文件仍存在不代表当前仍有进程持锁。
func lockInstallation(directory string) (func(), error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(directory, ".init.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, fmt.Errorf("另一个 init 正在运行或无法取得安装锁：%w", err)
	}
	return func() { _ = f.Close() }, nil
}
