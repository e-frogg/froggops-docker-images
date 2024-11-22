.PHONY: hadolint

# Default target
all: help

# Help target
help:
	@echo "Available commands:"
	@echo "  make hadolint <directory> - Run hadolint on Dockerfile in specified directory"
	@echo "  Example: make hadolint mkdocs"

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

# Catch-all target to allow for directory arguments
%:
	@: 