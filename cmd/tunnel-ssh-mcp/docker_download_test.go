// Created by Yanjunhui
package main

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_SSH/internal/remote"
)

func dockerCheckDownload(t *testing.T, ctx context.Context, c *mcp.ClientSession, path, want string) {
	t.Helper()
	result, err := c.CallTool(ctx, &mcp.CallToolParams{Name: "sftp-download", Arguments: map[string]any{"path": path}})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError || result.StructuredContent == nil || len(result.Content) != 1 {
		t.Fatalf("download %s: %+v", path, result)
	}
	structured, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("download has no text result")
	}
	for _, data := range [][]byte{structured, []byte(text.Text)} {
		var got remote.FileContent
		if err = json.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if got.Content != want || got.Encoding != "utf8" || got.Bytes != int64(len(want)) {
			t.Fatalf("download %s payload differs: %+v", path, got)
		}
	}
}
