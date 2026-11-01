// Created by Yanjunhui
package desktop

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
)

// Info is the current desktop size in pixels and its server-provided name.
// 中文：Info 包含桌面像素尺寸及服务端提供的名称。
type Info struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Name   string `json:"name"`
}

type pngBuffer struct{ bytes.Buffer }

// Write rejects excessive PNG output before buffering it.
// 中文：Write 在缓存前拒绝过大的 PNG 输出。
func (b *pngBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 16*1024*1024 {
		return 0, fmt.Errorf("screenshot exceeds 16 MiB PNG limit")
	}
	return b.Buffer.Write(p)
}

// PNG encodes a native framebuffer and returns an error above the output limit.
// 中文：PNG 编码原尺寸帧缓冲，超过输出边界时返回错误。
func PNG(frame image.Image) ([]byte, error) {
	var b pngBuffer
	if err := png.Encode(&b, frame); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}
