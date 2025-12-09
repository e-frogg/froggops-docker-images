AGENTS // fops-clock
====================

Mission
-------
- Build and maintain a static Go binary (`/fops-clock`) that can be injected into any image to emulate cron on glibc, musl, and distroless bases.
- Honor the *self-contained & lightweight* rule: no runtime deps beyond the binary (`CGO_ENABLED=0`, final stage `scratch`).
- Guarantee a consistent injection path via `COPY --from=ghcr.io/e-frogg/froggops-docker-images-fops-clock:tag /fops-clock /usr/local/bin/fops-clock`.

Runtime specification
---------------------
- Process must run as PID 1: handle `SIGINT`, `SIGTERM`, `SIGCHLD`, reap zombies, and exit cleanly.
- Loop responsibilities: compute next tick via `robfig/cron/v3`, sleep with interrupt support, execute the command, log with `[FOPS-CLOCK]` prefix.
- Shell detection: prefer `FOPS_CLOCK_CRON_SHELL` (default `/bin/sh`). If unavailable, split commands with `strings.Fields` for distroless targets.
- Required inputs: `FOPS_CLOCK_CRON_SCHEDULE`, `FOPS_CLOCK_COMMAND`. Optional: `FOPS_CLOCK_CRON_SHELL`. CLI flags `--schedule`, `--command`, `--shell` must override env vars.

Code conventions
----------------
- Only Go ≥ 1.21 + `github.com/robfig/cron/v3`. No extra deps.
- Keep logic in `fops-clock/src/main.go`, tests in `main_test.go`. Use standard library logging.
- Add unit tests for every functional change (config loading, cron parsing, command execution, signal-friendly pieces when possible).
- Comments only where necessary to explain non-trivial blocks; keep files ASCII.

Tests & QA
----------
- Primary command: `cd fops-clock/src && GOCACHE=$(mktemp -d) go test ./...`.
- Every PR must pass `.github/workflows/fops-clock.yml` (Go tests, Hadolint, Trivy, Docker build/push).
- Reproduce the build with `docker build -t ghcr.io/e-frogg/froggops-docker-images-fops-clock:dev -f fops-clock/Dockerfile fops-clock`.

Docker & Release
----------------
- Multi-stage Dockerfile required: builder `golang:${GO_VERSION}-alpine`, final `scratch`.
- Build flags: `-trimpath -ldflags "-s -w -extldflags '-static'"`.
- Maintain OCI labels (`org.opencontainers.image.*`) referencing `github.com/e-frogg/froggops-docker-images`. Document any workflow changes in `README.md`.

Collaboration process
---------------------
- Any new environment variable or flag must be documented in `README.md` and `AGENTS.md`.
- Update CI if new steps (linting, multi-arch, release automation) become mandatory.
- Ensure `COPY --from` examples stay minimal and accurate.

Pre-merge checklist
-------------------
1. Go tests green locally and in CI.
2. `docker build` succeeds (provide `buildx` instructions if multi-arch is required).
3. Documentation updated (README + AGENTS).
4. Behavioral changes summarized in the PR description (FR or EN) to aid cross-team maintenance.
