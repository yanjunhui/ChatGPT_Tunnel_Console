// Created by Yanjunhui
package remote

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/pkg/sftp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/attachment"
)

// UploadFileInput accepts exactly one local source or ChatGPT file parameter.
// 中文：UploadFileInput 要求本地来源和 ChatGPT 文件参数二选一。
type UploadFileInput struct {
	FileInput
	LocalPath string           `json:"local_path,omitempty" jsonschema:"Source on the MCP server filesystem; mutually exclusive with file"`
	File      *attachment.File `json:"file,omitempty" jsonschema:"ChatGPT attachment supplied by the host; bytes never pass through model context"`
}

// UploadSource preserves local-path uploads and adds staged ChatGPT attachments.
// Attachment failures preserve the destination; publishing requires POSIX rename.
// 中文：UploadSource 保留本地上传并支持附件；附件失败保留目标，发布要求 POSIX rename。
func (m *Manager) UploadSource(ctx context.Context, in UploadFileInput) (TransferResult, error) {
	return m.uploadSource(ctx, in, attachment.Fetch)
}

func (m *Manager) uploadSource(ctx context.Context, in UploadFileInput,
	fetch func(context.Context, attachment.File) (*os.File, error)) (TransferResult, error) {
	if (in.File == nil) == (in.LocalPath == "") {
		return TransferResult{}, fmt.Errorf("provide exactly one source: file or local_path")
	}
	if strings.TrimSpace(in.Path) == "" {
		return TransferResult{}, fmt.Errorf("remote path must not be empty")
	}
	if _, _, err := m.config.Profile(in.Profile); err != nil {
		return TransferResult{}, err
	}
	if in.File == nil {
		return m.UploadFile(ctx, TransferInput{FileInput: in.FileInput, LocalPath: in.LocalPath})
	}
	f, err := fetch(ctx, *in.File)
	if err != nil {
		return TransferResult{}, err
	}
	defer os.Remove(f.Name())
	defer f.Close()
	sc, target, cleanup, err := m.sftp(ctx, in.FileInput)
	if err != nil {
		return TransferResult{}, err
	}
	defer cleanup()
	return publishAttachment(sc, target, f)
}

func publishAttachment(sc *sftp.Client, target string, source io.Reader) (TransferResult, error) {
	if _, supported := sc.HasExtension("posix-rename@openssh.com"); !supported {
		return TransferResult{}, fmt.Errorf("attachment uploads require SFTP POSIX rename support")
	}
	temporary := path.Join(path.Dir(target), fmt.Sprintf(".tunnel-console-upload-%x", rand.Text()))
	f, err := sc.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL)
	if err != nil {
		return TransferResult{}, fmt.Errorf("cannot create remote attachment temporary file: %w", err)
	}
	defer sc.Remove(temporary)
	defer f.Close()
	if err = f.Chmod(0600); err != nil {
		return TransferResult{}, err
	}
	n, err := io.Copy(f, source)
	if err != nil {
		return TransferResult{}, fmt.Errorf("attachment SFTP copy failed; destination preserved: %w", err)
	}
	if err = f.Close(); err != nil {
		return TransferResult{}, err
	}
	if err = sc.PosixRename(temporary, target); err != nil {
		return TransferResult{}, fmt.Errorf("cannot publish attachment; destination preserved: %w", err)
	}
	return TransferResult{Bytes: n, Path: target}, nil
}
