// Created by Yanjunhui
package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestAttachmentFileParamsSchema(t *testing.T) {
	c := client(t)
	list, err := c.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range list.Tools {
		if tool.Name != "sftp-upload-file" {
			continue
		}
		data, _ := json.Marshal(tool)
		var schema struct {
			Meta  map[string][]string `json:"_meta"`
			Input struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"inputSchema"`
		}
		if err := json.Unmarshal(data, &schema); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(schema.Meta["openai/fileParams"], []string{"file"}) {
			t.Fatal("missing file metadata")
		}
		var file struct {
			Type       string
			Properties map[string]struct{ Type string }
			Required   []string
		}
		if err := json.Unmarshal(schema.Input.Properties["file"], &file); err != nil {
			t.Fatal(err)
		}
		if file.Type != "object" {
			t.Fatalf("file is not a direct object schema: %s", data)
		}
		for _, name := range []string{"download_url", "file_id", "mime_type", "file_name"} {
			if file.Properties[name].Type != "string" {
				t.Fatalf("missing file property %s", name)
			}
		}
		if !reflect.DeepEqual(file.Required, []string{"download_url", "file_id"}) {
			t.Fatal(file.Required)
		}
		if !reflect.DeepEqual(schema.Input.Required, []string{"path"}) {
			t.Fatal(schema.Input.Required)
		}
		return
	}
	t.Fatal("upload tool not registered")
}

func TestAttachmentToolLocalCompatibilityAndInvalidInputs(t *testing.T) {
	c := client(t)
	source := filepath.Join(t.TempDir(), "local.txt")
	if err := os.WriteFile(source, []byte("local-compatible"), 0600); err != nil {
		t.Fatal(err)
	}
	call(t, c, "sftp-upload-file", map[string]any{"path": "local.txt", "local_path": source}, false)
	if text := call(t, c, "sftp-download", map[string]any{"path": "local.txt"}, false); !strings.Contains(text, "local-compatible") {
		t.Fatal(text)
	}
	for _, args := range []map[string]any{
		{"path": "local.txt"},
		{"path": "local.txt", "file": map[string]any{"download_url": "https://127.0.0.1/private?secret=hidden", "file_id": "file-test"}},
		{"path": "local.txt", "local_path": source, "file": map[string]any{"download_url": "https://files.oaiusercontent.com/x", "file_id": "file-test"}},
	} {
		if text := call(t, c, "sftp-upload-file", args, true); strings.Contains(text, "hidden") {
			t.Fatal("URL leaked")
		}
	}
}
