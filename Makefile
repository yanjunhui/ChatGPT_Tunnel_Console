# Created by Yanjunhui
# Keep build/module caches and temporary test files inside the project.
# 将构建、模块缓存及测试临时文件保留在项目目录内。
export GOPATH := $(CURDIR)/.cache/go
export GOMODCACHE := $(CURDIR)/.cache/mod
export GOCACHE := $(CURDIR)/.cache/build
export TMPDIR := $(CURDIR)/.cache/tmp
TUNNEL_CONSOLE_DESKTOP_IMAGE ?= go-tunnel-console-desktop:latest
TUNNEL_CONSOLE_VNC_IMAGE ?= go-tunnel-console-vnc:latest

.PHONY: prepare build test vet linux install-tunnel docker-build docker-test desktop-build desktop-test check tidy vuln secrets
prepare:
	@mkdir -p "$(TMPDIR)" bin
build: prepare
	go build -trimpath -o bin/tunnel-console-mcp ./cmd/tunnel-console-mcp
	ln -sf tunnel-console-mcp bin/tunnel-ssh-mcp
test: prepare
	go test -race -count=1 -timeout=120s ./...
vet: prepare
	go vet ./...
linux: prepare
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -o bin/tunnel-console-mcp-linux-arm64 ./cmd/tunnel-console-mcp
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -o bin/tunnel-console-mcp-linux-amd64 ./cmd/tunnel-console-mcp
install-tunnel: build
	./bin/tunnel-console-mcp install-tunnel --dir ./bin
docker-test: prepare
	TUNNEL_SSH_DOCKER_TEST=1 go test -race -count=1 -timeout=240s -run '^TestDocker' -v ./cmd/tunnel-console-mcp
docker-build: prepare
	docker compose -f compose.yaml build dev mcp
desktop-build: prepare
	docker build --target desktop -t "$(TUNNEL_CONSOLE_DESKTOP_IMAGE)" .
	docker build --target vnc -t "$(TUNNEL_CONSOLE_VNC_IMAGE)" .
desktop-test: prepare
	TUNNEL_CONSOLE_VNC_TEST=1 TUNNEL_CONSOLE_DESKTOP_IMAGE="$(TUNNEL_CONSOLE_DESKTOP_IMAGE)" TUNNEL_CONSOLE_VNC_IMAGE="$(TUNNEL_CONSOLE_VNC_IMAGE)" go test -race -count=1 -timeout=300s -run '^TestDockerVNC' -v ./cmd/tunnel-console-mcp
check: prepare
	@test -z "$$(gofmt -l cmd internal)" || { gofmt -l cmd internal; exit 1; }
	python3 scripts/check_docs.py
	sh -n deploy/dev-entrypoint.sh deploy/desktop-entrypoint.sh deploy/desktop-session.sh
tidy: prepare
	go mod tidy
vuln: prepare
	go run golang.org/x/vuln/cmd/govulncheck@v1.9.0 ./...
secrets: prepare
	go run github.com/zricethezav/gitleaks/v8@v8.30.1 git --redact --no-banner
