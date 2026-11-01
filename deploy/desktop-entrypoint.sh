#!/bin/sh
# Created by Yanjunhui
set -eu
/usr/local/bin/dev-entrypoint &
ssh_pid=$!
runuser -u developer -- /usr/local/bin/desktop-session &
desktop_pid=$!
# Stop both services together; Docker can restart a failed desktop or SSH server.
# 同时停止两个服务，使 Docker 能在桌面或 SSH 进程失败时重启容器。
trap 'kill "$ssh_pid" "$desktop_pid" 2>/dev/null || true; wait || true' EXIT
trap 'exit 0' INT TERM
while kill -0 "$ssh_pid" 2>/dev/null && kill -0 "$desktop_pid" 2>/dev/null; do
    sleep 1 &
    wait "$!" || true
done
exit 1
