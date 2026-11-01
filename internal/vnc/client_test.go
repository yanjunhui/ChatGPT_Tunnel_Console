// Created by Yanjunhui
package vnc

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"image/png"
	"io"
	"testing"
)

type transcript struct {
	input  *bytes.Reader
	output bytes.Buffer
}

func (s *transcript) Read(p []byte) (int, error)  { return s.input.Read(p) }
func (s *transcript) Write(p []byte) (int, error) { return s.output.Write(p) }

func serverTranscript(version string, width, height uint16, tail []byte) *transcript {
	b := bytes.NewBufferString(version)
	if version == "RFB 003.003\n" {
		b.Write([]byte{0, 0, 0, 1})
	} else {
		b.Write([]byte{1, 1})
		if version == "RFB 003.008\n" {
			b.Write(make([]byte, 4))
		}
	}
	_ = binary.Write(b, binary.BigEndian, width)
	_ = binary.Write(b, binary.BigEndian, height)
	b.Write(make([]byte, 16))
	_ = binary.Write(b, binary.BigEndian, uint32(4))
	b.WriteString("test")
	b.Write(tail)
	return &transcript{input: bytes.NewReader(b.Bytes())}
}

func rectangle(x, y, w, h uint16, encoding int32, pixels []byte) []byte {
	b := new(bytes.Buffer)
	for _, v := range []uint16{x, y, w, h} {
		_ = binary.Write(b, binary.BigEndian, v)
	}
	_ = binary.Write(b, binary.BigEndian, encoding)
	b.Write(pixels)
	return b.Bytes()
}

func TestScreenshotVersionsAndPixels(t *testing.T) {
	for _, version := range []string{"RFB 003.003\n", "RFB 003.007\n", "RFB 003.008\n"} {
		t.Run(version, func(t *testing.T) {
			update := append([]byte{0, 0, 0, 1}, rectangle(0, 0, 2, 1, 0, []byte{0, 0, 255, 0, 0, 255, 0, 0})...)
			s := serverTranscript(version, 2, 1, update)
			c, err := New(s, "")
			if err != nil {
				t.Fatal(err)
			}
			data, err := c.Screenshot()
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			r, g, b, a := img.At(0, 0).RGBA()
			if r != 65535 || g != 0 || b != 0 || a != 65535 || img.Bounds().Dx() != 2 {
				t.Fatal("incorrect pixels")
			}
			if c.Info().Name != "test" || s.input.Len() != 0 {
				t.Fatal("incomplete protocol")
			}
		})
	}
}

func TestAuthenticationChallenge(t *testing.T) {
	challenge := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	got, err := challengeResponse("password", challenge)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := hex.DecodeString("b866924125c8eebb9debc1db61c538e2")
	if !bytes.Equal(got, want) {
		t.Fatalf("challenge response %x", got)
	}
}

func TestRejectMalformedHandshake(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("RFB 003.889\n"), []byte("RFB 003.008\n\x00"), []byte("RFB 003.008\n\x01\x13")} {
		if _, err := New(&transcript{input: bytes.NewReader(data)}, ""); err == nil {
			t.Fatal("accepted invalid handshake")
		}
	}
	for _, size := range [][2]uint16{{0, 1}, {1, 0}, {65535, 65535}, {4096, 4096}} {
		if _, err := New(serverTranscript("RFB 003.008\n", size[0], size[1], nil), ""); err == nil {
			t.Fatal("accepted oversized framebuffer")
		}
	}
	if _, err := New(serverTranscript("RFB 003.008\n", 1, 1, nil), "password"); err == nil {
		t.Fatal("silently downgraded password auth")
	}
}

func TestRejectInvalidRectangles(t *testing.T) {
	for _, rect := range [][]byte{rectangle(2, 0, 1, 1, 0, make([]byte, 4)), rectangle(0, 0, 1, 1, 99, nil), rectangle(0, 0, 2, 1, 0, []byte{1})} {
		c, err := New(serverTranscript("RFB 003.008\n", 2, 1, append([]byte{0, 0, 0, 1}, rect...)), "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err = c.Screenshot(); err == nil {
			t.Fatal("accepted malformed rectangle")
		}
	}
}

func TestInputValidationAndRelease(t *testing.T) {
	s := serverTranscript("RFB 003.008\n", 100, 100, nil)
	c, err := New(s, "")
	if err != nil {
		t.Fatal(err)
	}
	s.output.Reset()
	if err = c.Pointer([]desktop.PointerEvent{{X: 100, Y: 0}}); err == nil || s.output.Len() != 0 {
		t.Fatal("invalid input sent")
	}
	if err = c.Pointer([]desktop.PointerEvent{{X: 10, Y: 20, Buttons: 1}, {X: 30, Y: 40, Buttons: 1}}); err != nil {
		t.Fatal(err)
	}
	want := []byte{5, 1, 0, 10, 0, 20, 5, 1, 0, 30, 0, 40, 5, 0, 0, 30, 0, 40}
	if !bytes.Equal(s.output.Bytes(), want) {
		t.Fatal("pointer release missing")
	}
	s.output.Reset()
	if err = c.Key([]string{"Control_L", "l"}); err != nil {
		t.Fatal(err)
	}
	if s.output.Len() != 32 || s.output.Bytes()[17] != 0 || s.output.Bytes()[25] != 0 {
		t.Fatal("key release missing")
	}
	s.output.Reset()
	if err = c.Type("你好\n"); err != nil || s.output.Len() != 48 {
		t.Fatal("unicode text", err)
	}
	if _, err = io.ReadAll(s.input); err != nil {
		t.Fatal(err)
	}
}
