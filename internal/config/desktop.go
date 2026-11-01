// Created by Yanjunhui
package config

import (
	"fmt"
	"regexp"
	"time"
)

// Desktop enables direct X11 capture/input over SSH, independently of VNC.
// Display selects an existing local X display on the SSH host; no TCP display.
// 中文：Desktop 通过 SSH 独立提供 X11 截图和输入，Display 仅选择目标机本地显示服务。
type Desktop struct {
	Enabled bool   `toml:"enabled"`
	Display string `toml:"display"`
	Timeout string `toml:"timeout"`
}

var localDisplay = regexp.MustCompile(`^:[0-9]{1,5}(\.[0-9]{1,3})?$`)

func (d *Desktop) normalize() error {
	if d.Display == "" {
		d.Display = ":1"
	}
	if !localDisplay.MatchString(d.Display) {
		return fmt.Errorf("desktop display must be a local X display such as :1 or :0.0")
	}
	if d.Timeout == "" {
		d.Timeout = "15s"
	}
	duration, err := time.ParseDuration(d.Timeout)
	if err != nil || duration <= 0 || duration > 2*time.Minute {
		return fmt.Errorf("desktop timeout must be positive and at most 120s")
	}
	return nil
}
