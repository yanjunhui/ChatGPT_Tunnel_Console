<!-- Created by Yanjunhui -->
# Local Docker deployment and live tunnel acceptance

English | [简体中文](live-deployment.zh-CN.md)

Date: 2026-11-01 (Asia/Shanghai). Project: `ChatGPT_Tunnel_Console`.

## Current deployment: 0.3.2

The formal MCP runtime reached **0.3.2** at 20:16:44 on 2026-11-01. The bundled native X11 desktop and separately managed VNC service use the tested 0.3.0 images; only MCP changed for the 0.3.1 screenshot and 0.3.2 Unicode fixes. Existing tunnel credentials, SSH identity, workspace and developer home were retained. There are no published host ports.

| Check | Observed result |
|---|---|
| Running version | `tunnel-console-mcp version` returned `0.3.2` |
| Runtime images | `go-tunnel-console-mcp:0.3.2`, `go-tunnel-console-desktop:0.3.0`, `go-tunnel-console-vnc:0.3.0` |
| Executable SHA-256 | `1fe712315dc2c3faff3b16443760e4da7653b1d16a60a2efc6c8df604458603c` |
| Tunnel health | Required successful control-plane poll; live/ready HTTP 200; result `ok` |
| Real ChatGPT SSH | `developer`, `/workspace`, exit 0; configuration discovery separately reported both desktop capabilities enabled |
| Real screenshots | Both paths returned visible 1280×800 PNGs; ChatGPT read an undisclosed `VISUAL_PROOF_CEDAR_5873` marker from each without SSH/SFTP reads |
| Native input | Pointer, Control+u, text and Return produced `NATIVE_032:中文汉字αβ日文あいう`; readback was exact, mouse `(407,663)` |
| VNC input | Pointer, chord, text and Return produced `VNC_031:中文` with exact readback; mouse `(410,203)` |
| VNC stopped | VNC screenshot returned `Connection refused`; native capture and all input tools remained usable and produced `NATIVE_WITH_VNC_STOPPED:中文` |
| Service independence | Stopping VNC did not restart MCP or dev; the dev start time remained 19:54:33 and MCP 20:16:44 |
| VNC restored | Returned to `healthy`; real ChatGPT used VNC screenshot/pointer/key/type again and read back `VNC_RESTORED_032:中文` exactly |
| Cleanup | The five known test files were hashed and privately archived before removal; the dedicated directory was removed and its absence independently checked; ChatGPT `list-sessions` returned `[]` |
| Source CI | All four jobs passed for [0.3.2 source commit](https://github.com/yanjunhui/ChatGPT_Tunnel_Console/actions/runs/41305716797), including Linux amd64 Docker integration |

The screenshot A/B comparison and initial Unicode failure are preserved in the [verification record](verification.md). They were actual acceptance failures that required fixes, not successful checks. Input acknowledgment alone is never used as proof that an application processed the input. These results cover the bundled Linux X11 desktop, not native macOS/Windows/Wayland desktops, host reboot or network-outage recovery.

### Current operation and rollback

The local `.env` selects all three Compose files and explicit image tags. Use `docker compose stop vnc` / `docker compose start vnc` to manage only the optional service; `desktop-*` remains independent of it. Both backends still use SSH and share the X display supplied by dev. See [the desktop guide](desktop.md) for native-only configuration and external VNC targets.

Private backups and evidence are in `.cache/upgrade-0.3.0/`: the pre-upgrade `data`/workspace archive, developer-home/SSH-identity volume archive, prior environment files, container snapshots, activation logs, health results and ChatGPT evidence. Prior 0.2.0/0.3.0/0.3.1 images are retained. A runtime-only rollback requires stopping MCP, restoring the selected image tag in `.env`, then recreating only MCP with `--no-deps --no-build --pull never`; it ends in-memory sessions. Older versions restore their documented screenshot/Unicode limitations. A full 0.2.0 rollback also requires its matching configuration generation and two-file Compose setup. No successful-upgrade rollback drill was performed.

All sections below are historical deployment records, not the current container inventory or current rollback procedure.

## Historical 0.2.0 deployment

The formal deployment reached **0.2.0** at 19:30:42 on 2026-11-01. The poll-required health check and real ChatGPT SSH passed; the cached tool list did not expose VNC tools at that stage, so screenshot/input acceptance was not claimed. Private backups and evidence remain in `.cache/upgrade-0.2.0/`.

## Historical 0.1.1 deployment

The formal MCP container was upgraded to **0.1.1** at 2026-11-01 17:25:35. The real ChatGPT → OpenAI tunnel → Go MCP → SSH/SFTP path passed acceptance, including Chinese downloads, streaming file round trips, and an interactive terminal retaining its environment and working directory across calls. Dedicated test files and sessions were cleaned up.

The development container was neither recreated nor restarted. The existing tunnel, configuration, credentials, workspace and named volumes were reused. No other cloud-dev or Mac development tunnel was modified.

| Check | Observed result |
|---|---|
| MCP version | `tunnel-ssh-mcp version` inside the running container returned `0.1.1` |
| MCP container | `go-tunnel-ssh-mcp-1`, running, restart count 0 |
| Development container | Original `go-tunnel-ssh-dev-1` ID unchanged, `healthy`, restart count 0 |
| Persistence and ports | Mounts unchanged; config and SSH identity read-only to MCP; no published host ports |
| Restart policy | Both services use `unless-stopped` |
| Official health check | `healthz` HTTP 200 / `live`; `readyz` HTTP 200 / `ready`; check requiring a successful control-plane poll returned `result: ok` |
| Real tunnel traffic | As of 17:35:41: 18 forwarded commands and 18 response submissions with HTTP 200; 33 successful polls, including command retrieval HTTP 200 and empty polls HTTP 204 |
| Official doctor | Independent one-shot container returned `RESULT ok`; doctor alone is not treated as live acceptance |
| ChatGPT SSH | Returned `developer`, `/workspace`, `E2E_V011_20261101_B1`, exit code 0 |
| ChatGPT download | `sftp-download` directly returned Chinese text, `encoding: utf8`, `bytes: 33`; no empty object in this run |
| SFTP round trip | Inline upload, streaming download and streaming upload completed; three 33-byte files had identical SHA-256 hashes |
| Persistent PTY | Actual output included `PTY:V011_PTY_B1:/workspace/.e2e-go-tunnel-011-b1:END` |
| Cleanup | `list-sessions` returned `[]` after close; cleanup command exited 0; host checks confirmed old/new test directories absent and no retained SSH test processes in dev |

The close tool returned `process_exit_observed: false`, meaning it had not yet observed a remote exit status. This is not rewritten as synchronous process termination. A subsequent process check found only the listening sshd, confirming no remaining test session.

## Version and deployment evidence

- MCP tags: `go-tunnel-ssh-mcp:latest`, `go-tunnel-ssh-mcp:0.1.1`.
- MCP image: `sha256:5c85dfb439f629a05b1a9f3fd90e477e1527f3696b74e5ff216543cae8043a81`.
- Running Go executable SHA-256: `873dd5feb19b90dc4b7b19a7e7a19d629a3309938aad1e49d54abf4b6e3f5ebe`.
- Development image: `sha256:de4f8528ea967944a30097fb1732771d80aa807889ba17624de5ba8e73b0b114`.

Before upgrading, there were no active SSH sessions. A rollback image was preserved, then the old MCP was stopped before recreating only MCP. Two instances of the same tunnel ID were never run concurrently. Deployment used `docker compose up -d --no-deps --no-build --pull never --force-recreate mcp`.

The new instance logged one EOF warning on its startup metadata request. Before the first successful poll, the poll-required health check failed. The first successful poll arrived about 31 seconds later, followed by successful commands and responses. That startup metadata request is not reported as HTTP 200.

## Real ChatGPT acceptance

The pre-existing dedicated tunnel and account-installed connection were reused. Private account details, tunnel/plugin identifiers and chat URLs are retained only in the local evidence archive and are intentionally omitted from this public record.

The acceptance chat completed connection listing, SSH commands, SFTP download/upload/streaming, persistent terminal checks and cleanup. A temporary webpage response-loading failure was resolved by continuing with an available model. Container operation and tunnel polling continued throughout.

The download directly returned:

```json
{"content":"E2E_V011_B1：中文回读测试\n","encoding":"utf8","bytes":33}
```

`source.txt`, `stream-copy.txt` and `upload-copy.txt` all had SHA-256 `06b3cf70759145885b30f24cec4db0c3ff907453cc291bbc80dce3d3b77ebb48`. After success, only those three files and `.e2e-go-tunnel-011-b1` were removed. The old `.e2e-go-tunnel-20261101-a1` fixture was also removed after rechecking filenames and historical hashes.

Version 0.1.0 already returned full file content over direct stdio, but lacked `outputSchema` and `structuredContent`. Version 0.1.1 uses typed MCP Go SDK output to add both while retaining JSON text. Real ChatGPT downloads no longer showed an empty object in this acceptance run. Compatibility acceptance passed; this does not establish a unique cause for the earlier webpage symptom.

## Separate SSH acceptance inside the formal container

An independent stdio MCP process was also started through `docker compose exec`. It verified version 0.1.1, 13 tools and the download output contract. Real SSH returned `developer`, `/workspace`, `aarch64`, sudo UID 0 and a unique marker. Chinese text, empty files, binary data, missing-file errors, streaming round trips, structured/text equivalence, PTY state and cleanup all passed.

This test reached the formal container's SSH backend **without going through OpenAI**. It is recorded separately from the ChatGPT calls above. Unrelated full source tests were not rerun during deployment; prebuild test results are in the [verification record](verification.md).

## Preserved rollback

The old image index was missing, so both direct tagging and `docker commit` failed with a missing manifest. Before stopping the old container, its root filesystem was exported and imported, restoring its original ENTRYPOINT, CMD, USER, WORKDIR and ENV:

- Rollback tag: `go-tunnel-ssh-mcp:rollback-0.1.0-20261101`.
- Rollback image: `sha256:477934c79009e06cc3f4ed5363bf1cd750516e0df0db8103095387ce56df0030`.
- Starting this image offline with `version` returned `0.1.0`; the old executable hash matched. No live rollback drill was performed.
- The export did not include mounted configuration, SSH identity or workspace data, consistent with [Docker export volume semantics](https://docs.docker.com/reference/cli/docker/container/export/). Existing persistent directories and volumes remain in place.

If rollback is needed, handle active sessions first: these commands end in-memory sessions. They were documented, not executed during this successful upgrade:

```sh
docker compose stop mcp
docker image tag go-tunnel-ssh-mcp:rollback-0.1.0-20261101 go-tunnel-ssh-mcp:latest
docker compose up -d --no-deps --no-build --pull never --force-recreate mcp
docker compose exec -T mcp tunnel-ssh-mcp version
docker compose exec -T mcp tunnel-client health --port 8080 --require-control-plane-poll --json
```

The last check can temporarily fail before the first successful poll; inspect startup logs and subsequent polling. To redeploy 0.1.1, substitute `go-tunnel-ssh-mcp:0.1.1` for the rollback tag, still stopping the old instance before starting the replacement.

## Evidence locations

Evidence is under the ignored `.cache/upgrade-0.1.1/` directory:

- `before.json`, `after.json`, `final-state.json`: container/image identity, mounts and restart state.
- `health-final.json`, `metrics-final.txt`, `startup.log`, `doctor-isolated.log`: health and diagnostics.
- `chatgpt-final.txt`, `chatgpt-verified.png`, `chatgpt-forwarded-events.json`: webpage result, screenshot and 18 real command-forwarding events.
- `chatgpt-file-hashes.json`, `cleanup.json`, `dev-processes-final.txt`: file comparison and cleanup checks.
- `ssh-verification.json`, `stdio-events.json`, `verify_runtime.py`: independent stdio/SSH acceptance.
- `rollback-rootfs.tar`, `rollback-export.json`, `rollback-image.json`: rollback materials.

Real configuration and credentials remain in ignored `data/`; import materials are in `init-input/`. This document and acceptance screenshots contain no keys. Official integration reference: [Secure MCP Tunnel](https://developers.openai.com/api/docs/guides/secure-mcp-tunnels).
