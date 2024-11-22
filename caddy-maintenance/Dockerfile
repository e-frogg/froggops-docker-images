FROM caddy:2-builder AS builder

RUN xcaddy build \
    --with github.com/caddyserver/transform-encoder \
    --with github.com/e-frogg/fops-caddy-maintenance

FROM caddy:2

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

# Default Caddy ports
EXPOSE 443
EXPOSE 443/udp

CMD ["caddy", "run", "--config", "/etc/caddy/Caddyfile", "--adapter", "caddyfile"]