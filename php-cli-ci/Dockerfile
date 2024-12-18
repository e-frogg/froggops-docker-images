ARG FOPS_IMAGE_VERSION

FROM composer/composer:2-bin AS composer
FROM mlocati/php-extension-installer:latest AS php_extension_installer

FROM php:${FOPS_IMAGE_VERSION}-cli-alpine

# install deps packages + php extensions
COPY --from=php_extension_installer --link /usr/bin/install-php-extensions /usr/local/bin/

RUN apk add --update --no-cache \
    openssh-client \
    git \
    bash \
    libxslt-dev \
    libpng-dev \
    icu \
    icu-dev \
    libzip-dev \
    bzip2-dev \
    file \
    docker \
    docker-cli \
    docker-compose \
    curl \
    make \
    sudo \
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
        amqp ;

# composer install
ENV COMPOSER_ALLOW_SUPERUSER=1
ENV PATH="${PATH}:/root/.composer/vendor/bin"
COPY --from=composer --link /composer /usr/bin/composer

# castor (outil supplémentaire)
RUN composer global require jolicode/castor

# Configuration PHP
COPY --link ./php.ini $PHP_INI_DIR/conf.d/