// Created by Yanjunhui
package config

import (
	"path/filepath"
	"testing"
)

func TestVNCDefaultsAndCredentialPath(t *testing.T) {
	c, err := loadText(t, valid+"[profiles.test.vnc]\nenabled = true\npassword = 'file:keys/vnc'\n")
	if err != nil {
		t.Fatal(err)
	}
	v := c.Profiles["test"].VNC
	if v == nil || !v.Enabled || v.Host != "127.0.0.1" || v.Port != 5900 || v.Timeout != "15s" || !filepath.IsAbs(v.Password[5:]) {
		t.Fatalf("unexpected VNC configuration: %+v", v)
	}
}

func TestVNCRejectsInvalidConfiguration(t *testing.T) {
	for _, value := range []string{"port = -1", "port = 65536", "timeout = '0s'", "timeout = '121s'", "timeout = '-1s'", "host = 'host:5900'", "password = 'plaintext'"} {
		t.Run(value, func(t *testing.T) {
			if _, err := loadText(t, valid+"[profiles.test.vnc]\nenabled = true\n"+value+"\n"); err == nil {
				t.Fatal("accepted invalid VNC configuration")
			}
		})
	}
}

func TestOldProfilesLeaveVNCDisabled(t *testing.T) {
	c, err := loadText(t, valid)
	if err != nil || c.Profiles["test"].VNC != nil {
		t.Fatalf("legacy configuration changed: %v", err)
	}
}
