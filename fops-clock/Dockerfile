ARG GO_VERSION=1.22

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ .

ENV CGO_ENABLED=0
RUN GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build -trimpath -ldflags="-s -w -extldflags '-static'" \
    -o /fops-clock .

FROM scratch
LABEL org.opencontainers.image.source="https://github.com/e-frogg/froggops-docker-images" \
      org.opencontainers.image.title="fops-clock" \
      org.opencontainers.image.description="Portable cron-like runner binary for hyperconverged infrastructures" \
      org.opencontainers.image.licenses="MIT"

COPY --from=builder /fops-clock /fops-clock
ENTRYPOINT ["/fops-clock"]
