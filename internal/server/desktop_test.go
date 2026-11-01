// Created by Yanjunhui
package server

import (
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestDesktopToolsAndDisabledProfile(t *testing.T) {
	c := client(t)
	list, err := c.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, tool := range list.Tools {
		switch tool.Name {
		case "vnc-screenshot", "desktop-screenshot":
			if !tool.Annotations.ReadOnlyHint || tool.OutputSchema != nil {
				t.Fatal("screenshot contract missing")
			}
			seen[tool.Name] = true
		case "vnc-key", "vnc-type", "vnc-pointer", "desktop-key", "desktop-type", "desktop-pointer":
			if tool.Annotations.ReadOnlyHint {
				t.Fatal("input marked read-only")
			}
			seen[tool.Name] = true
		}
	}
	if len(seen) != 8 {
		t.Fatal("desktop tools missing")
	}
	for name := range seen {
		args := map[string]any{"keys": []string{"Return"}, "text": "hello", "events": []any{map[string]any{"x": 0, "y": 0, "buttons": 0}}}
		result, err := c.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || !result.IsError {
			t.Fatalf("%s accepted disabled desktop: %v", name, err)
		}
	}
}
