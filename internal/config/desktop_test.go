// Created by Yanjunhui
package config

import "testing"

func TestNativeDesktopConfiguration(t *testing.T) {
	c, err := loadText(t, valid+"[profiles.test.desktop]\nenabled = true\n")
	if err != nil {
		t.Fatal(err)
	}
	p := c.Profiles["test"]
	if p.Desktop == nil || !p.Desktop.Enabled || p.Desktop.Display != ":1" || p.Desktop.Timeout != "15s" || p.VNC != nil {
		t.Fatal("native defaults or independence missing")
	}
	for _, value := range []string{"display = ':1;id'", "display = 'host:1'", "display = ':100000'", "timeout = '0s'", "timeout = '-1s'", "timeout = '121s'", "timeout = 'invalid'"} {
		if _, err := loadText(t, valid+"[profiles.test.desktop]\nenabled = true\n"+value+"\n"); err == nil {
			t.Fatal("invalid desktop accepted", value)
		}
	}
	for _, display := range []string{":0", ":0.0", ":99999.999"} {
		if _, err := loadText(t, valid+"[profiles.test.desktop]\ndisplay = '"+display+"'\ntimeout = '120s'\n"); err != nil {
			t.Fatal(err)
		}
	}
}
