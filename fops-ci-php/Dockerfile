ARG FOPS_IMAGE_VERSION
ARG CDPX_IMAGE=ghcr.io/inem0o/cdpx:0.1.4@sha256:9e0e7209b2e073764abc96f5effe97f76d445b4e244ea860d90ac717041bc43f

FROM ${CDPX_IMAGE} AS cdpx

FROM node:26-trixie AS node

FROM composer/composer:2-bin AS composer
FROM mlocati/php-extension-installer:latest AS php_extension_installer

FROM php:${FOPS_IMAGE_VERSION}-cli


#
# PHP deps
#


COPY --from=php_extension_installer --link /usr/bin/install-php-extensions /usr/local/bin/

RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-client \
    git \
    bash \
    libxslt1-dev \
    libpng-dev \
    libicu-dev \
    libzip-dev \
    libbz2-dev \
    file \
    curl \
    make \
    sudo \
    ca-certificates \
    rsync \
    jq \
    && rm -rf /var/lib/apt/lists/* \
    && set -eux; \
    install-php-extensions \
        intl \
        zip \
        apcu \
        opcache \
        sockets \
        xsl \
        gd \
        bz2 \
        bcmath \
        pcntl \
        pcov \
        pdo \
        pdo_mysql \
        soap \
        redis \
        amqp



COPY --link ./php.ini $PHP_INI_DIR/conf.d/

ENV COMPOSER_ALLOW_SUPERUSER=1
ENV PATH="${PATH}:/root/.composer/vendor/bin"
ENV DISABLE_VERSION_CHECK=1
COPY --from=composer --link /composer /usr/bin/composer

RUN curl "https://github.com/jolicode/castor/releases/latest/download/castor.linux-amd64" -Lfso /usr/bin/castor && \
    chmod u+x /usr/bin/castor && \
    castor --version || \
    (echo "Could not install castor. Is the target directory writeable?" && (exit 1))

#
# Nodes deps
#
RUN apt-get update && apt-get install -y --no-install-recommends \
    curl \
    tar \
    gzip \
    && rm -rf /var/lib/apt/lists/*

COPY --from=node /usr/lib /usr/lib
COPY --from=node /usr/local/lib /usr/local/lib
COPY --from=node /usr/local/include /usr/local/include
COPY --from=node /usr/local/bin /usr/local/bin

ENV PATH="/usr/local/lib/node_modules/.bin:${PATH}"

RUN node --version && npm --version


#
# Docker client deps
#



RUN apt-get update
RUN apt-get install -y ca-certificates curl
RUN install -m 0755 -d /etc/apt/keyrings
RUN curl -fsSL https://download.docker.com/linux/debian/gpg -o /etc/apt/keyrings/docker.asc
RUN chmod a+r /etc/apt/keyrings/docker.asc

RUN echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/debian \
  $(. /etc/os-release && echo "$VERSION_CODENAME") stable" | \
  tee /etc/apt/sources.list.d/docker.list > /dev/null
RUN apt-get update

RUN apt-get install -y docker-ce-cli docker-buildx-plugin docker-compose-plugin




#
# Tools / System
#

RUN apt-get update && apt-get install -y --no-install-recommends \
    libnss3-tools \
    && rm -rf /var/lib/apt/lists/* \
    && set -eux; \
    mkdir -p /root/.pki/nssdb \
    && certutil -d /root/.pki/nssdb -N


# cdpx embedded bundle
# cdpx 0.1.4 must invoke the real installer path because /opt/cdpx/install
# resolves its bundle root incorrectly when called through its symlink.
COPY --from=cdpx /opt/cdpx /opt/cdpx

RUN set -eux; \
    /opt/cdpx/bin/embedded-install --link /usr/local/bin/cdpx; \
    cdpx --version; \
    php -S 127.0.0.1:8099 -t /opt/cdpx/site-packages \
        >/tmp/cdpx-smoke-server.log 2>&1 & \
    server_pid=$!; \
    trap 'cdpx session stop >/dev/null 2>&1 || true; kill "$server_pid" 2>/dev/null || true' 0; \
    attempts=0; \
    until curl -fsS http://127.0.0.1:8099/cdpx/__init__.py >/dev/null; do \
        attempts=$((attempts + 1)); \
        [ "$attempts" -lt 50 ]; \
        sleep 0.1; \
    done; \
    eval "$(cdpx session start \
        --run-id fops-ci-image-smoke \
        --authority observation \
        --origins 'http://127.0.0.1:8099' \
        --ttl 60 \
        --export)"; \
    cdpx goto http://127.0.0.1:8099/cdpx/__init__.py >/dev/null; \
    cdpx wait body >/dev/null; \
    cdpx session stop >/dev/null; \
    kill "$server_pid"; \
    wait "$server_pid" 2>/dev/null || true; \
    rm -f /tmp/cdpx-smoke-server.log; \
    trap - 0


COPY --from=gitlab/glab:latest /usr/bin/glab /usr/local/bin/glab
