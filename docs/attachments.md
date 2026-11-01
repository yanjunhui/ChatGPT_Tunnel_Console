<!-- Created by Yanjunhui -->
# ChatGPT attachment uploads

English | [简体中文](attachments.zh-CN.md) | [Documentation](README.md)

Upload a file attached to your ChatGPT message directly to an SSH host with `sftp-upload-file`. ChatGPT supplies the file to the tool, so you do not need to paste file contents or download links into the conversation.

## Upload a file

1. Attach a file to your ChatGPT message and enable the connected plugin.
2. Specify the SSH profile and destination, for example: “Upload this file to `/workspace/import/input.csv` on local-dev.”
3. Create the destination's parent directory first if it does not exist. The tool writes the exact destination you specify, regardless of the original attachment name.
4. Check the returned destination and byte count. When exact content identity matters, compare the destination's checksum as well.

Uploads overwrite an existing destination. Attachments support empty files and arbitrary binary content. A completed attachment has permissions 0600; change its permissions separately if another account needs access.

If attachment input is missing after an upgrade, [refresh the installed plugin's tools](troubleshooting.md#tools-or-parameters-still-look-outdated). The `sftp-upload-file` tool must expose a `file` parameter. The host supplies its temporary download URL; signed URLs should not be copied into chat or logs.

## Transfer limits

The attachment downloader accepts up to 512 MiB and has a five-minute download deadline, independently of `output_bytes`. ChatGPT/client upload limits and request deadlines may be lower. An expired URL requires a fresh attachment.

The file is downloaded completely into private temporary storage before any remote write begins. The tool then uploads a temporary sibling file and atomically renames it to the destination. The SFTP server must support `posix-rename@openssh.com`, as OpenSSH SFTP does.

A failed download or SFTP publication preserves the existing destination. Local staging is removed when the request finishes normally. Forced process termination may leave an OS temporary file; a disconnected SSH session can prevent cleanup of a remote `.tunnel-console-upload-*` file. Confirm no transfer is active before removing a leftover file.

## Files already on the MCP server

Use `local_path` instead of an attachment when the source exists on the machine or container running MCP:

```json
{"path":"/workspace/import/input.csv","local_path":"/workspace/staging/input.csv"}
```

Provide exactly one source: `file` or `local_path`. Local-path uploads stream directly to the remote destination and can leave a partial file on failure. Both methods overwrite the destination. In the default Docker setup, MCP and the bundled SSH host share `/workspace`; choose different source and destination paths.

## Client integration

The tool follows the [OpenAI file input contract](https://developers.openai.com/plugins/reference#define-file-inputs). Its `_meta["openai/fileParams"]` identifies `file`, with required `download_url` and `file_id`, and optional `mime_type` and `file_name`. Attachment bytes are transferred outside the model's text context.

The downloader validates HTTPS hosts, redirects and public DNS addresses. It rejects private/loopback targets, embedded credentials and non-443 ports. Errors omit signed URLs and HTTP response bodies. Allowed host families and exact result/error formats are listed in the [tool reference](tool-reference.md). Host and network egress policies still apply.
