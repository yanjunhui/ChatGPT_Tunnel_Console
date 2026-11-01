// Created by Yanjunhui
package remote

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/pkg/sftp"
)

// FileInput uses SSH-host paths; relative paths resolve against the profile workdir.
// 中文：FileInput 使用 SSH 主机路径，相对路径以配置的工作目录为基准。
type FileInput struct {
	Profile string `json:"profile,omitempty"`
	Path    string `json:"path" jsonschema:"Remote path; relative to the profile workdir when configured"`
}

// FileEntry contains SFTP directory metadata; Mode is a human-readable mode string.
// 中文：FileEntry 包含 SFTP 目录项元数据，Mode 为可读的权限字符串。
type FileEntry struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	Mode      string    `json:"mode"`
	Modified  time.Time `json:"modified"`
	Directory bool      `json:"directory"`
}

func (m *Manager) sftp(ctx context.Context, in FileInput) (*sftp.Client, string, func(), error) {
	_, p, err := m.config.Profile(in.Profile)
	if err != nil {
		return nil, "", nil, err
	}
	if in.Path == "" {
		return nil, "", nil, fmt.Errorf("remote path must not be empty")
	}
	client, err := dial(ctx, p)
	if err != nil {
		return nil, "", nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	cleanup := func() { stop(); _ = client.Close() }
	sc, err := sftp.NewClient(client)
	if err != nil {
		cleanup()
		return nil, "", nil, fmt.Errorf("start SFTP: %w", err)
	}
	remotePath := in.Path
	if !path.IsAbs(remotePath) && p.Workdir != "" {
		remotePath = path.Join(p.Workdir, remotePath)
	}
	return sc, remotePath, func() { _ = sc.Close(); cleanup() }, nil
}

// ListFiles reads directory entries or returns a profile, connection or SFTP error.
// 中文：ListFiles 读取目录项，配置、连接或 SFTP 失败时返回错误。
func (m *Manager) ListFiles(ctx context.Context, in FileInput) ([]FileEntry, error) {
	sc, remotePath, cleanup, err := m.sftp(ctx, in)
	if err != nil {
		return nil, err
	}
	defer cleanup()
	files, err := sc.ReadDirContext(ctx, remotePath)
	if err != nil {
		return nil, fmt.Errorf("list remote directory: %w", err)
	}
	result := make([]FileEntry, 0, len(files))
	for _, f := range files {
		result = append(result, FileEntry{Name: f.Name(), Size: f.Size(), Mode: f.Mode().String(), Modified: f.ModTime(), Directory: f.IsDir()})
	}
	return result, nil
}

// DownloadInput selects utf8 (the default) or base64 for an inline file response.
// 中文：DownloadInput 为内联下载选择 utf8（默认）或 base64 编码。
type DownloadInput struct {
	FileInput
	Encoding string `json:"encoding,omitempty" jsonschema:"utf8 (default) or base64"`
}

// FileContent reports original bytes, not the encoded content length. Empty files
// retain all three fields so structured and JSON-text results have the same shape.
// 中文：FileContent 的 Bytes 是原始字节数，不是编码后长度；空文件也保留全部字段，
// 以保持结构化结果与 JSON 文本结果的格式一致。
type FileContent struct {
	Content  string `json:"content"`
	Encoding string `json:"encoding"`
	Bytes    int64  `json:"bytes"`
}

// Download reads at most OutputBytes of file data. Invalid encoding, invalid UTF-8,
// oversized files and I/O failures return errors without a successful payload.
// 中文：Download 最多内联读取 OutputBytes 字节；编码、UTF-8、大小或 I/O 校验失败
// 返回错误，不返回成功文件内容。
func (m *Manager) Download(ctx context.Context, in DownloadInput) (FileContent, error) {
	if in.Encoding == "" {
		in.Encoding = "utf8"
	}
	if in.Encoding != "utf8" && in.Encoding != "base64" {
		return FileContent{}, fmt.Errorf("encoding must be utf8 or base64")
	}
	sc, remotePath, cleanup, err := m.sftp(ctx, in.FileInput)
	if err != nil {
		return FileContent{}, err
	}
	defer cleanup()
	f, err := sc.Open(remotePath)
	if err != nil {
		return FileContent{}, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, int64(m.config.OutputBytes)+1))
	if err != nil {
		return FileContent{}, err
	}
	if len(data) > m.config.OutputBytes {
		return FileContent{}, fmt.Errorf("file exceeds inline output_bytes; use sftp-download-file to stream without an inline limit")
	}
	content := string(data)
	if in.Encoding == "base64" {
		content = base64.StdEncoding.EncodeToString(data)
	} else if !utf8.Valid(data) {
		return FileContent{}, fmt.Errorf("file is not UTF-8; use base64 or sftp-download-file")
	}
	return FileContent{Content: content, Encoding: in.Encoding, Bytes: int64(len(data))}, nil
}

// UploadInput writes inline content; Mode is decimal POSIX permissions, with zero
// leaving permissions unchanged by this tool rather than requesting mode 0000.
// 中文：UploadInput 写入内联内容；Mode 使用十进制权限，零表示工具不主动 chmod，
// 并非设置为 0000。
type UploadInput struct {
	FileInput
	Content  string `json:"content"`
	Encoding string `json:"encoding,omitempty" jsonschema:"utf8 (default) or base64"`
	Mode     uint32 `json:"mode,omitempty" jsonschema:"Optional POSIX file permissions as a decimal integer (420 = 0644)"`
}

// TransferResult reports copied bytes and the destination path on its owning host.
// 中文：TransferResult 返回复制字节数和目标所在主机上的路径。
type TransferResult struct {
	Bytes int64  `json:"bytes"`
	Path  string `json:"path"`
}

// Upload decodes inline content and truncates the remote destination. Parent
// directories must exist; write failures may leave a partially written file.
// 中文：Upload 解码内容并截断远端目标文件；父目录须已存在，失败时可能留下部分内容。
func (m *Manager) Upload(ctx context.Context, in UploadInput) (TransferResult, error) {
	var data io.Reader = strings.NewReader(in.Content)
	if in.Encoding == "base64" {
		decoded, err := base64.StdEncoding.DecodeString(in.Content)
		if err != nil {
			return TransferResult{}, fmt.Errorf("invalid base64 content")
		}
		data = strings.NewReader(string(decoded))
	} else if in.Encoding != "" && in.Encoding != "utf8" {
		return TransferResult{}, fmt.Errorf("encoding must be utf8 or base64")
	}
	if in.Mode > 0777 {
		return TransferResult{}, fmt.Errorf("mode must be 0..0777")
	}
	sc, remotePath, cleanup, err := m.sftp(ctx, in.FileInput)
	if err != nil {
		return TransferResult{}, err
	}
	defer cleanup()
	return upload(sc, remotePath, data, in.Mode)
}

func upload(sc *sftp.Client, remotePath string, data io.Reader, mode uint32) (TransferResult, error) {
	f, err := sc.OpenFile(remotePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return TransferResult{}, err
	}
	n, copyErr := io.Copy(f, data)
	closeErr := f.Close()
	if copyErr != nil {
		return TransferResult{}, fmt.Errorf("upload failed after %d bytes; remote file may be partial: %w", n, copyErr)
	}
	if closeErr != nil {
		return TransferResult{}, closeErr
	}
	if mode != 0 {
		if err = sc.Chmod(remotePath, os.FileMode(mode)); err != nil {
			return TransferResult{}, err
		}
	}
	return TransferResult{Bytes: n, Path: remotePath}, nil
}

// TransferInput joins an SSH-host Path with a LocalPath on the MCP server.
// LocalPath is never interpreted as a path on the ChatGPT user's device.
// 中文：TransferInput 的 Path 位于 SSH 主机，LocalPath 位于 MCP 服务端，
// 不表示 ChatGPT 用户设备上的路径。
type TransferInput struct {
	FileInput
	LocalPath string `json:"local_path" jsonschema:"Path on the MCP server filesystem, not the ChatGPT client filesystem"`
}

// UploadFile streams a regular local file to SSH without an inline size limit.
// Local read and remote write errors are returned; remote writes are not atomic.
// 中文：UploadFile 流式上传 MCP 侧普通文件，不受内联大小限制；
// 本地读取或远端写入失败时返回错误，远端覆盖不具备原子性。
func (m *Manager) UploadFile(ctx context.Context, in TransferInput) (TransferResult, error) {
	f, err := os.Open(in.LocalPath)
	if err != nil {
		return TransferResult{}, fmt.Errorf("open local file: %w", err)
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return TransferResult{}, err
	}
	if !stat.Mode().IsRegular() {
		return TransferResult{}, fmt.Errorf("local source must be a regular file")
	}
	sc, remotePath, cleanup, err := m.sftp(ctx, in.FileInput)
	if err != nil {
		return TransferResult{}, err
	}
	defer cleanup()
	return upload(sc, remotePath, f, 0)
}

// DownloadFile streams into a sibling temporary file, then renames it to LocalPath.
// Transfer or rename failure preserves an existing destination and removes the temp.
// 中文：DownloadFile 先写入目标同目录的临时文件，成功后重命名；
// 传输或重命名失败时保留原目标并移除临时文件。
func (m *Manager) DownloadFile(ctx context.Context, in TransferInput) (TransferResult, error) {
	if in.LocalPath == "" {
		return TransferResult{}, fmt.Errorf("local_path must not be empty")
	}
	sc, remotePath, cleanup, err := m.sftp(ctx, in.FileInput)
	if err != nil {
		return TransferResult{}, err
	}
	defer cleanup()
	source, err := sc.Open(remotePath)
	if err != nil {
		return TransferResult{}, err
	}
	defer source.Close()
	// Publish only after a full transfer, so failures preserve an existing file.
	// 中文：完整传输后才替换目标，确保失败时保留已有文件。
	dest, err := os.CreateTemp(filepath.Dir(in.LocalPath), ".tunnel-ssh-download-*")
	if err != nil {
		return TransferResult{}, err
	}
	defer os.Remove(dest.Name())
	n, copyErr := io.Copy(dest, source)
	closeErr := dest.Close()
	if copyErr != nil {
		return TransferResult{}, fmt.Errorf("download after %d bytes: %w", n, copyErr)
	}
	if closeErr != nil {
		return TransferResult{}, closeErr
	}
	if err = os.Rename(dest.Name(), in.LocalPath); err != nil {
		return TransferResult{}, err
	}
	return TransferResult{Bytes: n, Path: in.LocalPath}, nil
}
