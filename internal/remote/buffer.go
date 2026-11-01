// Created by Yanjunhui
package remote

import (
	"fmt"
	"sync"
)

// Buffer retains recent output and uses absolute byte cursors to report gaps.
// 中文：Buffer 保留最近输出，以绝对字节游标报告被丢弃的内容。
type Buffer struct {
	mu    sync.Mutex
	data  []byte
	limit int
	total int64
}

// Output describes retained text, the next absolute cursor and bytes lost since
// the requested cursor. Cursors count bytes, not Unicode characters.
// 中文：Output 包含保留文本、下次绝对游标及从请求游标起丢失的字节数；游标不按字符计数。
type Output struct {
	Text         string `json:"text"`
	NextCursor   int64  `json:"next_cursor"`
	DroppedBytes int64  `json:"dropped_bytes"`
}

// NewBuffer creates a bounded buffer; limit must be nonnegative.
// 中文：NewBuffer 创建有界缓冲区；limit 不得为负数。
func NewBuffer(limit int) *Buffer { return &Buffer{limit: limit} }

// Write accepts all bytes while retaining at most limit bytes of the newest output.
// 中文：Write 接收全部字节，但只保留不超过 limit 字节的最新输出。
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := len(p)
	b.total += int64(n)
	if n >= b.limit {
		b.data = append(b.data[:0], p[n-b.limit:]...)
	} else {
		if extra := len(b.data) + n - b.limit; extra > 0 {
			b.data = b.data[extra:]
		}
		b.data = append(b.data, p...)
	}
	return n, nil
}

// Read returns available bytes from cursor without consuming them. Negative or
// future cursors fail; expired cursors report the gap in DroppedBytes.
// 中文：Read 从 cursor 读取但不消费数据；负数或超前游标报错，过期游标报告丢失字节数。
func (b *Buffer) Read(cursor int64) (Output, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if cursor < 0 || cursor > b.total {
		return Output{}, fmt.Errorf("cursor must be between 0 and %d", b.total)
	}
	start := b.total - int64(len(b.data))
	dropped := int64(0)
	if cursor < start {
		dropped = start - cursor
		cursor = start
	}
	return Output{Text: string(b.data[cursor-start:]), NextCursor: b.total, DroppedBytes: dropped}, nil
}
