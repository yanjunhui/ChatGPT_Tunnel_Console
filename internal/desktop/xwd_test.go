// Created by Yanjunhui
package desktop

import (
	"bytes"
	"encoding/binary"
	"image/color"
	"image/png"
	"testing"
)

func xwdFixture(order, bits uint32) []byte {
	h := [25]uint32{104, 7, 2, 24, 2, 1, 0, order, 32, 0, 32, bits, 8, 4, 0xff0000, 0xff00, 0xff, 8, 256, 1, 2, 1}
	var b bytes.Buffer
	_ = binary.Write(&b, binary.BigEndian, h)
	b.Write(make([]byte, 4+12))
	row := make([]byte, 8)
	for x := 0; x < 2; x++ {
		pixel := row[x*int(bits/8) : (x+1)*int(bits/8)]
		if order == 0 {
			copy(pixel, []byte{3, 2, 1})
		} else {
			copy(pixel[len(pixel)-3:], []byte{1, 2, 3})
		}
	}
	b.Write(row)
	return b.Bytes()
}

func TestXWDOriginalPixelsAndPadding(t *testing.T) {
	for _, order := range []uint32{0, 1} {
		for _, bits := range []uint32{24, 32} {
			data, info, err := XWD(bytes.NewReader(xwdFixture(order, bits)))
			if err != nil || info.Width != 2 || info.Height != 1 {
				t.Fatal(info, err)
			}
			frame, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			for x := 0; x < 2; x++ {
				if color.RGBAModel.Convert(frame.At(x, 0)) != (color.RGBA{1, 2, 3, 255}) {
					t.Fatal("pixel changed", order, bits, x)
				}
			}
		}
	}
}

func TestXWDRejectsInvalidAndTruncatedFrames(t *testing.T) {
	for _, item := range [][2]uint32{{0, 99}, {0, 4197}, {1, 6}, {2, 1}, {3, 32}, {4, 0}, {4, 8193}, {5, 8388608}, {6, 1}, {7, 2}, {11, 16}, {12, 1}, {12, 999999}, {13, 3}, {14, 0}, {19, 65537}} {
		data := xwdFixture(0, 32)
		binary.BigEndian.PutUint32(data[item[0]*4:], item[1])
		if output, _, err := XWD(bytes.NewReader(data)); err == nil || output != nil {
			t.Fatal("accepted invalid header", item)
		}
	}
	data := xwdFixture(0, 32)
	for n := 0; n < len(data); n++ {
		if output, _, err := XWD(bytes.NewReader(data[:n])); err == nil || output != nil {
			t.Fatal("accepted truncated frame", n)
		}
	}
}

func FuzzXWD(f *testing.F) {
	f.Add(xwdFixture(0, 32))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, data []byte) { _, _, _ = XWD(bytes.NewReader(data)) })
}
