// Created by Yanjunhui
package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"
)

type previous struct {
	target     string
	definition []byte
	exists     bool
}

func (i Installer) snapshot() (previous, error) {
	var old previous
	var err error
	old.target, err = os.Readlink(i.Manager.Paths.Current())
	if err != nil && !os.IsNotExist(err) {
		return old, fmt.Errorf("current 必须是安装器创建的符号链接：%w", err)
	}
	old.definition, err = os.ReadFile(i.Manager.Paths.Definition)
	old.exists = err == nil
	if err != nil && !os.IsNotExist(err) {
		return old, err
	}
	return old, nil
}

func (i Installer) activate(ctx context.Context, generation string, definition []byte) error {
	old, err := i.snapshot()
	if err != nil {
		return err
	}
	if err = i.Manager.Stop(ctx); err != nil {
		return fmt.Errorf("停止旧服务失败，未切换配置：%w", err)
	}
	if err = switchCurrent(i.Manager.Paths.Current(), generation); err == nil {
		err = atomicFile(i.Manager.Paths.Definition, definition, 0644)
	}
	if err == nil {
		err = i.Manager.Start(ctx)
	}
	if err == nil {
		return nil
	}
	// Recovery must still run after the initiating request has been canceled.
	// 中文：即使原请求已取消，也要使用独立期限尝试恢复旧服务。
	rollbackCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if restoreErr := i.restore(rollbackCtx, old); restoreErr != nil {
		return errors.Join(fmt.Errorf("安装失败：%w", err), fmt.Errorf("恢复旧服务也失败，请检查服务日志：%w", restoreErr))
	}
	return fmt.Errorf("安装失败，已恢复先前配置和服务：%w", err)
}

func (i Installer) restore(ctx context.Context, old previous) error {
	if err := i.Manager.Disable(ctx); err != nil {
		return err
	}
	if old.target != "" {
		if err := switchCurrent(i.Manager.Paths.Current(), old.target); err != nil {
			return err
		}
	} else if err := os.Remove(i.Manager.Paths.Current()); err != nil && !os.IsNotExist(err) {
		return err
	}
	if old.exists {
		if err := atomicFile(i.Manager.Paths.Definition, old.definition, 0644); err != nil {
			return err
		}
		return i.Manager.Start(ctx)
	}
	if err := os.Remove(i.Manager.Paths.Definition); err != nil && !os.IsNotExist(err) {
		return err
	}
	if i.Manager.Platform == "linux" {
		_, err := i.Manager.Run(ctx, "systemctl", "daemon-reload")
		return err
	}
	return nil
}
