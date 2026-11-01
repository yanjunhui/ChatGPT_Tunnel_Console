<!-- Created by Yanjunhui -->
# Third-party notices

English | [简体中文](THIRD_PARTY_NOTICES.zh-CN.md)

Original project code is licensed under [MIT](LICENSE). Dependencies and external tools retain their upstream licenses; the repository's license does not relicense them. This source repository does not vendor those components or publish compiled binaries/images.

## Direct Go dependencies

Versions are locked by `go.mod` and checksums by `go.sum`.

| Component | Version | Upstream license |
|---|---|---|
| BurntSushi/toml | v1.6.0 | [MIT](https://github.com/BurntSushi/toml/blob/v1.6.0/COPYING) |
| creack/pty | v1.1.24 | [MIT](https://github.com/creack/pty/blob/v1.1.24/LICENSE) |
| modelcontextprotocol/go-sdk | v1.8.0 | [Apache-2.0 and legacy MIT contributions](https://github.com/modelcontextprotocol/go-sdk/blob/v1.8.0/LICENSE); upstream documentation uses CC-BY-4.0 |
| pkg/sftp | v1.13.12 | [BSD-2-Clause](https://github.com/pkg/sftp/blob/v1.13.12/LICENSE) |
| golang.org/x/crypto | v0.58.0 | [BSD-3-Clause](https://github.com/golang/crypto/blob/v0.58.0/LICENSE) |
| golang.org/x/sys | v0.49.0 | [BSD-3-Clause](https://github.com/golang/sys/blob/v0.49.0/LICENSE) |
| golang.org/x/term | v0.47.0 | [BSD-3-Clause](https://github.com/golang/term/blob/v0.47.0/LICENSE) |

The MCP SDK's licensing notice describes an ongoing transition: contributions without relicensing consent retain MIT terms. Do not describe the entire SDK as exclusively MIT or exclusively Apache-2.0. Consult its exact version's LICENSE for the applicable terms.

Transitive Go dependencies are listed in `go.mod` and can be enumerated with `go list -m all`. Their license files remain in the downloaded module sources. When distributing a binary, collect the notices and license texts for the actual linked dependencies, not only this list of direct dependencies.

## External runtime and development tools

- [OpenAI tunnel-client v0.0.19](https://github.com/openai/tunnel-client/blob/v0.0.19/LICENSE) uses Apache-2.0 and is downloaded from its official release. Its bundle includes other components such as [Cloudflare cloudflared](https://github.com/cloudflare/cloudflared/blob/master/LICENSE), which retain their own terms.
- The Dockerfile uses official Go, Node.js and Debian images and installs Debian packages. Their license and source-availability requirements remain applicable, including to copies made into the development image.
- The optional desktop image installs TigerVNC, Openbox, xterm, X11 utilities and fonts under their respective package licenses; this project's MIT license does not relicense them. Exact copyright/license inventories are in `/usr/share/doc/<package>/copyright` inside the built image; review applicable source and notice obligations before redistribution. The Go RFB client is independently implemented here and does not vendor a third-party VNC client library.
- govulncheck and Gitleaks are development checks downloaded separately; their versions are pinned in the Makefile. They are not part of the MCP runtime.

Before redistributing a built image or executable, retain all required upstream copyright, license and NOTICE files and satisfy any applicable source-distribution obligations. This document is an inventory for this source release, not a replacement for upstream license texts or a complete binary-distribution notice bundle.

OpenAI, ChatGPT, Codex and other product names identify compatible services or dependencies. This independent project is not an official OpenAI product and claims no affiliation or endorsement.
