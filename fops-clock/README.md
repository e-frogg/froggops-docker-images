# fops-clock

`fops-clock` is a tiny static Go binary that turns any container into a cron-style worker. It runs as PID 1, sleeps until the next tick, executes the configured command, and handles `SIGINT`, `SIGTERM`, and `SIGCHLD`. The binary ships from a `scratch` image and can be injected into existing images via `COPY --from`.

---

## Key Features

- **Ultra portable** – `CGO_ENABLED=0`, works on glibc, musl, and distroless images.
- **Shell aware** – uses `/bin/sh` (or `FOPS_CLOCK_CRON_SHELL`) when present, or executes commands directly otherwise.
- **PID 1 safe** – reaps zombies and propagates termination signals.
- **Flexible config** – supports both environment variables and CLI flags (`--schedule`, `--command`, `--shell`).

---

## Quick Start

### Inject into your own image

```dockerfile
FROM python:3.12-slim
COPY --from=ghcr.io/e-frogg/froggops-docker-images-fops-clock:latest /fops-clock /usr/local/bin/fops-clock
ENV FOPS_CLOCK_CRON_SCHEDULE="*/10 * * * *"
ENV FOPS_CLOCK_COMMAND="python /app/task.py"
ENTRYPOINT ["/usr/local/bin/fops-clock"]
```

### Run directly with Docker

```bash
docker run --rm \
  -e FOPS_CLOCK_CRON_SCHEDULE="*/1 * * * *" \
  -e FOPS_CLOCK_COMMAND="echo hello" \
  ghcr.io/e-frogg/froggops-docker-images-fops-clock:latest
```

### Provide CLI arguments instead of env vars

```bash
docker run --rm ghcr.io/e-frogg/froggops-docker-images-fops-clock:latest \
  --schedule "*/5 * * * *" \
  --command "python /app/task.py" \
  --shell /bin/sh
```

Flags always override the environment configuration.

---

## Configuration

| Variable                   | Description                                                                                 | Required |
|---------------------------|---------------------------------------------------------------------------------------------|----------|
| `FOPS_CLOCK_CRON_SCHEDULE`| Cron expression (`minute hour day month weekday`) parsed by `robfig/cron/v3`.               | ✅       |
| `FOPS_CLOCK_COMMAND`      | Command to execute. Pipes and redirects work when a shell is available.                    | ✅       |
| `FOPS_CLOCK_CRON_SHELL`   | Optional shell path (`/bin/sh` default). If missing/unavailable, the command is executed directly. | ❌       |

All logs go to `stdout`/`stderr`, so `docker logs` / `kubectl logs` just work.

---

## Build & Release

Build the artifact image locally:

```bash
docker build -t ghcr.io/e-frogg/froggops-docker-images-fops-clock:dev -f fops-clock/Dockerfile fops-clock
```

The resulting image contains only `/fops-clock` and is meant to be pushed to `ghcr.io/e-frogg/froggops-docker-images-fops-clock`.

---

## Local Development

Unit tests:

```bash
cd fops-clock/src
GOCACHE=$(mktemp -d) go test ./...
```

Run the binary directly:

```bash
FOPS_CLOCK_CRON_SCHEDULE="*/1 * * * *" FOPS_CLOCK_COMMAND="echo dev run" go run .
# or
go run . --schedule "*/1 * * * *" --command "echo via flag"
```

---

## CI/CD

- `.github/workflows/fops-clock.yml` runs Go tests, Hadolint, Trivy, then builds and pushes `ghcr.io/e-frogg/froggops-docker-images-fops-clock` (plus provenance attestation).
- `.github/workflows/build.yml` remains the umbrella workflow for the other images in this repository.

---

## Useful References

- `fops-clock/AGENTS.md` – contribution rules and checklist.
- `fops-clock/src/main.go` – cron loop, signal handling, CLI parsing.
- `fops-clock/Dockerfile` – multi-stage build producing the static binary.

Happy hacking!
