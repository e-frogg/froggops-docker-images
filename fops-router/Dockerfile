
ARG FOPS_IMAGE_VERSION=2.10.2

FROM caddy:${FOPS_IMAGE_VERSION}-builder AS builder

WORKDIR /src

COPY --link plugins/fops-caddy-router ./plugins/fops-caddy-router

RUN xcaddy build \
    --with github.com/caddyserver/transform-encoder@15eef974326185626c6d33e5e5f7f452dc6518b5 \
    --with github.com/caddy-dns/ovh@v1.1.0 \
    --with github.com/e-frogg/fops-caddy-maintenance@v0.1.9 \
    --with github.com/e-frogg/fops-caddy-router=./plugins/fops-caddy-router

FROM caddy:${FOPS_IMAGE_VERSION}

COPY --from=builder /src/caddy /usr/bin/caddy

RUN apk add --no-cache bash curl
COPY --link rootfs /
RUN chmod 755 /usr/local/bin/docker-entrypoint.sh

EXPOSE 80
EXPOSE 443
EXPOSE 443/udp

ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]
