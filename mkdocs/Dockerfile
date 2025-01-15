ARG FOPS_IMAGE_VERSION
FROM squidfunk/mkdocs-material:${FOPS_IMAGE_VERSION}
LABEL org.opencontainers.image.source=https://github.com/e-frogg/froggops-docker-images
LABEL org.opencontainers.image.description="Mkdocs with custom plugins"
LABEL org.opencontainers.image.licenses=MIT

# hadolint ignore=DL3013
RUN pip install --no-cache-dir markdown-callouts mkdocs-literate-nav mkdocs-glightbox
