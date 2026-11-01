<!-- Created by Yanjunhui -->
# Troubleshooting

English | [简体中文](troubleshooting.zh-CN.md) | [Documentation](README.md)

Start with the stage that fails: service startup, tunnel discovery, SSH connection, or a particular tool. A running container confirms the process is up; an actual command through the plugin checks the full connection.

## Inspect service status and logs

For Docker, run these from the deployment directory. Include your desktop/VNC overlays, or use the deployment's saved `COMPOSE_FILE`:

```sh
docker compose ps
docker compose logs --tail=100 mcp
docker compose logs --tail=100 dev
```

For a native service:

```sh
# macOS
sudo launchctl print system/com.yanjunhui.tunnel-ssh-mcp
sudo tail -n 100 /var/log/tunnel-ssh-mcp/stderr.log

# Linux
sudo systemctl status tunnel-ssh-mcp.service
sudo journalctl -u tunnel-ssh-mcp.service -n 100 --no-pager
```

Record the version, deployment mode, failing tool and relevant error. Remove credentials, signed attachment URLs, tunnel IDs and private host/account details before sharing logs. See [Contributing](../CONTRIBUTING.md) for reporting problems.

## The tunnel or plugin is missing

- Check that the tunnel belongs to the Platform organization associated with the current ChatGPT account or workspace.
- Confirm that the service uses the same tunnel ID selected in the plugin, and that the runtime key grants Read + Use permission.
- Check outbound HTTPS access from the service host and tunnel-client errors in the MCP logs.
- Run only one stdio tunnel-client for each tunnel ID. Stop an old instance before starting its replacement.

For manual installations, `doctor` runs the official tunnel diagnostics:

```sh
./bin/tunnel-console-mcp doctor --config config.toml
```

Use the active configuration path. Diagnostics contact the tunnel service; a successful result does not test SSH credentials. After the tunnel appears, enable the plugin and run a command on a profile.

## Tools or parameters still look outdated

1. Confirm that the intended deployment was rebuilt and restarted. The source checkout and the running binary can be different versions.
2. Open the management page for the exact installed plugin and refresh its tools. Check the plugin identity and signed-in account, especially if multiple plugins have the same display name.
3. Confirm discovery of the expected tool or parameter, such as the `file` input of `sftp-upload-file`, then try it in a new chat with that plugin enabled.

Refreshing a browser tab or creating a new chat does not refresh another plugin's saved tool catalog. Likewise, refreshing a plugin in one account does not update a separate installation in another account. See the [OpenAI metadata refresh instructions](https://developers.openai.com/plugins/deploy/connect-chatgpt#refresh-metadata).

To inspect a running Docker binary's version:

```sh
docker compose exec mcp tunnel-console-mcp version
```

## Profiles appear, but commands fail

`list-connections` reads configuration without connecting to the target. Check the selected profile's host, port, user, authentication method and `workdir`.

- The target must be reachable from the MCP service. In a container, `127.0.0.1` means that container, not the Docker host or another service.
- The SSH account must have access to its private key or configured password source and permission to enter the working directory.
- Native boot services cannot depend on a login shell's SSH agent. Import a private-key file or password instead.
- Commands use POSIX shells. If a command works interactively but fails here, check PATH, shell startup files, environment variables and whether a PTY is required.

For the bundled environment, inspect the `dev` container's status and logs. For external SSH, test connectivity from the service's network environment. The service currently does not verify SSH host keys; see the [security policy](../SECURITY.md).

## Configuration changes have no effect

Configuration is loaded at startup. Edit or import the configuration used by the running service, then restart MCP. Docker normally uses `data/current/config.toml`, while a native service uses `/usr/local/lib/tunnel-ssh-mcp/current/config.toml`.

The initializer reuses saved configuration by default. For Docker initialization, use `--reconfigure` to reopen the wizard. Both installation modes accept `--config` to import a complete replacement. The wizard replaces the profile selection; use a complete TOML when retaining multiple profiles. Keep the same configuration mount for initialization and runtime. See [installation and upgrades](installation.md).

`check` validates TOML structure; it does not test credentials or connectivity. A restart clears in-memory sessions, so finish or record active work first.

## A command times out or output is missing

Independent commands default to 60 seconds. Increase `timeout_seconds`, or use a background session and read its output in later calls. Disabling the command deadline does not disable client-side request limits.

When `dropped_bytes` is nonzero, output has exceeded the retained buffer. Read more frequently, increase `output_bytes` within its limit, or redirect full logs to a remote file. Pass the previous `output.next_cursor` for incremental reads; negative cursors and positions beyond produced output fail.

Input submission does not mean the command completed. Inspect the output and exit status. If a request fails after starting, check the remote process before repeating it to avoid duplicate work.

## A file transfer fails

| Symptom | What to check |
|---|---|
| Destination not found or permission denied | Create the parent directory on the SSH host and check the SSH account's write permissions |
| Local source not found | `local_path` must exist on the MCP server, including inside its container |
| Inline download is too large or not UTF-8 | Use `sftp-download-file` for larger files, or `encoding: "base64"` for small binary files |
| Attachment URL expired | Reattach the file so ChatGPT supplies a fresh download link |
| Attachment source rejected | Supply one source, `file` or `local_path`; let ChatGPT provide the file object |
| Attachment publication unsupported | The SFTP server must support `posix-rename@openssh.com`; OpenSSH SFTP supports this extension |

The attachment downloader has a 512 MiB limit and a five-minute download deadline. ChatGPT/client upload limits and request deadlines may be lower. Network access restrictions can also prevent downloading an otherwise valid attachment. Do not paste temporary URLs or keys into bug reports.

Failed attachment publication preserves the original destination. A disconnected SSH session may leave a private `.tunnel-console-upload-*` temporary file; inspect it and confirm no upload is active before removing it. Inline and local-path uploads can leave partial destination files. See [attachment uploads](attachments.md) for details.

## Desktop capture or input fails

- **Tool disabled:** enable `desktop` or `vnc` on the selected profile and restart MCP. The configuration flags shown by `list-connections` are not live desktop probes.
- **Native display unavailable:** check the Linux X11 display, `xwd`, `xdotool`, `C.UTF-8`, and the SSH account's display authentication. Do not disable X access control globally.
- **VNC connection fails:** check the target address from the SSH host, the VNC listener, SSH forwarding and supported authentication. For the bundled desktop, include both desktop and VNC overlays.
- **Pointer out of bounds:** take a fresh screenshot and use original image coordinates. Desktop dimensions may have changed.
- **Text lands in the wrong place:** check focus and recapture before retrying. For non-ASCII native input, use short chunks (about 100 characters at the default timeout), or adjust the desktop timeout up to 120 seconds.
- **Image is too large:** reduce the target desktop resolution. Client message limits may be lower than the server's image limits.

The native backend supports Linux X11. Supported VNC authentication, key names and protocol limits are documented in the [desktop](desktop.md), [VNC](vnc.md) and [tool reference](tool-reference.md) guides.
