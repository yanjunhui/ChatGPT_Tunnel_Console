<!-- Created by Yanjunhui -->
# ChatGPT attachment uploads

English | [简体中文](attachments.zh-CN.md) | [README](../README.md)

Version 0.3.3 adds direct attachment input to `sftp-upload-file`. Refresh the ChatGPT application's tool catalog after upgrading. The descriptor uses the [official OpenAI file parameter contract](https://developers.openai.com/plugins/reference#define-file-inputs): `_meta["openai/fileParams"]` identifies `file`, with required `download_url` and `file_id` and optional `mime_type` and `file_name`.

Attach a file in ChatGPT and ask it to upload to an explicit destination such as `/workspace/import/input.csv`. The destination parent must exist. The host supplies the temporary download URL; do not paste signed URLs into chat or copy attachment bytes into tool arguments. Success returns the byte count and destination path; verify the destination hash when byte identity matters.

For a file already on the MCP server, the existing contract remains:

```json
{"path":"/workspace/import/input.csv","local_path":"/workspace/staging/input.csv"}
```

Provide exactly one of `file` or `local_path`. Both sources overwrite the destination. Attachments are downloaded completely to a private temporary file, then copied into a private SFTP sibling and atomically renamed into place. The SSH server must support `posix-rename@openssh.com`; a completed attachment has permissions 0600. Local-path uploads retain their original non-atomic streaming semantics.

Attachment downloads are limited to 512 MiB and five minutes, independently of `output_bytes`. Empty files and arbitrary binary content are supported. HTTPS hosts, redirects and public DNS addresses are validated; private/loopback targets, embedded credentials and non-443 ports are rejected. Temporary URLs and HTTP bodies are not included in errors. See [tool reference](tool-reference.md) for the allowed host families and exact failure behavior. These application restrictions do not replace host/network egress policy.

After an expired URL, request a fresh attachment rather than repeatedly retrying the same link. A failed download never starts a remote write. Failed SFTP publication preserves the prior destination; a disconnected SSH session can prevent cleanup of its private `.tunnel-console-upload-*` temporary file. Local staging is always removed during normal request unwinding; process termination may leave an OS temporary file.

Automated validation covers protocol metadata/schema, source selection, existing local-path behavior, empty/Unicode/binary content, byte limits, redirects, cancellation, signed-URL redaction and real SFTP atomic publication. Those checks are distinct from a live ChatGPT attachment transfer; deployment records must state whether the latter was actually exercised.
