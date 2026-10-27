# froggops-docker-images

Images are built and published to `ghcr.io/e-frogg/froggops-docker-images-*`
when their Dockerfile or build workflow changes on `main`.

The `fops-ci-php:8.4`, `:8.5` and `:latest` images include Castor 1.9.0.
The `:8.3` image retains Castor 1.5.0 because the 1.9.0 PHAR requires
PHP >=8.4.1. Castor downloads are pinned and verified with SHA-256.
`latest` remains an alias for the PHP 8.4 image.

Builds pull their base images to refresh floating tags before rebuilding.

All `fops-ci-php` images include the Debian `python3` package (standard
library only, no pip) for stdlib-only CI checks.
