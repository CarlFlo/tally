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
RUN apk add --no-cache ca-certificates && addgroup -g 10001 tally && adduser -D -H -u 10001 -G tally tally && mkdir -p /config && chown tally:tally /config
COPY --from=backend /out/tally /usr/local/bin/tally
USER tally:tally
ENV APP_DATA_DIR=/config APP_ADDR=:8080
VOLUME /config
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD ["tally", "healthcheck"]
ENTRYPOINT ["tally"]
