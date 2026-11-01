// Created by Yanjunhui
package server

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/desktop"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/remote"
)

func registerDesktop(s *mcp.Server, manager *remote.Manager) {
	mcp.AddTool(s, &mcp.Tool{Name: "vnc-screenshot", Description: "Capture a VNC desktop through its configured SSH profile. Returns a native PNG image and width/height metadata. Use these exact pixel coordinates for pointer events. VNC must be explicitly enabled in the profile.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}},
		func(ctx context.Context, _ *mcp.CallToolRequest, in remote.DesktopInput) (*mcp.CallToolResult, desktop.Info, error) {
			data, info, err := manager.Screenshot(ctx, in)
			if err != nil {
				return nil, desktop.Info{}, err
			}
			metadata, err := json.Marshal(info)
			if err != nil {
				return nil, desktop.Info{}, err
			}
			return &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.ImageContent{Data: data, MIMEType: "image/png"},
				&mcp.TextContent{Text: string(metadata)},
			}}, info, nil
		})
	bind(s, "vnc-pointer", "Send a bounded move/click/drag/scroll gesture in native screenshot pixels. Buttons are released at the end. Input delivery does not prove application completion; take a new screenshot. Do not retry blindly after an error.", false, manager.Pointer)
	bind(s, "vnc-key", "Press and release one keyboard chord, such as Control_L+l or Return. Keys are released before the call ends. Inspect a new screenshot to verify the result.", false, manager.Key)
	bind(s, "vnc-type", "Type text as Unicode key press/release events, without clipboard access. Newline sends Return. Target Unicode support varies; inspect a new screenshot after input.", false, manager.Type)
}
