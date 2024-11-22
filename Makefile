.PHONY: hadolint trivy

# Default target
all: help

# Help target
help:
	@echo "Available commands:"
	@echo "  make hadolint <directory>                      - Run hadolint on Dockerfile in specified directory"
	@echo "  make trivy <directory> [FOPS_IMAGE_VERSION=x]  - Run trivy scanner on local image for specified directory"
	@echo "  Example: make hadolint mkdocs"
	@echo "  Example: make trivy mkdocs"
	@echo "  Example: make trivy mkdocs FOPS_IMAGE_VERSION=1.0.0"

# Hadolint target
hadolint:
	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		echo "Please specify a directory. Example: make hadolint mkdocs"; \
		exit 1; \
	fi
	@for arg in $(filter-out $@,$(MAKECMDGOALS)); do \
		if [ ! -f "$$arg/Dockerfile" ]; then \
			echo "Error: $$arg/Dockerfile does not exist"; \
			exit 1; \
		fi; \
		echo "Running hadolint on $$arg/Dockerfile..."; \
		docker run --rm -i hadolint/hadolint < "$$arg/Dockerfile"; \
	done

# Trivy target
trivy:
	@if [ -z "$(filter-out $@,$(MAKECMDGOALS))" ]; then \
		echo "Please specify a directory. Example: make trivy mkdocs"; \
		exit 1; \
	fi
	@for arg in $(filter-out $@,$(MAKECMDGOALS)); do \
		if [ ! -f "$$arg/Dockerfile" ]; then \
			echo "Error: $$arg/Dockerfile does not exist"; \
			exit 1; \
		fi; \
		echo "Building image for $$arg..."; \
		docker build -t "local/$$arg:test" "$$arg" \
			$(if $(FOPS_IMAGE_VERSION),--build-arg FOPS_IMAGE_VERSION=$(FOPS_IMAGE_VERSION),); \
		echo "Running trivy scan on local/$$arg:test..."; \
		docker run --rm -v /var/run/docker.sock:/var/run/docker.sock aquasec/trivy image \
			--severity HIGH,CRITICAL \
			--ignore-unfixed \
			--exit-code 1 \
			"local/$$arg:test"; \
		docker rmi "local/$$arg:test"; \
	done

# Catch-all target to allow for directory arguments
%:
	@: 