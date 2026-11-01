#!/bin/sh
# Created by Yanjunhui
set -eu
mkdir -p /run/sshd /var/lib/tunnel-ssh /home/developer/.ssh /workspace
# Keep client and host identities in the shared volume across container replacement.
# 将客户端和主机身份保存在持久卷中，使容器重建后仍可复用。
if [ ! -f /var/lib/tunnel-ssh/client_key ]; then
    ssh-keygen -q -t ed25519 -N '' -f /var/lib/tunnel-ssh/client_key
fi
if [ ! -f /var/lib/tunnel-ssh/host_key ]; then
    ssh-keygen -q -t ed25519 -N '' -f /var/lib/tunnel-ssh/host_key
fi
cp /var/lib/tunnel-ssh/client_key.pub /home/developer/.ssh/authorized_keys
chown developer:developer /home/developer /home/developer/.ssh /home/developer/.ssh/authorized_keys
chown developer:developer /var/lib/tunnel-ssh/client_key
chmod 700 /home/developer/.ssh
chmod 600 /home/developer/.ssh/authorized_keys /var/lib/tunnel-ssh/client_key
exec /usr/sbin/sshd -D -e -f /etc/ssh/sshd_config
