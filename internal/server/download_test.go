// Created by Yanjunhui
package server

import (
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/remote"
)

func TestMCPDownloadOutputSchema(t *testing.T) {
	c := client(t)
	list, err := c.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range list.Tools {
		if tool.Name != "sftp-download" {
			continue
		}
		data, err := json.Marshal(tool.OutputSchema)
		if err != nil {
			t.Fatal(err)
		}
		var schema struct {
			Type       string
			Properties map[string]struct{ Type string }
			Required   []string
		}
		if err := json.Unmarshal(data, &schema); err != nil || schema.Type != "object" {
			t.Fatalf("download must advertise an object output schema: %s (%v)", data, err)
		}
		for name, kind := range map[string]string{"content": "string", "encoding": "string", "bytes": "integer"} {
			if schema.Properties[name].Type != kind || !slices.Contains(schema.Required, name) {
				t.Errorf("missing required %s property %q: %s", kind, name, data)
			}
		}
		return
	}
	t.Fatal("sftp-download missing from discovery")
}

func TestMCPDownloadStructuredContent(t *testing.T) {
	c := client(t)
	cases := []struct{ name, content, encoding string }{
		{"empty", "", ""},
		{"unicode", "GO_TUNNEL_SFTP：中文往返测试\n", "utf8"},
		{"json-text", `{"content":[{"type":"text","text":"literal data"}]}`, ""},
		{"binary", string([]byte{0, 1, 255, 254}), "base64"},
		{"limit", strings.Repeat("x", 4096), "utf8"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded := base64.StdEncoding.EncodeToString([]byte(tc.content))
			call(t, c, "sftp-upload", map[string]any{"path": "download.bin", "content": encoded, "encoding": "base64"}, false)
			result, err := c.CallTool(t.Context(), &mcp.CallToolParams{Name: "sftp-download", Arguments: map[string]any{
				"path": "download.bin", "encoding": tc.encoding,
			}})
			if err != nil || result.IsError {
				t.Fatalf("download: %+v %v", result, err)
			}
			want := remote.FileContent{Content: tc.content, Encoding: "utf8", Bytes: int64(len(tc.content))}
			if tc.encoding == "base64" {
				want.Content, want.Encoding = encoded, "base64"
			}
			checkDownloadPayload(t, result, want)
		})
	}
}

func checkDownloadPayload(t *testing.T, result *mcp.CallToolResult, want remote.FileContent) {
	t.Helper()
	if result.StructuredContent == nil {
		t.Fatal("download has no structured result; structured-only clients cannot read the file")
	}
	data, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var got remote.FileContent
	if err = json.Unmarshal(data, &got); err != nil || got != want {
		t.Fatalf("structured download: %+v, want %+v (%v)", got, want, err)
	}
	if len(result.Content) != 1 {
		t.Fatalf("expected one compatible text result, got %d", len(result.Content))
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("download text result missing")
	}
	if err = json.Unmarshal([]byte(text.Text), &got); err != nil || got != want {
		t.Fatalf("text download differs: %s (%v)", text.Text, err)
	}
}

func TestMCPDownloadErrorsHaveNoSuccessPayload(t *testing.T) {
	c := client(t)
	call(t, c, "sftp-upload", map[string]any{"path": "invalid.bin", "content": "/w==", "encoding": "base64"}, false)
	call(t, c, "sftp-upload", map[string]any{"path": "large.txt", "content": strings.Repeat("x", 4097)}, false)
	cases := []struct {
		name string
		args map[string]any
	}{
		{"missing-argument", map[string]any{}},
		{"empty-path", map[string]any{"path": ""}},
		{"missing-file", map[string]any{"path": "missing.txt"}},
		{"unknown-profile", map[string]any{"path": "invalid.bin", "profile": "missing"}},
		{"invalid-encoding", map[string]any{"path": "invalid.bin", "encoding": "unknown"}},
		{"invalid-utf8", map[string]any{"path": "invalid.bin"}},
		{"overflow", map[string]any{"path": "large.txt"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := c.CallTool(t.Context(), &mcp.CallToolParams{Name: "sftp-download", Arguments: tc.args})
			if err != nil {
				t.Fatal(err)
			}
			if !result.IsError || result.StructuredContent != nil || len(result.Content) == 0 {
				t.Fatalf("expected tool error without a success payload: %+v", result)
			}
		})
	}
}
