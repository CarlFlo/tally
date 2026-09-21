# syntax=docker/dockerfile:1
FROM node:24-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN --mount=type=cache,target=/root/.npm npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27.1-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/tally .

FROM alpine:3.24
RUN apk upgrade --no-cache && apk add --no-cache ca-certificates su-exec
RUN mkdir -p /config
RUN mkdir -p /usr/local/libexec
COPY --from=backend /out/tally /usr/local/libexec/tally
COPY docker-tally.sh /usr/local/bin/tally
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod 755 /usr/local/bin/tally /usr/local/libexec/tally /usr/local/bin/docker-entrypoint.sh
ENV APP_DATA_DIR=/config APP_ADDR=:8080 PUID=10001 PGID=10001
VOLUME /config
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD ["tally", "healthcheck"]
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh"]
CMD ["tally"]
