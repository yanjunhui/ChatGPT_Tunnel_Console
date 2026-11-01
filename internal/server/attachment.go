// Created by Yanjunhui
package server

import (
	"context"
	"encoding/json"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/remote"
)

func registerAttachmentUpload(s *mcp.Server, manager *remote.Manager) {
	destructive := true
	mcp.AddTool(s, &mcp.Tool{
		Name:        "sftp-upload-file",
		Description: "Upload exactly one ChatGPT file attachment or local_path on the MCP server to remote path. Overwrites the destination. Attachments stage privately (512 MiB maximum) and publish atomically via SFTP; local_path retains streaming behavior. Prefer file for chat attachments; never copy file bytes into tool arguments.",
		Meta:        mcp.Meta{"openai/fileParams": []string{"file"}},
		InputSchema: attachmentSchema(),
		Annotations: &mcp.ToolAnnotations{DestructiveHint: &destructive},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in remote.UploadFileInput) (*mcp.CallToolResult, any, error) {
		out, err := manager.UploadSource(ctx, in)
		if err != nil {
			return nil, nil, err
		}
		data, err := json.Marshal(out)
		if err != nil {
			return nil, nil, err
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(data)}}}, nil, nil
	})
}

func attachmentSchema() map[string]any {
	// OpenAI fileParams requires an object, not the SDK's nullable pointer union.
	// 中文：OpenAI fileParams 要求对象，不能使用 SDK 为指针推导的可空联合类型。
	text := map[string]any{"type": "string"}
	file := map[string]any{"type": "object", "additionalProperties": false,
		"properties": map[string]any{"download_url": text, "file_id": text, "mime_type": text, "file_name": text},
		"required":   []string{"download_url", "file_id"}}
	return map[string]any{"type": "object", "additionalProperties": false,
		"properties": map[string]any{"profile": text, "path": text, "local_path": text, "file": file},
		"required":   []string{"path"}}
}
