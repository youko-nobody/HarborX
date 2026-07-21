FROM golang:1.25-alpine AS backend-build

WORKDIR /app
ENV GOMAXPROCS=1
COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -p=1 -o /out/harborx ./cmd/server
RUN go build -p=1 -o /out/harborx-agent ./cmd/agent

FROM node:22-alpine AS frontend-build

WORKDIR /web
COPY web/package.json web/package-lock.json web/tsconfig.json web/tsconfig.node.json web/vite.config.ts web/index.html ./
COPY web/src ./src

RUN npm ci && npm run build

FROM alpine:3.21

WORKDIR /app
COPY --from=backend-build /out/harborx /app/harborx
COPY --from=backend-build /out/harborx-agent /app/harborx-agent
COPY --from=frontend-build /web/dist /app/web-dist
COPY templates /app/templates
COPY internal/storage/schema.sql /app/schema.sql
COPY internal/storage/seeds.sql /app/seeds.sql

EXPOSE 18080

ENV HARBORX_DATA_DIR=/app/data
ENV HARBORX_DB_PATH=/app/data/harborx.sqlite
ENV HARBORX_WEB_DIST_DIR=/app/web-dist

CMD ["/app/harborx"]
