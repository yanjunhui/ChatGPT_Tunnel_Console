// Created by Yanjunhui
package tunnel

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Release pins the default official tunnel-client bundle independently of the MCP version.
// 中文：Release 固定默认官方隧道客户端版本，与 MCP 服务版本分别管理。
const Release = "v0.0.19"

// Install downloads the official bundle and verifies its published SHA-256.
// It supports macOS/Linux on amd64/arm64 and returns download, checksum or file errors.
// 中文：Install 下载并校验官方发布包，支持 macOS/Linux 的 amd64/arm64，
// 下载、校验或文件操作失败时返回错误。
func Install(ctx context.Context, directory, version string) error {
	if !regexp.MustCompile(`^v\d+\.\d+\.\d+$`).MatchString(version) {
		return fmt.Errorf("invalid release version")
	}
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return fmt.Errorf("supported platforms: Linux and macOS")
	}
	if runtime.GOARCH != "amd64" && runtime.GOARCH != "arm64" {
		return fmt.Errorf("supported architectures: amd64 and arm64")
	}
	if err := os.MkdirAll(directory, 0755); err != nil {
		return err
	}
	temp, err := os.MkdirTemp(directory, ".tunnel-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(temp)
	asset := fmt.Sprintf("tunnel-client-%s-%s-%s.zip", version, runtime.GOOS, runtime.GOARCH)
	base := "https://github.com/openai/tunnel-client/releases/download/" + version + "/"
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	archive := filepath.Join(temp, asset)
	manifest := filepath.Join(temp, "SHA256SUMS.txt")
	if err = download(ctx, base+asset, archive); err != nil {
		return err
	}
	if err = download(ctx, base+"SHA256SUMS.txt", manifest); err != nil {
		return err
	}
	if err = verify(archive, manifest, asset); err != nil {
		return err
	}
	if err = unpack(archive, directory); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "Installed official tunnel-client %s in %s (SHA-256 verified)\n", version, directory)
	return nil
}

func download(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("download official tunnel release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download official tunnel release: HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, resp.Body)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func verify(archive, manifest, asset string) error {
	data, err := os.ReadFile(manifest)
	if err != nil {
		return err
	}
	expected := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(strings.TrimPrefix(fields[1], "*"), "./") == asset {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if len(expected) != 64 {
		return fmt.Errorf("checksum manifest has no valid entry for requested asset")
	}
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err = io.Copy(hash, f); err != nil {
		return err
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return fmt.Errorf("tunnel release SHA-256 mismatch")
	}
	return nil
}

func unpack(archive, directory string) error {
	z, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer z.Close()
	found := false
	for _, entry := range z.File {
		name := filepath.Base(entry.Name)
		if entry.FileInfo().IsDir() {
			continue
		}
		if name != "tunnel-client" && name != "cloudflared" {
			continue
		}
		if name == "tunnel-client" {
			if found {
				return fmt.Errorf("ambiguous tunnel-client bundle")
			}
			found = true
		}
		if err = extract(entry, filepath.Join(directory, name)); err != nil {
			return err
		}
	}
	if !found {
		return fmt.Errorf("release bundle does not contain tunnel-client")
	}
	return nil
}

func extract(entry *zip.File, destination string) error {
	source, err := entry.Open()
	if err != nil {
		return err
	}
	defer source.Close()
	f, err := os.CreateTemp(filepath.Dir(destination), ".extract-*")
	if err != nil {
		return err
	}
	defer os.Remove(f.Name())
	_, copyErr := io.Copy(f, source)
	modeErr := f.Chmod(0755)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if modeErr != nil {
		return modeErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(f.Name(), destination)
}
