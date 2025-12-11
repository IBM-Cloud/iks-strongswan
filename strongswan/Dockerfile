FROM alpine:3.23

RUN apk update && \
    apk upgrade && \
    apk add bash conntrack-tools curl iptables iptables-legacy nmap strongswan sudo && \
    apk info -v && \
    rm -f /var/cache/apk/*

EXPOSE 500/udp 4500/udp

RUN addgroup -S --gid 2000 strongswan && \
    adduser -S --uid 2000 -h /home/strongswan strongswan -G strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /sbin/ip' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /usr/sbin/iptables' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /usr/sbin/iptables-legacy' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /usr/bin/nmap' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /bin/cp' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /bin/ls' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /bin/sed' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /bin/kill' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /sbin/sysctl' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /usr/sbin/ipsec' >> /etc/sudoers.d/strongswan
RUN echo '%strongswan ALL=(ALL) NOPASSWD: /usr/sbin/conntrack' >> /etc/sudoers.d/strongswan

# When we reset the image to discard history, the setuid bit is lost on the sudo and ping commands.
# We put the bit back here.
RUN chmod 4755 /usr/bin/sudo
RUN chmod 4755 /bin/ping

COPY --chown=strongswan:strongswan calicoctl        /usr/local/bin/calicoctl
COPY --chown=strongswan:strongswan calicoCmd.sh     /usr/local/bin/calicoCmd
COPY --chown=strongswan:strongswan runHelmTest.sh   /usr/local/bin/runHelmTest
COPY --chown=strongswan:strongswan vpnDebug.sh      /usr/local/bin/vpnDebug
COPY --chown=strongswan:strongswan strongswan       /usr/local/bin/strongswan

RUN chown -R strongswan:strongswan /tmp
RUN mkdir /calico-secrets
RUN chown -R strongswan:strongswan /calico-secrets

USER strongswan:strongswan

ENTRYPOINT ["/usr/local/bin/strongswan"]
