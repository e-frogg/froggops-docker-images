ARG FOPS_IMAGE_VERSION
FROM caddy:${FOPS_IMAGE_VERSION}-builder AS builder

RUN xcaddy build \
    --with github.com/caddyserver/transform-encoder \
    --with github.com/e-frogg/fops-caddy-maintenance

FROM caddy:${FOPS_IMAGE_VERSION}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

EXPOSE 443
EXPOSE 443/udp

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]