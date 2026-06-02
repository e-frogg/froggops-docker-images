#!/bin/sh
set -e

umask 022

mkdir -p \
	/run/caddy \
	/data/fops-router/generated \
	/data/fops-router/runtime \
	/etc/fops-router/entrypoints

printf '# Runtime CADDY_MAIN_DIRECTIVES\n%s\n' "${CADDY_MAIN_DIRECTIVES:-}" > /data/fops-router/runtime/main-directives.Caddyfile
printf '# Runtime CADDY_VHOSTS\n%s\n' "${CADDY_VHOSTS:-}" > /data/fops-router/runtime/vhosts.Caddyfile

if [ ! -f /data/fops-router/generated/routes.Caddyfile ]; then
	printf '# Dynamic fops-router routes\n' > /data/fops-router/generated/routes.Caddyfile
fi

exec "$@"
