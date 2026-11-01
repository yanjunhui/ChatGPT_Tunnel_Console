// Created by Yanjunhui
package setup

import "github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/config"

func prepareDesktop(d draft, p config.Profile, name, directory string, files map[string][]byte) (config.Profile, error) {
	if p.VNC == nil {
		return p, nil
	}
	// Copy optional settings before replacing references; imports remain unchanged.
	// 中文：替换凭据引用前复制可选配置，保持原始导入对象不变。
	v := *p.VNC
	var err error
	v.Password, err = captureSecret(d, v.Password, name+"-vnc-password", directory, files)
	p.VNC = &v
	return p, err
}
