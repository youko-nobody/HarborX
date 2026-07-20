FROM golang:1.22-alpine AS backend-build

WORKDIR /app
COPY go.mod ./
COPY cmd ./cmd
COPY internal ./internal

RUN go build -o /out/harborx ./cmd/server

FROM node:22-alpine AS frontend-build

WORKDIR /web
COPY web/package.json web/tsconfig.json web/tsconfig.node.json web/vite.config.ts web/index.html ./
COPY web/src ./src

RUN npm install && npm run build

FROM alpine:3.21

WORKDIR /app
COPY --from=backend-build /out/harborx /app/harborx
COPY --from=frontend-build /web/dist /app/web-dist
COPY templates /app/templates
COPY internal/storage/schema.sql /app/schema.sql
COPY internal/storage/seeds.sql /app/seeds.sql

EXPOSE 18080

CMD ["/app/harborx"]
