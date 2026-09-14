# syntax=docker/dockerfile:1
FROM node:22-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.27.1-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist ./web/dist
RUN go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/tally .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -g 10001 tally && adduser -D -H -u 10001 -G tally tally && mkdir -p /config && chown tally:tally /config
COPY --from=backend /out/tally /usr/local/bin/tally
USER tally:tally
ENV APP_DATA_DIR=/config APP_ADDR=:8080
VOLUME /config
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=15s --retries=3 CMD ["tally", "healthcheck"]
ENTRYPOINT ["tally"]
