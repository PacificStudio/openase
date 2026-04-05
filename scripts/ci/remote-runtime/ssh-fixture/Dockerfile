FROM debian:bookworm-slim

RUN apt-get update \
    && apt-get install -y --no-install-recommends bash ca-certificates openssh-server procps \
    && rm -rf /var/lib/apt/lists/*

RUN useradd --create-home --shell /bin/bash openase \
    && mkdir -p /run/sshd /home/openase/.ssh /home/openase/.config/systemd/user /home/openase/.openase/logs /home/openase/.openase/fake-systemd \
    && chown -R openase:openase /home/openase

COPY scripts/ci/remote-runtime/ssh-fixture/systemctl /usr/local/bin/systemctl
COPY scripts/ci/remote-runtime/ssh-fixture/journalctl /usr/local/bin/journalctl
COPY scripts/ci/remote-runtime/ssh-fixture/start-sshd.sh /usr/local/bin/start-sshd.sh

RUN chmod +x /usr/local/bin/systemctl /usr/local/bin/journalctl /usr/local/bin/start-sshd.sh \
    && printf '%s\n' \
        'Port 22' \
        'PubkeyAuthentication yes' \
        'PasswordAuthentication no' \
        'ChallengeResponseAuthentication no' \
        'UsePAM no' \
        'PermitRootLogin no' \
        'AllowUsers openase' \
        'AuthorizedKeysFile .ssh/authorized_keys' \
        >> /etc/ssh/sshd_config

ENTRYPOINT ["/usr/local/bin/start-sshd.sh"]
