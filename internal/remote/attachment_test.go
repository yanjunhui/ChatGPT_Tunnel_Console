// Created by Yanjunhui
package remote

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/attachment"
)

func TestAttachmentPublishesAndCleans(t *testing.T) {
	m, fixture := newTestManager(t)
	for _, data := range []string{"", "中文附件\n", string([]byte{0, 255, 1})} {
		var staged string
		fetch := func(context.Context, attachment.File) (*os.File, error) {
			f, err := os.CreateTemp(t.TempDir(), "attachment-")
			if err != nil {
				return nil, err
			}
			staged = f.Name()
			_, _ = f.WriteString(data)
			_, _ = f.Seek(0, 0)
			return f, nil
		}
		target := filepath.Join(fixture.Root, "attachment.bin")
		if err := os.WriteFile(target, []byte("original"), 0644); err != nil {
			t.Fatal(err)
		}
		out, err := m.uploadSource(t.Context(), UploadFileInput{FileInput: FileInput{Path: "attachment.bin"}, File: &attachment.File{}}, fetch)
		if err != nil || out.Bytes != int64(len(data)) {
			t.Fatalf("upload: %+v %v", out, err)
		}
		got, _ := os.ReadFile(target)
		stat, _ := os.Stat(target)
		if string(got) != data || stat.Mode().Perm() != 0600 {
			t.Fatal("attachment content or permissions differ")
		}
		if _, err := os.Stat(staged); !os.IsNotExist(err) {
			t.Fatal("local staging leaked")
		}
		pending, _ := filepath.Glob(filepath.Join(fixture.Root, ".tunnel-console-upload-*"))
		if len(pending) != 0 {
			t.Fatal("remote staging leaked")
		}
	}
}

func TestAttachmentValidationAndFailuresPreserveTarget(t *testing.T) {
	m, fixture := newTestManager(t)
	target := filepath.Join(fixture.Root, "existing")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	fetch := func(context.Context, attachment.File) (*os.File, error) {
		called = true
		return nil, errors.New("download failed")
	}
	for _, in := range []UploadFileInput{
		{FileInput: FileInput{Path: "existing"}},
		{FileInput: FileInput{Path: "existing"}, File: &attachment.File{}, LocalPath: "x"},
		{FileInput: FileInput{Path: ""}, File: &attachment.File{}},
		{FileInput: FileInput{Path: "existing", Profile: "missing"}, File: &attachment.File{}},
	} {
		if _, err := m.uploadSource(t.Context(), in, fetch); err == nil {
			t.Fatal("invalid source accepted")
		}
	}
	if called {
		t.Fatal("fetched before validation")
	}
	_, err := m.uploadSource(t.Context(), UploadFileInput{FileInput: FileInput{Path: "existing"}, File: &attachment.File{}}, fetch)
	got, _ := os.ReadFile(target)
	if err == nil || string(got) != "original" {
		t.Fatal("download failure touched target")
	}
}

// interruptedReader simulates failure after some remote bytes have been staged.
// 中文：interruptedReader 模拟远端暂存部分字节后的读取失败。
type interruptedReader struct{ sent bool }

func (r *interruptedReader) Read(p []byte) (int, error) {
	if r.sent {
		return 0, io.ErrUnexpectedEOF
	}
	r.sent = true
	return copy(p, "partial"), nil
}

func TestAttachmentCopyAndRenameFailure(t *testing.T) {
	m, fixture := newTestManager(t)
	target := filepath.Join(fixture.Root, "existing")
	if err := os.WriteFile(target, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	sc, destination, cleanup, err := m.sftp(t.Context(), FileInput{Path: "existing"})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err = publishAttachment(sc, destination, &interruptedReader{}); err == nil {
		t.Fatal("copy failure ignored")
	}
	got, _ := os.ReadFile(target)
	if string(got) != "original" {
		t.Fatal("partial copy replaced target")
	}
	directory := filepath.Join(fixture.Root, "directory")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if _, err = publishAttachment(sc, directory, strings.NewReader("new")); err == nil {
		t.Fatal("rename failure ignored")
	}
	pending, _ := filepath.Glob(filepath.Join(fixture.Root, ".tunnel-console-upload-*"))
	if len(pending) != 0 {
		t.Fatal("failed remote staging leaked")
	}
}
