#!/bin/sh
# Created by Yanjunhui
set -eu
export DISPLAY=:1
export LANG=C.UTF-8
Xvfb :1 -screen 0 1280x800x24 -nolisten tcp &
display_pid=$!
# Native capture/input use this local display without any VNC process.
# 原生截图和输入直接使用本地显示服务，不需要 VNC 进程。
trap 'kill "$display_pid" 2>/dev/null || true; wait || true' EXIT
trap 'exit 0' INT TERM
attempt=0
until xdpyinfo >/dev/null 2>&1; do
    kill -0 "$display_pid"
    attempt=$((attempt + 1))
    [ "$attempt" -lt 50 ] || exit 1
    sleep 0.1
done
openbox &
xterm -u8 -fa 'DejaVu Sans Mono' -fs 12 -geometry 100x30+40+50 -title 'ChatGPT Tunnel Console' &
wait "$display_pid"
