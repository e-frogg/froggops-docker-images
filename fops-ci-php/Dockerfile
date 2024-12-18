ARG FOPS_IMAGE_VERSION

FROM node:23 AS node

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
COPY --from=composer --link /composer /usr/bin/composer

RUN composer global require jolicode/castor \
    && ln -s /usr/bin/castor /root/.composer/vendor/bin/castor

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

RUN npx playwright install
RUN npx playwright install-deps


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


