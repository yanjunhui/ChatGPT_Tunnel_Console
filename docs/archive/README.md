<!-- Created by Yanjunhui -->
# Project archive

English | [简体中文](README.zh-CN.md) | [Documentation](../README.md)

These documents preserve implementation decisions and checks recorded on 2026-11-01. They describe the revisions and environments named in each record, not the current state of a running installation. For current setup and operation, use the [installation guide](../installation.md) and [usage guide](../usage.md).

## Verification and deployment records

- [Verification record](verification.md) — automated checks and integration results by revision.
- [Deployment record](live-deployment.md) — recorded deployments, live tunnel checks and rollback context.

The records retain their original results, versions and evidence references. Private evidence paths may only be available to the deployment owner. Test results are limited to the checks described; a local test, tunnel health check and real ChatGPT tool call are different forms of evidence.

## Implementation plans

- [SSH/SFTP console](plans/2026-11-01-go-tunnel-ssh.md)
- [Docker initialization](plans/2026-11-01-docker-init.md)
- [Native installation service](plans/2026-11-01-init-service.md)
- [Desktop and VNC support](plans/2026-11-01-console-vnc.md)

These plans are historical development documents. Their task lists are not an installation procedure or a current product roadmap. User-facing behavior is documented in the guides and [tool reference](../tool-reference.md).
