// Created by Yanjunhui
package attachment

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"testing"
)

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func fixture(t *testing.T, handler http.HandlerFunc) *http.Client {
	t.Helper()
	s := httptest.NewTLSServer(handler)
	t.Cleanup(s.Close)
	target, _ := url.Parse(s.URL)
	base := s.Client().Transport
	return &http.Client{CheckRedirect: redirect, Transport: transportFunc(func(r *http.Request) (*http.Response, error) {
		copy := r.Clone(r.Context())
		copy.URL.Host = target.Host
		return base.RoundTrip(copy)
	})}
}

func TestDownloadBytesAndPermissions(t *testing.T) {
	for _, data := range [][]byte{nil, []byte("中文\n"), {0, 255, 1}, bytes.Repeat([]byte{7}, 4096)} {
		client := fixture(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(data) })
		f, err := download(t.Context(), File{DownloadURL: "https://files.oaiusercontent.com/input?secret=hidden", FileID: "file-test"}, client, 4096)
		if err != nil {
			t.Fatal(err)
		}
		got, err := io.ReadAll(f)
		stat, _ := f.Stat()
		_ = f.Close()
		_ = os.Remove(f.Name())
		if err != nil || !bytes.Equal(got, data) || stat.Mode().Perm() != 0600 {
			t.Fatalf("invalid staged bytes/mode: %v", err)
		}
	}
}

func TestDownloadFailuresCleanAndRedact(t *testing.T) {
	t.Setenv("TMPDIR", t.TempDir())
	cases := []struct {
		name    string
		handler http.HandlerFunc
	}{
		{"expired", func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "secret body", 403) }},
		{"declared-limit", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "100")
			_, _ = w.Write([]byte("x"))
		}},
		{"chunked-limit", func(w http.ResponseWriter, _ *http.Request) {
			w.(http.Flusher).Flush()
			_, _ = w.Write([]byte("12345"))
		}},
		{"truncated", func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "4")
			_, _ = w.Write([]byte("x"))
		}},
		{"redirect-private", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "https://127.0.0.1/secret", 302) }},
		{"redirect-loop", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://files.oaiusercontent.com/again", 302)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := download(t.Context(), File{DownloadURL: "https://files.oaiusercontent.com/f?secret=hidden", FileID: "file-test"}, fixture(t, tc.handler), 4)
			if err == nil || strings.Contains(err.Error(), "hidden") || strings.Contains(err.Error(), "secret body") {
				t.Fatalf("unsafe error: %v", err)
			}
			entries, _ := os.ReadDir(os.TempDir())
			if len(entries) != 0 {
				t.Fatal("staging file leaked")
			}
		})
	}
}

func TestRedirectDropsRefererAndCancellation(t *testing.T) {
	client := fixture(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "https://files.oaiusercontent.com/end", 302)
			return
		}
		if r.Header.Get("Referer") != "" {
			t.Error("signed Referer leaked")
		}
		_, _ = w.Write([]byte("ok"))
	})
	f, err := download(t.Context(), File{DownloadURL: "https://files.oaiusercontent.com/start?secret=hidden", FileID: "file-test"}, client, 4)
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	_ = os.Remove(f.Name())
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = download(ctx, File{DownloadURL: "https://files.oaiusercontent.com/start?secret=hidden", FileID: "file-test"}, client, 4)
	if err == nil || strings.Contains(err.Error(), "hidden") {
		t.Fatal(err)
	}
}

func TestSourceValidationAndPublicAddresses(t *testing.T) {
	for _, raw := range []string{"", "http://files.oaiusercontent.com/x", "https://files.oaiusercontent.com.evil.test/x", "https://user@files.oaiusercontent.com/x", "https://files.oaiusercontent.com:444/x", "https://127.0.0.1/x", "https://files.oaiusercontent.com/x#fragment"} {
		if validateURL(raw) == nil {
			t.Errorf("accepted %q", raw)
		}
	}
	for _, raw := range []string{"https://files.oaiusercontent.com/x", "https://store.blob.core.windows.net/x", "https://bucket.s3.amazonaws.com:443/x"} {
		if err := validateURL(raw); err != nil {
			t.Error(err)
		}
	}
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "::1", "::ffff:127.0.0.1", "169.254.169.254", "fc00::1", "100.64.0.1", "198.18.0.1", "224.0.0.1", "2001:db8::1"} {
		if publicIP(netip.MustParseAddr(raw)) {
			t.Errorf("accepted nonpublic %s", raw)
		}
	}
	if !publicIP(netip.MustParseAddr("1.1.1.1")) {
		t.Fatal("public IP rejected")
	}
	client := &http.Client{Transport: transportFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("secret=hidden") })}
	_, err := download(t.Context(), File{DownloadURL: "https://files.oaiusercontent.com/x", FileID: "file-test"}, client, 4)
	if err == nil || strings.Contains(err.Error(), "hidden") {
		t.Fatal(err)
	}
	_, err = download(t.Context(), File{DownloadURL: "https://files.oaiusercontent.com/x"}, client, 4)
	if err == nil {
		t.Fatal("missing file ID accepted")
	}
}
