// Created by Yanjunhui
// Package attachment downloads ChatGPT file parameters without exposing signed URLs.
// 中文：attachment 下载 ChatGPT 文件参数，不在日志或错误中暴露签名 URL。
package attachment

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// File is the OpenAI file-parameter contract; optional metadata never selects a path.
// 中文：File 对应 OpenAI 文件参数；可选元数据不参与目标路径选择。
type File struct {
	DownloadURL string `json:"download_url"`
	FileID      string `json:"file_id"`
	MIMEType    string `json:"mime_type,omitempty"`
	FileName    string `json:"file_name,omitempty"`
}

// MaxBytes bounds attachment staging on disk independently of inline output limits.
// 中文：MaxBytes 限制附件临时磁盘占用，与内联输出上限相互独立。
const MaxBytes int64 = 512 << 20

// Fetch returns a rewound mode-0600 temporary file. The caller must close and remove
// it. Download, size and cancellation errors remove it and never include the URL.
// 中文：Fetch 返回已回卷的 0600 临时文件，调用方须关闭并删除；失败自动删除且错误不含 URL。
func Fetch(ctx context.Context, file File) (*os.File, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	transport := &http.Transport{DialContext: publicDial, TLSHandshakeTimeout: 15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second, DisableCompression: true}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, CheckRedirect: redirect}
	return download(ctx, file, client, MaxBytes)
}

func download(ctx context.Context, file File, client *http.Client, limit int64) (*os.File, error) {
	if strings.TrimSpace(file.FileID) == "" {
		return nil, fmt.Errorf("file.file_id must not be empty")
	}
	if err := validateURL(file.DownloadURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, file.DownloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid attachment request")
	}
	req.Header.Set("Accept-Encoding", "identity")
	response, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("attachment download failed or canceled; request a fresh ChatGPT file")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("attachment download returned HTTP %d; request a fresh ChatGPT file", response.StatusCode)
	}
	if response.ContentLength > limit {
		return nil, fmt.Errorf("attachment exceeds %d byte limit", limit)
	}
	return stage(response.Body, limit)
}

func stage(body io.Reader, limit int64) (*os.File, error) {
	f, err := os.CreateTemp("", "tunnel-console-attachment-*")
	if err != nil {
		return nil, fmt.Errorf("cannot create private attachment temporary file")
	}
	ok := false
	defer func() {
		if !ok {
			_ = f.Close()
			_ = os.Remove(f.Name())
		}
	}()
	n, err := io.Copy(f, io.LimitReader(body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("attachment download interrupted or staging failed")
	}
	if n > limit {
		return nil, fmt.Errorf("attachment exceeds %d byte limit", limit)
	}
	if _, err = f.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("cannot rewind staged attachment")
	}
	ok = true
	return f, nil
}
