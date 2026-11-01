// Created by Yanjunhui
package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/yanjunhui/ChatGPT_Tunnel_Console/internal/testssh"
)

func TestStdioWorker(t *testing.T) {
	if os.Getenv("TUNNEL_SSH_TEST_WORKER") != "1" {
		return
	}
	err := run(context.Background(), []string{"serve", "--config", os.Getenv("TUNNEL_SSH_TEST_CONFIG")})
	if err != nil {
		os.Exit(2)
	}
	os.Exit(0)
}

func TestRealStdioProcess(t *testing.T) {
	s := testssh.Start(t)
	data, err := toml.Marshal(s.Config)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config with spaces.toml")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestStdioWorker$")
	cmd.Env = append(os.Environ(), "TUNNEL_SSH_TEST_WORKER=1", "TUNNEL_SSH_TEST_CONFIG="+path)
	cmd.Stderr = os.Stderr
	c, err := mcp.NewClient(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil).Connect(t.Context(), &mcp.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	tools, err := c.ListTools(t.Context(), nil)
	if err != nil || len(tools.Tools) != 17 {
		t.Fatalf("tool discovery: %+v %v", tools, err)
	}
	result, err := c.CallTool(t.Context(), &mcp.CallToolParams{Name: "run-command", Arguments: map[string]any{"command": "printf STDIO_SUCCESS"}})
	if err != nil || result.IsError {
		t.Fatalf("command: %+v %v", result, err)
	}
	if !strings.Contains(result.Content[0].(*mcp.TextContent).Text, "STDIO_SUCCESS") {
		t.Fatalf("unexpected output: %+v", result)
	}
}

func TestCLIRejectsUnknownCommand(t *testing.T) {
	if err := run(t.Context(), []string{"unknown"}); err == nil {
		t.Fatal("accepted unknown command")
	}
}

func TestInitHelpAndInvalidArguments(t *testing.T) {
	if err := run(t.Context(), []string{"init", "--help"}); err != nil {
		t.Fatal(err)
	}
	if err := run(t.Context(), []string{"init", "unexpected"}); err == nil {
		t.Fatal("accepted positional arguments")
	}
}
