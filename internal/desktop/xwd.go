// Created by Yanjunhui
package desktop

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
)

// XWD decodes a bounded X11 ZPixmap dump into an unscaled PNG and metadata.
// Only 24-bit RGB TrueColor or identity-mapped DirectColor with 24/32-bit storage is accepted; malformed,
// indexed-color, oversized or truncated dumps fail without a partial image.
// 中文：XWD 将有界 X11 ZPixmap 转为原尺寸 PNG 和元数据；仅支持 24 位 RGB
// TrueColor 或恒等映射 DirectColor、24/32 位存储，错误、索引色、过大或截断数据不返回部分图片。
func XWD(reader io.Reader) ([]byte, Info, error) {
	var header [25]uint32
	if err := binary.Read(reader, binary.BigEndian, &header); err != nil {
		return nil, Info{}, err
	}
	if err := validateXWD(header); err != nil {
		return nil, Info{}, err
	}
	if _, err := io.CopyN(io.Discard, reader, int64(header[0]-100)); err != nil {
		return nil, Info{}, err
	}
	if err := xwdColors(reader, header); err != nil {
		return nil, Info{}, err
	}
	info := Info{Width: int(header[4]), Height: int(header[5]), Name: "X11 desktop"}
	frame := image.NewRGBA(image.Rect(0, 0, info.Width, info.Height))
	row := make([]byte, int(header[12]))
	for y := 0; y < info.Height; y++ {
		if _, err := io.ReadFull(reader, row); err != nil {
			return nil, Info{}, err
		}
		xwdRow(frame.Pix[y*frame.Stride:(y+1)*frame.Stride], row, int(header[11]/8), header[7])
	}
	data, err := PNG(frame)
	return data, info, err
}

func validateXWD(h [25]uint32) error {
	if h[0] < 100 || h[0] > 4196 || h[1] != 7 || h[2] != 2 || h[3] != 24 || h[6] != 0 || h[7] > 1 || (h[13] != 4 && h[13] != 5) {
		return fmt.Errorf("unsupported XWD header; require version 7, 24-bit RGB ZPixmap")
	}
	if h[4] == 0 || h[5] == 0 || h[4] > 8192 || h[5] > 8192 || uint64(h[4])*uint64(h[5]) > 8388608 {
		return fmt.Errorf("X11 desktop exceeds screenshot dimension limits")
	}
	if (h[11] != 24 && h[11] != 32) || h[12] < h[4]*(h[11]/8) || h[12] > h[4]*4+4 || h[14] != 0xff0000 || h[15] != 0xff00 || h[16] != 0xff || h[19] > 65536 {
		return fmt.Errorf("unsupported XWD pixel layout or color table")
	}
	return nil
}

func xwdColors(reader io.Reader, h [25]uint32) error {
	if h[13] == 4 {
		_, err := io.CopyN(io.Discard, reader, int64(h[19])*12)
		return err
	}
	// xwd synthesizes this identity DirectColor table when composing mixed visuals.
	// 中文：xwd 合成多种视觉类窗口时生成恒等 DirectColor 表，必须验证后才能按 RGB 解码。
	if h[17] != 8 || h[18] != 256 || h[19] != 256 {
		return fmt.Errorf("unsupported XWD DirectColor table")
	}
	var seen [256]bool
	for range 256 {
		var entry [12]byte
		if _, err := io.ReadFull(reader, entry[:]); err != nil {
			return err
		}
		pixel := binary.BigEndian.Uint32(entry[:4])
		index := byte(pixel)
		value := uint16(index) * 257
		if pixel != uint32(index)*0x010101 || seen[index] ||
			binary.BigEndian.Uint16(entry[4:6]) != value ||
			binary.BigEndian.Uint16(entry[6:8]) != value ||
			binary.BigEndian.Uint16(entry[8:10]) != value {
			return fmt.Errorf("XWD DirectColor requires a complete identity color table")
		}
		seen[index] = true
	}
	return nil
}

func xwdRow(target, source []byte, stride int, order uint32) {
	for x := 0; x < len(target)/4; x++ {
		pixel := source[x*stride : (x+1)*stride]
		var r, g, b byte
		if order == 0 {
			b, g, r = pixel[0], pixel[1], pixel[2]
		} else {
			r, g, b = pixel[stride-3], pixel[stride-2], pixel[stride-1]
		}
		copy(target[x*4:], []byte{r, g, b, 255})
	}
}
