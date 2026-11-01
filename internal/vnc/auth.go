// Created by Yanjunhui
package vnc

import (
	"crypto/des"
	"encoding/binary"
	"fmt"
	"math/bits"
)

func validatePassword(password string) error {
	if len(password) > 8 {
		return fmt.Errorf("classic VNC passwords must be at most eight ASCII bytes")
	}
	for _, b := range []byte(password) {
		if b < 32 || b > 126 {
			return fmt.Errorf("classic VNC passwords must use printable ASCII")
		}
	}
	return nil
}

func (c *Client) authenticate(version, password string) error {
	want := byte(1)
	if password != "" {
		want = 2
	}
	if err := c.selectSecurity(version, want); err != nil {
		return err
	}
	if want == 2 {
		challenge, err := c.read(16)
		if err != nil {
			return err
		}
		response, err := challengeResponse(password, challenge)
		if err != nil {
			return err
		}
		if err = c.write(response); err != nil {
			return err
		}
	}
	if want == 2 || version == "RFB 003.008\n" {
		result, err := c.read(4)
		if err != nil {
			return err
		}
		if binary.BigEndian.Uint32(result) != 0 {
			return fmt.Errorf("VNC authentication rejected")
		}
	}
	return nil
}

func (c *Client) selectSecurity(version string, want byte) error {
	if version == "RFB 003.003\n" {
		security, err := c.read(4)
		if err != nil {
			return err
		}
		if binary.BigEndian.Uint32(security) != uint32(want) {
			return fmt.Errorf("VNC server does not offer the configured authentication method")
		}
		return nil
	}
	n, err := c.read(1)
	if err != nil {
		return err
	}
	types, err := c.read(int(n[0]))
	if err != nil {
		return err
	}
	for _, security := range types {
		if security == want {
			return c.write([]byte{want})
		}
	}
	return fmt.Errorf("VNC server does not offer the configured authentication method; use None or VncAuth over SSH")
}

func challengeResponse(password string, challenge []byte) ([]byte, error) {
	if len(challenge) != 16 {
		return nil, fmt.Errorf("VNC challenge must be 16 bytes")
	}
	if err := validatePassword(password); err != nil {
		return nil, err
	}
	// Classic VNC reverses each password byte's bit order before legacy DES.
	// SSH supplies transport encryption; this challenge does not replace it.
	// 中文：传统 VNC 在 DES 前逐字节反转密码位序；传输加密由 SSH 提供。
	var key [8]byte
	for i, b := range []byte(password) {
		key[i] = bits.Reverse8(b)
	}
	block, err := des.NewCipher(key[:])
	if err != nil {
		return nil, err
	}
	response := make([]byte, 16)
	block.Encrypt(response[:8], challenge[:8])
	block.Encrypt(response[8:], challenge[8:])
	return response, nil
}
