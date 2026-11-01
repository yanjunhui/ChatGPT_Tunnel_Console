// Created by Yanjunhui
package remote

import (
	"bytes"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSFTPTextAndBinary(t *testing.T) {
	m, _ := newTestManager(t)
	for _, data := range [][]byte{nil, []byte("中文\nhello"), {0, 1, 255, 254}} {
		in := FileInput{Path: "roundtrip.bin"}
		encoded := base64.StdEncoding.EncodeToString(data)
		up, err := m.Upload(t.Context(), UploadInput{FileInput: in, Content: encoded, Encoding: "base64", Mode: 0600})
		if err != nil || up.Bytes != int64(len(data)) {
			t.Fatalf("upload %+v %v", up, err)
		}
		down, err := m.Download(t.Context(), DownloadInput{FileInput: in, Encoding: "base64"})
		if err != nil || down.Content != encoded {
			t.Fatalf("download %+v %v", down, err)
		}
	}
	entries, err := m.ListFiles(t.Context(), FileInput{Path: "."})
	if err != nil || len(entries) < 2 {
		t.Fatalf("listing %+v %v", entries, err)
	}
}

func TestSFTPStreamingHasNoInlineLimit(t *testing.T) {
	m, _ := newTestManager(t)
	data := bytes.Repeat([]byte{0, 255, 3}, 32768)
	dir := t.TempDir()
	source, dest := filepath.Join(dir, "source"), filepath.Join(dir, "dest")
	if err := os.WriteFile(source, data, 0600); err != nil {
		t.Fatal(err)
	}
	in := FileInput{Path: "big.bin"}
	if _, err := m.UploadFile(t.Context(), TransferInput{FileInput: in, LocalPath: source}); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Download(t.Context(), DownloadInput{FileInput: in}); err == nil {
		t.Fatal("inline overflow not reported")
	}
	if _, err := m.DownloadFile(t.Context(), TransferInput{FileInput: in, LocalPath: dest}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dest)
	if err != nil || !bytes.Equal(got, data) {
		t.Fatal("streamed bytes differ", err)
	}
}

func TestSFTPFailuresPreserveExistingData(t *testing.T) {
	m, s := newTestManager(t)
	path := filepath.Join(s.Root, "existing")
	if err := os.WriteFile(path, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := m.Upload(t.Context(), UploadInput{FileInput: FileInput{Path: "existing"}, Content: "!bad!", Encoding: "base64"}); err == nil {
		t.Fatal("accepted bad base64")
	}
	if _, err := m.DownloadFile(t.Context(), TransferInput{FileInput: FileInput{Path: "missing"}, LocalPath: path}); err == nil {
		t.Fatal("downloaded missing file")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "original" {
		t.Fatal("failed transfer overwrote file")
	}
	for _, input := range []FileInput{{Path: ""}, {Profile: "missing", Path: "."}} {
		if _, err := m.ListFiles(t.Context(), input); err == nil {
			t.Fatal("accepted invalid path/profile")
		}
	}
}

func TestSFTPUnrestrictedPathAndInvalidUTF8(t *testing.T) {
	m, s := newTestManager(t)
	directory := t.TempDir()
	full := filepath.Join(directory, "outside-profile")
	if _, err := m.Upload(t.Context(), UploadInput{FileInput: FileInput{Path: full}, Content: "outside"}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(full)
	if string(data) != "outside" {
		t.Fatal("absolute path upload failed")
	}
	if err := os.WriteFile(filepath.Join(s.Root, "invalid"), []byte{255}, 0600); err != nil {
		t.Fatal(err)
	}
	_, err := m.Download(t.Context(), DownloadInput{FileInput: FileInput{Path: "invalid"}})
	if err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatal("binary file silently corrupted", err)
	}
}
