# ──────────────────────────────────────────────
# HarborX — server + agent (multi-stage, non-root)
#
# Build targets:
#   docker build -t harborx          # default: server
#   docker build -t harborx-agent -f Dockerfile -t harborx-agent --target=agent
#
# GHCR CI pushes both images under ghcr.io/youko-nobody/harborx
#   and ghcr.io/youko-nobody/harborx-agent
# ──────────────────────────────────────────────

ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-alpine AS backend-build

WORKDIR /app
ENV CGO_ENABLED=0 GOMAXPROCS=1
RUN apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN mkdir -p /out && \
    go build -trimpath -ldflags="-s -w" -p=1 -o /out/harborx ./cmd/server && \
    go build -trimpath -ldflags="-s -w" -p=1 -o /out/harborx-agent ./cmd/agent

FROM node:22-alpine AS frontend-build

WORKDIR /web
COPY web/package.json web/package-lock.json web/tsconfig.json web/tsconfig.node.json web/vite.config.ts web/index.html ./
COPY web/src ./src
RUN npm ci && npm run build

# ──────────────────────────────────────────────
# Agent-only runtime (small, no frontend, no sqlite, no healthcheck)
# ──────────────────────────────────────────────
FROM alpine:3.21 AS agent

RUN apk add --no-cache ca-certificates tzdata wget && \
    addgroup -g 1000 harborx && \
    adduser -u 1000 -G harborx -s /bin/sh -D harborx

WORKDIR /app
COPY --from=backend-build /out/harborx-agent /app/harborx-agent
RUN chown -R harborx:harborx /app

USER harborx
ENTRYPOINT ["/app/harborx-agent"]

# ──────────────────────────────────────────────
# Server runtime (default target)
# ──────────────────────────────────────────────
FROM alpine:3.21 AS server

RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -g 1000 harborx && \
    adduser -u 1000 -G harborx -s /bin/sh -D harborx

RUN mkdir -p /app/data /app/templates && chown -R harborx:harborx /app

USER harborx
WORKDIR /app

COPY --from=backend-build /out/harborx /app/harborx
COPY --from=backend-build /out/harborx-agent /app/harborx-agent
COPY --from=frontend-build /web/dist /app/web-dist
COPY --chown=harborx:harborx templates /app/templates
COPY --chown=harborx:harborx internal/storage/schema.sql /app/schema.sql
COPY --chown=harborx:harborx internal/storage/seeds.sql /app/seeds.sql

EXPOSE 18080

ENV HARBORX_DATA_DIR=/app/data
ENV HARBORX_DB_PATH=/app/data/harborx.sqlite
ENV HARBORX_WEB_DIST_DIR=/app/web-dist

HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 \
  CMD wget -qO- http://127.0.0.1:18080/api/v1/health >/dev/null || exit 1

ENTRYPOINT ["/app/harborx"]
CMD ["--port", "18080"]