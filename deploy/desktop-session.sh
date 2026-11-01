#!/bin/sh
# Created by Yanjunhui
set -eu
export DISPLAY=:1
Xtigervnc :1 -localhost yes -SecurityTypes None -geometry 1280x800 -depth 24 -AlwaysShared -nolisten tcp &
vnc_pid=$!
# VNC stays on loopback; the MCP client reaches it through authenticated SSH.
# VNC 仅监听回环地址，由 MCP 客户端通过已鉴权 SSH 访问。
trap 'kill "$vnc_pid" 2>/dev/null || true; wait || true' EXIT
trap 'exit 0' INT TERM
attempt=0
until xdpyinfo >/dev/null 2>&1; do
    kill -0 "$vnc_pid"
    attempt=$((attempt + 1))
    [ "$attempt" -lt 50 ] || exit 1
    sleep 0.1
done
openbox &
xterm -u8 -fa 'DejaVu Sans Mono' -fs 12 -geometry 100x30+40+50 -title 'ChatGPT Tunnel Console' &
wait "$vnc_pid"
