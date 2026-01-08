
ARG FOPS_IMAGE_VERSION

FROM caddy:${FOPS_IMAGE_VERSION}-builder AS builder

RUN xcaddy build \
    --with github.com/caddyserver/transform-encoder \
    --with github.com/caddy-dns/ovh \
    --with github.com/e-frogg/fops-caddy-maintenance

FROM caddy:${FOPS_IMAGE_VERSION}

COPY --from=builder /usr/bin/caddy /usr/bin/caddy

RUN apk add --no-cache bash curl
COPY --link rootfs /

EXPOSE 443
EXPOSE 443/udp

# Définir umask 022 pour que les fichiers de log soient 0644 (lisibles par tous)
CMD ["sh", "-c", "umask 022 && exec caddy run --config /etc/caddy/Caddyfile --adapter caddyfile"]
