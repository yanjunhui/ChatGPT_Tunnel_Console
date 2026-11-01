// Created by Yanjunhui
package server

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/remote"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/testssh"
)

func client(t *testing.T) *mcp.ClientSession {
	t.Helper()
	fixture := testssh.Start(t)
	manager := remote.New(fixture.Config)
	t.Cleanup(manager.Close)
	ct, st := mcp.NewInMemoryTransports()
	ss, err := New(manager).Connect(t.Context(), st, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "1"}, nil).Connect(t.Context(), ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func call(t *testing.T, c *mcp.ClientSession, name string, args map[string]any, failed bool) string {
	t.Helper()
	result, err := c.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	if result.IsError != failed {
		t.Fatalf("%s: expected error=%v; got %+v", name, failed, result)
	}
	if len(result.Content) == 0 {
		t.Fatal("missing content")
	}
	return result.Content[0].(*mcp.TextContent).Text
}

func TestMCPDiscoveryAndCommand(t *testing.T) {
	c := client(t)
	list, err := c.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(list.Tools) != 21 {
		t.Fatalf("expected 21 tools; got %d", len(list.Tools))
	}
	for _, tool := range list.Tools {
		if tool.Name == "run-command" && tool.Annotations.ReadOnlyHint {
			t.Fatal("arbitrary shell mislabeled read-only")
		}
	}
	text := call(t, c, "list-connections", map[string]any{}, false)
	if strings.Contains(text, "client-key") {
		t.Fatal("credential path exposed in discovery")
	}
	text = call(t, c, "run-command", map[string]any{"command": "printf hello\nprintf world"}, false)
	var result remote.CommandResult
	if err = json.Unmarshal([]byte(text), &result); err != nil || result.Stdout.Text != "helloworld" {
		t.Fatalf("%s %v", text, err)
	}
	call(t, c, "run-command", map[string]any{"command": "exit 9"}, true)
	call(t, c, "run-command", map[string]any{"command": "true", "profile": "unknown"}, true)
	call(t, c, "signal-process", map[string]any{"pid": 0}, true)
}

func TestMCPFilesAndSessionSchemas(t *testing.T) {
	c := client(t)
	call(t, c, "sftp-upload", map[string]any{"path": "mcp.txt", "content": "你好"}, false)
	text := call(t, c, "sftp-download", map[string]any{"path": "mcp.txt"}, false)
	if !strings.Contains(text, "你好") {
		t.Fatal(text)
	}
	call(t, c, "open-session", map[string]any{"name": "terminal"}, false)
	call(t, c, "write-session-input", map[string]any{"name": "terminal", "data": "printf done\\n\n"}, false)
	call(t, c, "read-session-output", map[string]any{"name": "terminal", "cursor": 0}, false)
	call(t, c, "close-session", map[string]any{"name": "terminal"}, false)
	call(t, c, "read-session-output", map[string]any{"name": "terminal"}, true)
}
