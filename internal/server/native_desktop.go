// Created by Yanjunhui
package server

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/remote"
)

func registerNativeDesktop(s *mcp.Server, manager *remote.Manager) {
	mcp.AddTool(s, &mcp.Tool{Name: "desktop-screenshot", Description: "Capture the configured native X11 desktop directly over SSH, independently of VNC. Returns PNG image content and native width/height metadata. Use native pixels for input.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(ctx context.Context, _ *mcp.CallToolRequest, in remote.DesktopInput) (*mcp.CallToolResult, desktop.Info, error) {
			data, info, err := manager.DesktopScreenshot(ctx, in)
			if err != nil {
				return nil, desktop.Info{}, err
			}
			metadata, err := json.Marshal(info)
			if err != nil {
				return nil, desktop.Info{}, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.ImageContent{Data: data, MIMEType: "image/png"}, &mcp.TextContent{Text: string(metadata)}}}, info, nil
		})
	bind(s, "desktop-pointer", "Move/click/drag/scroll on the native X11 desktop directly over SSH. Use screenshot pixel coordinates. Buttons are released after each gesture; inspect a new screenshot after input.", false, manager.DesktopPointer)
	bind(s, "desktop-key", "Press and release a key chord on the native X11 desktop over SSH, without VNC. Inspect application state after input; do not blindly retry errors.", false, manager.DesktopKey)
	bind(s, "desktop-type", "Type UTF-8 through native X11 key events over SSH without VNC or clipboard access. Newlines send Return and may submit commands/forms; inspect the result.", false, manager.DesktopType)
}
