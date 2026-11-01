// Created by Yanjunhui
package vnc

import (
	"bytes"
	"encoding/binary"
	"image/png"
	"strings"
	"testing"
)

func TestFullFrameCoverageAndResize(t *testing.T) {
	first := append([]byte{0, 0, 0, 1}, rectangle(0, 0, 1, 1, 0, []byte{0, 0, 255, 0})...)
	last := append([]byte{0, 0, 0, 1}, rectangle(1, 0, 1, 1, 0, []byte{0, 255, 0, 0})...)
	resize := append([]byte{0, 0, 0, 1}, rectangle(0, 0, 2, 1, -223, nil)...)
	for _, tc := range []struct {
		name  string
		width uint16
		data  []byte
		valid bool
	}{
		{"partial", 2, first, false},
		{"overlap", 2, append(append([]byte{}, first...), first...), false},
		{"complete", 2, append(append([]byte{}, first...), last...), true},
		{"resize", 1, append(append(resize, first...), last...), true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := New(serverTranscript("RFB 003.008\n", tc.width, 1, tc.data), "")
			if err != nil {
				t.Fatal(err)
			}
			data, err := c.Screenshot()
			if !tc.valid {
				if err == nil || len(data) != 0 {
					t.Fatal("partial frame returned")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(bytes.NewReader(data))
			if err != nil || img.Bounds().Dx() != 2 || c.Info().Width != 2 {
				t.Fatal("resize metadata or PNG incorrect", err)
			}
			_, g, _, a := img.At(1, 0).RGBA()
			if g != 65535 || a != 65535 {
				t.Fatal("incomplete second pixel")
			}
		})
	}
}

func TestPasswordHandshakeVersions(t *testing.T) {
	challenge := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	for _, version := range []string{"RFB 003.003\n", "RFB 003.007\n", "RFB 003.008\n"} {
		for _, rejected := range []bool{false, true} {
			b := bytes.NewBufferString(version)
			if version == "RFB 003.003\n" {
				b.Write([]byte{0, 0, 0, 2})
			} else {
				b.Write([]byte{2, 1, 2})
			}
			b.Write(challenge)
			status := uint32(0)
			if rejected {
				status = 1
			}
			_ = binary.Write(b, binary.BigEndian, status)
			initial := serverTranscript("RFB 003.003\n", 1, 1, nil)
			b.Write(initial.inputBytes(t)[16:])
			wire := &transcript{input: bytes.NewReader(b.Bytes())}
			c, err := New(wire, "password")
			if rejected {
				if err == nil {
					t.Fatal("rejected password accepted")
				}
				continue
			}
			if err != nil || c.Info().Width != 1 {
				t.Fatal("password handshake failed", version, err)
			}
			response, _ := challengeResponse("password", challenge)
			if !bytes.Contains(wire.output.Bytes(), response) {
				t.Fatal("challenge response not sent")
			}
		}
	}
}

func (s *transcript) inputBytes(t *testing.T) []byte {
	t.Helper()
	b := make([]byte, s.input.Len())
	if _, err := s.input.Read(b); err != nil {
		t.Fatal(err)
	}
	return b
}

func TestInputBoundsBeforeSending(t *testing.T) {
	for _, input := range []string{"", strings.Repeat("x", 4097), "\xff", "\x00", "\x1b"} {
		if ValidateText(input) == nil {
			t.Fatal("invalid text accepted")
		}
	}
	if ValidateText(strings.Repeat("中", 4096)) != nil {
		t.Fatal("valid maximum text rejected")
	}
	for _, keys := range [][]string{nil, make([]string, 17), {"F13"}, {"unknown"}} {
		if _, err := ValidateKeys(keys); err == nil {
			t.Fatal("invalid chord accepted")
		}
	}
	for _, events := range [][]PointerEvent{nil, make([]PointerEvent, 257), {{X: -1}}, {{Y: 8192}}, {{Buttons: 32}}} {
		if ValidatePointer(events) == nil {
			t.Fatal("invalid pointer accepted")
		}
	}
	for _, password := range []string{"123456789", "中文", "bad\n"} {
		if _, err := New(&transcript{input: bytes.NewReader(nil)}, password); err == nil {
			t.Fatal("invalid password accepted")
		}
	}
	s := serverTranscript("RFB 003.008\n", 100, 100, nil)
	c, err := New(s, "")
	if err != nil {
		t.Fatal(err)
	}
	s.output.Reset()
	if err := c.Pointer([]PointerEvent{{X: 1, Y: 1, Buttons: 1}, {X: 100}}); err == nil || s.output.Len() != 0 {
		t.Fatal("invalid batch partly sent")
	}
	if err := c.Type("\r\n\t"); err != nil || s.output.Len() != 32 {
		t.Fatal("CRLF did not produce one Return", err)
	}
}

func FuzzHandshake(f *testing.F) {
	f.Add([]byte("RFB 003.008\n\x00"))
	f.Add([]byte("RFB 003.003\n\x00\x00\x00\x01"))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = New(&transcript{input: bytes.NewReader(data)}, "")
	})
}
