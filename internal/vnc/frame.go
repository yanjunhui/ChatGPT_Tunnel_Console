// Created by Yanjunhui
package vnc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"io"
)

type pngBuffer struct{ bytes.Buffer }

// Write limits encoded output before buffering it; an oversized PNG is discarded.
// 中文：Write 在缓存前限制编码输出大小，超限 PNG 整体丢弃。
func (b *pngBuffer) Write(p []byte) (int, error) {
	if b.Len()+len(p) > 16*1024*1024 {
		return 0, fmt.Errorf("VNC screenshot exceeds 16 MiB PNG limit")
	}
	return b.Buffer.Write(p)
}

// Screenshot requests a full Raw framebuffer and returns a PNG at native size.
// It rejects oversized, unsupported or out-of-bounds updates; transport closure
// interrupts reads. A partial framebuffer is never returned as a complete image.
// 中文：Screenshot 请求完整 Raw 帧并返回原尺寸 PNG；拒绝过大、不支持或越界
// 更新，关闭传输可中断读取，不将不完整帧作为完整截图返回。
func (c *Client) Screenshot() ([]byte, error) {
	c.frame = image.NewRGBA(image.Rect(0, 0, c.info.Width, c.info.Height))
	c.covered = 0
	defer func() { c.frame = nil }()
	format := []byte{0, 0, 0, 0, 32, 24, 0, 1, 0, 255, 0, 255, 0, 255, 16, 8, 0, 0, 0, 0}
	encodings := []byte{2, 0, 0, 2, 0, 0, 0, 0, 255, 255, 255, 33}
	if err := c.write(append(format, encodings...)); err != nil {
		return nil, err
	}
	if err := c.requestFrame(); err != nil {
		return nil, err
	}
	for c.covered != c.info.Width*c.info.Height {
		updated, err := c.readMessage()
		if err != nil {
			return nil, fmt.Errorf("read VNC framebuffer: %w", err)
		}
		if updated && c.covered != c.info.Width*c.info.Height {
			if err = c.requestFrame(); err != nil {
				return nil, err
			}
		}
	}
	var result pngBuffer
	if err := png.Encode(&result, c.frame); err != nil {
		return nil, err
	}
	return result.Bytes(), nil
}

func (c *Client) requestFrame() error {
	request := make([]byte, 10)
	request[0] = 3
	binary.BigEndian.PutUint16(request[6:], uint16(c.info.Width))
	binary.BigEndian.PutUint16(request[8:], uint16(c.info.Height))
	return c.write(request)
}

func (c *Client) readMessage() (bool, error) {
	kind, err := c.read(1)
	if err != nil {
		return false, err
	}
	switch kind[0] {
	case 0:
		header, err := c.read(3)
		if err != nil {
			return false, err
		}
		for range int(binary.BigEndian.Uint16(header[1:])) {
			if err := c.readRectangle(); err != nil {
				return false, err
			}
		}
		return true, nil
	case 2:
		return false, nil
	case 3:
		header, err := c.read(7)
		if err != nil {
			return false, err
		}
		n := binary.BigEndian.Uint32(header[3:])
		if n > 1024*1024 {
			return false, fmt.Errorf("VNC clipboard message exceeds 1 MiB")
		}
		_, err = io.CopyN(io.Discard, c.rw, int64(n))
		return false, err
	default:
		return false, fmt.Errorf("unsupported VNC server message %d", kind[0])
	}
}

func (c *Client) readRectangle() error {
	header, err := c.read(12)
	if err != nil {
		return err
	}
	x, y := int(binary.BigEndian.Uint16(header)), int(binary.BigEndian.Uint16(header[2:]))
	w, h := int(binary.BigEndian.Uint16(header[4:])), int(binary.BigEndian.Uint16(header[6:]))
	encoding := int32(binary.BigEndian.Uint32(header[8:]))
	if encoding == -223 {
		return c.resize(w, h)
	}
	if encoding != 0 {
		return fmt.Errorf("unsupported VNC encoding %d; only Raw and DesktopSize were requested", encoding)
	}
	if w == 0 || h == 0 || x+w > c.info.Width || y+h > c.info.Height {
		return fmt.Errorf("VNC rectangle outside framebuffer")
	}
	row := make([]byte, w*4)
	for yy := y; yy < y+h; yy++ {
		if _, err = io.ReadFull(c.rw, row); err != nil {
			return err
		}
		c.copyRow(x, yy, row)
	}
	return nil
}

func (c *Client) copyRow(x, y int, source []byte) {
	start := c.frame.PixOffset(x, y)
	for i := 0; i < len(source); i += 4 {
		pixel := c.frame.Pix[start+i : start+i+4]
		if pixel[3] == 0 {
			c.covered++
		}
		pixel[0], pixel[1], pixel[2], pixel[3] = source[i+2], source[i+1], source[i], 255
	}
}
