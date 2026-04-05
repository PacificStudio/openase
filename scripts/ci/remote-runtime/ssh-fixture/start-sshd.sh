#!/bin/sh
set -eu

mkdir -p /run/sshd /home/openase/.ssh /home/openase/.config/systemd/user /home/openase/.openase/logs /home/openase/.openase/fake-systemd
touch /home/openase/.ssh/authorized_keys
chmod 700 /home/openase/.ssh
chmod 600 /home/openase/.ssh/authorized_keys
chown -R openase:openase /home/openase/.ssh /home/openase/.config /home/openase/.openase

exec /usr/sbin/sshd -D -e
