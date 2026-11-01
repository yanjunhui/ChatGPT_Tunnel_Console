# Created by Yanjunhui
FROM node:24-bookworm-slim AS node

FROM golang:1.26.10-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -o /out/tunnel-ssh-mcp ./cmd/tunnel-ssh-mcp

FROM build AS tunnel
# Download and verify the official bundle during build, not at runtime.
# 构建时下载并校验官方发布包，运行时不再下载。
ARG TUNNEL_CLIENT_VERSION=v0.0.19
RUN /out/tunnel-ssh-mcp install-tunnel --version "${TUNNEL_CLIENT_VERSION}" --dir /out/tunnel

FROM debian:bookworm-slim AS runtime
# The runtime image carries MCP and the tunnel; development tools stay in dev.
# 运行镜像只包含 MCP 和隧道组件，开发工具放在 dev 镜像中。
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tini \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --uid 1000 mcp
COPY --from=build /out/tunnel-ssh-mcp /usr/local/bin/tunnel-ssh-mcp
COPY --from=tunnel /out/tunnel/ /usr/local/bin/
USER mcp
WORKDIR /home/mcp
ENTRYPOINT ["/usr/bin/tini", "--", "tunnel-ssh-mcp"]
CMD ["run", "--config", "/etc/tunnel-ssh/current/config.toml"]

FROM debian:bookworm-slim AS dev
COPY --from=runtime /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
RUN sed -i 's|http://deb.debian.org|https://deb.debian.org|g' /etc/apt/sources.list.d/debian.sources \
    && apt-get -o Acquire::Retries=3 update && apt-get -o Acquire::Retries=3 install -y --no-install-recommends \
    openssh-server openssh-client sudo bash ca-certificates curl git make build-essential \
    python3 python3-pip python3-venv cmake ninja-build tmux ripgrep fd-find jq tree tini \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --create-home --shell /bin/bash --uid 1000 developer \
    && passwd -d developer \
    && printf 'developer ALL=(ALL) NOPASSWD: ALL\n' > /etc/sudoers.d/developer
COPY --from=build /usr/local/go /usr/local/go
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY deploy/dev-entrypoint.sh /usr/local/bin/dev-entrypoint
COPY deploy/sshd_config /etc/ssh/sshd_config
# SSH non-login commands may not inherit ENV PATH; provide standard-path symlinks.
# SSH 非登录命令可能不继承 ENV PATH，因此在标准路径中创建工具链接。
RUN chmod 755 /usr/local/bin/dev-entrypoint \
    && ln -s /usr/local/go/bin/go /usr/local/bin/go \
    && ln -s /usr/local/go/bin/gofmt /usr/local/bin/gofmt \
    && ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx \
    && ln -s /usr/bin/fdfind /usr/local/bin/fd
ENV PATH="/usr/local/go/bin:${PATH}"
WORKDIR /workspace
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/dev-entrypoint"]
