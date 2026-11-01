<!-- Created by Yanjunhui -->
# Verification record

English | [简体中文](verification.zh-CN.md)

Date: 2026-11-01 (Asia/Shanghai). Formal runtime status is recorded in live-deployment.md. Early snapshots below describe 0.1.0; later sections record native initialization, Docker initialization, download compatibility and the formal upgrade.

Historical limitations apply to their own stage. Subsequent real-tunnel progress is recorded in [live deployment](live-deployment.md).

## Initial verification: passed

| Check | Result |
|---|---|
| Full `go test -race -count=1 -timeout=120s -json ./...` with `TUNNEL_SSH_DOCKER_TEST=1` | 27 tests passed, 0 failed, 0 skipped; includes a stdio subprocess entry point |
| `go vet ./...` | Passed |
| Native macOS arm64 build | Passed |
| Static Linux arm64 / amd64 cross-compilation | Passed |
| Official tunnel-client download and SHA-256 check | v0.0.19, passed |
| Official executable version and arguments | `0.0.19+01d67bb4da0d8359b808b4761f8db171ab682cd7`; used stdio and credential-reference flags supported |
| Docker MCP and development image builds | linux/arm64, passed |
| Docker MCP → OpenSSH → development environment | Identity, directory, sudo, Go/Python/Node/npm/fd, persistent PTY, SFTP upload/download/streaming passed |
| Test cleanup | Test containers and volumes removed; no running test tunnel |
| Configuration templates | Native and Docker templates parsed successfully |
| Source size | Go functions at most 50 lines; files at most 999 nonblank lines |

Command/SFTP tests use real SSH against a temporary loopback server; Docker tests additionally use real OpenSSH. MCP tests cover SDK in-memory transport and independent stdio subprocesses, discovering 13 tools and executing real SSH calls.

Boundary coverage includes empty commands/files, invalid/unknown configuration, port/buffer bounds, negative/huge/zero timeouts, output truncation and cursors, missing workdirs, nonzero exit, cancellation, concurrent/duplicate sessions, process exit, invalid base64/UTF-8, large streaming transfers, failed downloads preserving an existing file, and arbitrary absolute upload paths.

Fixed and verified at that stage: multiline scripts continuing after a failed workdir change, and Go missing from Docker OpenSSH's default command PATH. Debian HTTP download failures during image construction were resolved using HTTPS sources and official Node/npm image contents.

## Initial image snapshot

```text
go-tunnel-ssh-mcp:latest
sha256:74f67985961ba53351771fa8fc7c20dc9b412bd24c73deaeed9aa9a638f4c3a4

go-tunnel-ssh-dev:latest
sha256:b30b7cae40a042215e86431727316dde947febcec84e522bc5d5c51c42154e68
```

Container results: `developer`, `/workspace`, sudo UID `0`, Go `1.26.10 linux/arm64`, Python `3.11.2`, Node `24.22.2`, npm `11.19.1`, fd `8.6.0`. The MCP image reported `0.1.0`; the official tunnel reported `0.0.19`.

Local build logs: `.cache/docker-build.log`, `.cache/build-checks.log`. Test events: `.cache/test-results.jsonl`. These generated verification files are ignored.

## Unverified at initial delivery

At this stage no real OpenAI tunnel credentials were configured/read, no live tunnel was started and no existing deployment was modified. ChatGPT/Codex → OpenAI tunnel → this service, reconnection and live authentication remained unverified. Local/container tests did not substitute for live acceptance with the user's own tunnel ID and runtime key.

Linux amd64 compiled successfully, but Docker acceptance ran on arm64, not on an amd64 host.

## Native initialization and system-service verification

Date: 2026-11-01 (Asia/Shanghai).

- Full `go test -race -count=1 -timeout=120s -json ./...`: 44 passed, 0 failed, 1 skipped. The skipped test was the separately enabled legacy Docker OpenSSH acceptance, not rerun at this stage.
- `go vet`, native macOS arm64 and Linux arm64/amd64 builds passed.
- Real macOS `plutil -lint` accepted the generated LaunchDaemon plist.
- Real `systemd-analyze verify` in an isolated Linux container accepted generated units, including paths with spaces and percent signs.
- Initialization tests covered prompts, invalid ports, EOF, empty credentials, encrypted keys/wrong passphrases, SSH-agent lifecycle restrictions, environment credential persistence, mode 0600, secret-free output, failed-first-install cleanup, update rollback, repeated updates and concurrent-init locking.
- Real PTY tests covered secret-input editing, Ctrl+C, timeout cancellation and terminal-state restoration.
- A service-command recorder verified stop/register/enable/start/rollback order. No real tunnel credentials were used; no persistent service was installed in the host's `/Library/LaunchDaemons` or `/etc/systemd/system`, and host reboot startup was not tested.

Service-manager activation and authenticated tunnel readiness are different acceptance checks. Linux units use `Type=exec` to surface executable startup failures through the manager; see [systemd.service](https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html). macOS definitions were checked against local `launchd.plist(5)` and `launchctl(1)` manuals.

Logs: `.cache/init-test-results.jsonl`, `.cache/init-build-checks.log`. Isolated definitions: `.cache/service-check/`. Installer implementation: `internal/setup/`; service rendering/control: `internal/service/`.

## Docker initialization verification

Date: 2026-11-01 (Asia/Shanghai). Tests used a rebuilt linux/arm64 product image from the then-current Go source.

| Check | Result |
|---|---|
| Full `go test -race -count=1 -timeout=120s -json ./...` | 54 passed, 0 failed; 2 Docker tests enabled separately next |
| `TUNNEL_SSH_DOCKER_TEST=1 go test -race -count=1 -timeout=240s -json -run '^TestDocker' ./cmd/tunnel-ssh-mcp` | 2 passed, 0 failed, 0 skipped |
| First non-interactive init without config | Failed without publishing current configuration |
| Real `compose.init.yaml` import | Passed while the development container and its SSH key did not yet exist |
| Repeated init after import files were removed | Passed; persisted credentials no longer depended on input files |
| Real Docker PTY reconfiguration | Tunnel/key/default-dev wizard completed, secrets hidden, one-shot container exited normally |
| Runtime Compose configuration access | UID/GID 1000 could read files owned 1000:1000 with mode 0600; config mount rejected writes |
| Generated config → MCP → OpenSSH | Discovery of 13 tools, commands, sudo, toolchain, SFTP round trips and persistent PTY passed after first import, repeated init and interactive reconfiguration |
| Failure/boundary handling | Empty/missing credentials, invalid targets, EOF, mixed flags, invalid UID/GID, publication failure, cancellation and concurrent init covered; failure did not replace old config |
| Native regression | Existing systemd/launchd installation, credential copying, rollback and PTY tests passed |
| Static/build checks | `go vet`, macOS arm64, Linux arm64/amd64 passed; function/file size limits satisfied |
| Compose definitions | Independent init required no existing credentials, ports or service dependencies; runtime config read-only, `unless-stopped` |
| Cleanup | Isolated containers and named volumes removed; no live tunnel started |

Images at this stage:

```text
go-tunnel-ssh-mcp:latest
sha256:9ed327134941a2253fd8e43c7ce0a42013043932d8efa4ad0860edd6a3c3ca32

go-tunnel-ssh-dev:latest
sha256:de4f8528ea967944a30097fb1732771d80aa807889ba17624de5ba8e73b0b114
```

Tests launched an isolated development container through runtime Compose and overrode the MCP command with `serve --config /etc/tunnel-ssh/current/config.toml`. This verified generated configuration through real SSH without connecting invented credentials to OpenAI. The formal image default remained `run --config /etc/tunnel-ssh/current/config.toml`.

No real OpenAI credentials were available at this stage; ChatGPT/Codex live integration and Docker startup after host reboot were not tested. Linux amd64 was compiled, while Docker acceptance used arm64. Native services were not installed on the host.

Logs: `.cache/docker-init-unit-results.jsonl`, `.cache/docker-init-integration-results.jsonl`, `.cache/docker-init-build.log`, `.cache/docker-init-build-checks.log`. Resolved Compose definitions: `.cache/docker-init-compose.json`, `.cache/docker-runtime-compose.json`. All are ignored verification artifacts.

## 0.1.1 download-result compatibility verification

Date: 2026-11-01 (Asia/Shanghai). This section describes the local repair stage before the formal upgrade. The user limited that stage to local verification; the later authorized deployment appears in the next section.

The old direct stdio response contained full file text, and three transferred files had matching content/hashes. The cause of ChatGPT's empty object was not established. The confirmed local contract gap was missing `outputSchema` and `structuredContent`. A new contract test failed against the old implementation; 0.1.1 added SDK typed output while retaining the original text JSON. This change alone did not prove the live symptom's root cause.

| Check | Result |
|---|---|
| Full `go test -race -count=1 -timeout=120s -json ./...` | 55 top-level tests passed (69 pass events including subtests), 0 failed; 2 Docker tests enabled separately |
| `TUNNEL_SSH_DOCKER_TEST=1 go test -race -count=1 -timeout=240s -json -run '^TestDocker' ./cmd/tunnel-ssh-mcp` | 2 passed, 0 failed, 0 skipped |
| Download contract | Discovery included all three required fields and types; structured data exactly matched decoded text JSON |
| Success boundaries | Empty file, Chinese text/newlines, JSON content, base64 containing NUL/non-UTF-8 bytes, exact inline limit |
| Failure boundaries | Missing arguments, empty path, nonexistent file, unknown profile, invalid encoding/UTF-8, over-limit content; errors without successful structured results |
| Isolated Docker/OpenSSH | 13 tools; commands, sudo, toolchain, Chinese inline download, streamed download/upload and readback passed after first init, repeated init and real PTY reconfiguration |
| Persistent terminal | Real OpenSSH PTY retained environment/workdir across tool calls, produced the marker and closed; this was local acceptance |
| Static checks/builds | `go vet`, macOS arm64 and static Linux arm64/amd64 passed; functions/files within size limits |
| Version at that stage | Local binary/new image returned 0.1.1; formal container still returned 0.1.0 with its ID, image and restart count unchanged |
| Cleanup | Isolated containers/networks/volumes removed; readback found only formal containers and persistent volumes |

Local MCP image: `sha256:5c85dfb439f629a05b1a9f3fd90e477e1527f3696b74e5ff216543cae8043a81`, tagged `go-tunnel-ssh-mcp:latest`. Docker tests used Linux arm64. The development image was not rebuilt; it remained the Docker-initialization image. Linux amd64 was cross-compiled only.

Logs/counts under `.cache/download-result/`: `before-tests.log`, `unit-results.jsonl`, `docker-results.jsonl`, `test-summary.json`, `build-checks.log`, `docker-build.log`. `before/` retains target-file snapshots from before that change because the workspace had no Git repository.

Output guidance: [official OpenAI MCP result documentation](https://developers.openai.com/plugins/build/mcp-server#return-useful-results-without-ui); encoding and error behavior were also checked against the locally locked MCP Go SDK v1.8.0. The live download and ChatGPT PTY checks pending at that stage were completed below.

## 0.1.1 formal upgrade and live tunnel acceptance

Date: 2026-11-01 (Asia/Shanghai). Formal MCP upgraded at 17:25:35; evidence/cleanup checked at 17:35:41. The user explicitly authorized this upgrade and tunnel/SSH acceptance. Product source was unchanged during deployment; it used the previously tested 0.1.1 image.

| Check | Result |
|---|---|
| Running version | Formal container returned 0.1.1; image and executable SHA-256 matched deployment records |
| Deployment scope | MCP alone recreated; dev ID/start time unchanged; mounts, credential references, no published ports and restart policies preserved |
| Official health | `health --port 8080 --require-control-plane-poll --json` returned ok; live/ready both HTTP 200 |
| Real control plane | 33 successful polls, 18 forwarded commands, 18 HTTP 200 response submissions; last-success timestamp advanced |
| Formal-container stdio → SSH | 13 tools, version, output contract, identity/workdir/sudo, Chinese/empty/binary/missing files, streaming and persistent PTY passed; this group did not pass through OpenAI |
| ChatGPT → tunnel → SSH | Actual command returned developer, /workspace, dedicated marker and exit code 0 |
| ChatGPT SFTP | Direct download returned `E2E_V011_B1：中文回读测试\n`, utf8, 33 bytes; three files matched after inline/streaming transfers |
| ChatGPT PTY | Two input calls retained variables/workdir; actual output `PTY:V011_PTY_B1:/workspace/.e2e-go-tunnel-011-b1:END` |
| Session cleanup | Close had not yet observed exit; subsequent list was `[]`, and container process check found no remaining SSH session |
| File cleanup | Old/new dedicated acceptance directories and stdio test directory removed; other work files retained |
| Rollback materials | 0.1.0 root-filesystem image preserved, startup config restored, offline version check passed; no live rollback |

The empty ChatGPT download object did not recur in this run; compatibility acceptance passed without claiming a unique cause for the old issue. The webpage temporarily failed to load replies and showed a model limit; switching to an available model completed acceptance while the tunnel remained operational.

See [live deployment](live-deployment.md) for full runtime identifiers, raw evidence paths, transient startup EOF, rollback commands and the acceptance chat. Network-failure reconnection, host-reboot recovery and amd64 hardware execution were not tested in this normal-function acceptance stage.

## English-first documentation and bilingual comments

Date: 2026-11-01 (Asia/Shanghai), after the formal upgrade. This change updates documentation and source/configuration comments only; the application version remains 0.1.1. Runtime messages and MCP descriptions are unchanged.

| Check | Result |
|---|---|
| Source equivalence | All 45 Go files have identical non-comment token streams before/after the change |
| Comment coverage | Exported production declarations documented; explanatory Go comments contain English and Chinese; function-size limits satisfied |
| Configuration equivalence | Three TOML templates and all three resolved Compose definitions unchanged; Dockerfile, Makefile, shell and SSH configuration have identical non-comment content |
| Documentation | 18 Markdown files in 9 English/Chinese pairs; local links, paired code examples, SHA-256 values, headers, fences and file-size limits checked |
| Local regression | `go test -race -count=1 -timeout=120s -json ./...`: 55 top-level tests passed (69 pass events including subtests), 0 failed; 2 opt-in Docker tests skipped |
| Static/build checks | `make vet build` and shell syntax check passed |
| Deployment scope | No Docker image rebuild or formal-service restart; this stage did not repeat live ChatGPT acceptance or Docker integration tests |

Evidence under ignored `.cache/docs-bilingual/`: `before/`, `test-results.jsonl`, `test-summary.json`, `build-checks.log`, `deployment-config-checks.json`, `document-checks.json`, and the source/document verification helpers. Historical live results above remain tied to their original acceptance stage.

## 0.1.2 public-source release review

Date: 2026-11-01 (Asia/Shanghai). This review prepares the public MIT source repository. It does not deploy 0.1.2: the formal MCP remains 0.1.1, with container IDs, images, startup times and restart counts unchanged.

| Check | Result |
|---|---|
| CLI defect reproduction | Five subcommands originally returned exit 1 for `--help`; the regression failed before the fix and passed afterward. All six flag-based subcommands now return exit 0. |
| Local tests and builds | Full race suite, `go vet`, native build, Linux arm64/amd64 cross-builds and module tidy passed on macOS arm64 with Go 1.26.10. |
| Docker acceptance | Both opt-in tests passed with separate `public-0.1.2` image tags: initialization, wizard, persistence/permissions, real OpenSSH, PTY state and SFTP round trips. Test containers and volumes were removed. |
| Documentation and source checks | 26 documents in 13 bilingual pairs passed link/example/hash checks; 47 Go files passed public-comment, bilingual-comment and function-length checks. |
| Clean checkout and Linux ownership | CI reproduced a missing build temporary directory, then native-Linux fixture ownership failures. The build now prepares its directory; Docker fixtures grant UID 1000 workspace access and restore host ownership on cleanup. Configuration remains private. |
| Reachable vulnerability scan | govulncheck v1.9.0 reports no reachable or imported-package vulnerabilities after raising the toolchain from 1.26.0 to 1.26.10. |
| Module-only advisory | GO-2026-5932 concerns the unmaintained `golang.org/x/crypto/openpgp` package. This project does not import it; the advisory has no fixed version. This is recorded rather than suppressed. |
| Public content | Gitleaks v8.30.1 found no secrets in the staged source export. Private deployment identifiers were redacted; credentials, runtime data, binaries and raw logs are excluded. |

Local evidence is retained under `.cache/public-release/` and is deliberately not published. GitHub Actions independently runs Linux/macOS checks and Linux Docker integration; consult the repository's Actions page for the status of a particular commit. Native boot-service installation and a new ChatGPT end-to-end run were not performed in this phase; the earlier 0.1.1 live acceptance remains a separate historical record.

## 0.2.0 Console rename and VNC acceptance

Date: 2026-11-01 (Asia/Shanghai). This is a source release, with separate candidate Docker image tags. The formal 0.1.1 containers retain their IDs, images, start times and zero restart counts; the existing supervisor's required control-plane-poll health check passes.

| Check | Result |
|---|---|
| Configuration and initialization | Old profiles leave VNC disabled; defaults, invalid endpoints/timeouts/references, desktop wizard choice and persisted VNC credentials pass. Imported credentials remain usable after source files are removed. |
| MCP contract | 17 discoverable tools, including four VNC tools; screenshot returns native PNG image content and structured width/height/name; only screenshots are annotated read-only among desktop tools. |
| RFB protocol | 3.3/3.7/3.8, None/VncAuth, independent DES challenge vector, rejected authentication/no downgrade, malformed rectangles, overlapping/partial frames, resize and complete pixel coverage pass. |
| Inputs and lifecycle | Pointer/chord/text bounds, ordered press/release, CRLF, cancellation, deadline and manager shutdown over real SSH forwarding pass. |
| Real desktop | MCP stdio → OpenSSH → TigerVNC returns 1280×800 PNGs; actual pointer reaches (100,100); MCP keyboard input runs a command in xterm whose file reads back `VNC_KEYBOARD_OK\n`. A returned screenshot was visually inspected. |
| Authentication | A second real TigerVNC server returns a 64×48 PNG with the correct persisted password, rejects a wrong password and does not echo password values. |
| Docker deployment | Base SSH/SFTP/PTY/init regression and the desktop Compose overlay pass on Linux arm64. Desktop health checks SSH/X display; no VNC/SSH/MCP host ports are published. Temporary resources are cleaned. |
| Local checks | Full race suite, vet, native macOS arm64 build and Linux arm64/amd64 cross-builds pass; module tidy adds no dependency. 30 Markdown files form 15 checked bilingual pairs; 63 Go files pass declaration/comment/function limits. |
| Fuzzing | Ten-second RFB handshake fuzz run: 8,291,917 executions, no crash. This is bounded fuzz evidence, not exhaustive protocol proof. |
| Security review | No reachable or imported-package Go vulnerabilities; the previously recorded module-only OpenPGP advisory remains. Redacted history and publication-source scans are part of release validation. |

A window-title lookup initially failed because the terminal shell rewrites its title; acceptance now locates the visible XTerm class. A later Docker Desktop configuration-permission assertion failed once without a diagnostic. The test now prints safe ownership/mode information on failure; its subsequent full run passed. No permission relaxation or retry workaround was added, and the single transient failure's cause is unconfirmed.

Evidence is private under `.cache/console-vnc/`, including the returned desktop PNG and test/build/scan logs. The public repository is now `ChatGPT_Tunnel_Console`; deployment identifiers are retained for upgrade compatibility. GitHub Actions checks Linux/macOS and Docker independently for each commit. No real ChatGPT VNC call, formal 0.2.0 upgrade, external desktop installation, reboot recovery or network-failure reconnection was performed in this stage. Earlier 0.1.1 live tunnel acceptance remains separate.

## 0.3.0 independent desktop acceptance

Date: 2026-11-01 (Asia/Shanghai). The native backend uses X11 over SSH; the optional VNC backend remains independently configured. No new Go dependency was introduced.

| Check | Observed result |
|---|---|
| Local regression | Full race suite, vet, macOS arm64 build and Linux arm64/amd64 cross-builds passed. |
| Codec and configuration | XWD 24/32-bit storage, both byte orders, padded rows, original pixel values, invalid headers/dimensions/color tables and all truncation points passed; native display/timeout validation and import copying passed. |
| Lifecycle | Real SSH cancellation, manager shutdown and queued-operation deadlines release connections and the desktop gate. |
| Actual desktop independence | With no VNC program installed in dev and no VNC service started, native screenshot, pointer, chord, Chinese typing and Enter passed. Both paths then passed together; after stopping VNC, VNC calls failed and native operations still passed. |
| Observable input effects | Pointer position read back as (100,100). Separate xterm commands produced four exact markers with Chinese text; files were read back through SSH. Original 1280×800 MCP PNGs were decoded and visually inspected. |
| VNC-only configuration | A separate profile without native settings used password-authenticated x11vnc successfully; incorrect passwords were rejected without disclosure. |
| Docker regression | Initialization, permissions, real OpenSSH/SFTP/PTY and toolchain checks passed on arm64; temporary test containers and volumes were removed. |
| Fuzzing | Ten-second XWD fuzz run: 6,762,395 executions, no crash; bounded evidence, not exhaustive proof. |
| Documentation and source | 32 Markdown files / 16 bilingual pairs passed; 72 Go files passed public/bilingual comments and function-size checks. |
| Security | No reachable or imported-package Go vulnerabilities; the existing module-only OpenPGP advisory remains. History and publication-source scans use redacted output. |

Reproduced and fixed during acceptance: an ASCII SSH locale rejected Unicode; 1 ms xdotool input could lose temporary Unicode mappings (12 ms restored readback); `mousemove --sync` stalled at unchanged coordinates (exit 124 under a one-second reproduction); cross-container MIT-SHM access failed, so the separate VNC service uses socket capture with `-noshm`. IPv6 listeners are disabled in that sidecar.

Evidence is private under `.cache/independent-desktop/`. These local checks do not substitute for the real ChatGPT calls recorded separately in [live deployment](live-deployment.md). Host reboot recovery, network-failure recovery and external native desktop installation were not tested in this stage.
