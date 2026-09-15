# Tally

Tally is a modern, self-hosted TV show tracker for keeping up with shows, upcoming episodes, watch progress, and manual torrent searches.

## Features

- Calendar for past and upcoming episodes
- Personal TV show library and watch progress
- Multiple profiles
- TV show search and metadata from TVmaze
- Manual torrent search through Torznab providers
- Send selected torrents to qBittorrent
- Background jobs, statistics, notifications, and logs
- Scheduled and manual backups
- Local or OIDC authentication
- Responsive web interface with light and dark themes

## Docker Compose

```yaml
services:
  tally:
    build: .
    image: tally:local
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      APP_DATA_DIR: /config
      APP_ADDR: :8080
      TZ: Europe/Stockholm
    volumes:
      - tally-config:/config

volumes:
  tally-config:
```

Start Tally:

```sh
docker compose up -d --build
```

Then open **http://localhost:8080**.

## Commands

Tally runs the web server by default:

```sh
tally
tally serve
```

Available commands:

```text
tally serve
tally healthcheck
tally backup
tally verify-backup <archive>
tally restore <archive>
tally link-identity <profile-id-or-name> <issuer> <subject>
tally reset-password <profile-id-or-name>
tally delete-backup <filename>
```

When using Docker Compose:

```sh
docker compose run --rm tally backup
docker compose run --rm tally verify-backup /config/backups/example.zip
docker compose run --rm tally restore /config/backups/example.zip
docker compose run --rm tally link-identity "My profile" https://example.com subject
docker compose run --rm tally reset-password "My profile"
docker compose run --rm tally delete-backup example.zip
```

For `link-identity` and `reset-password`, the profile argument may be the opaque profile ID or an exact, unique display name. Ambiguous display names are rejected rather than guessed.

Commands that access the Tally data volume require exclusive access. Stop the running container first:

```sh
docker compose stop tally
# run command
docker compose up -d
```

`healthcheck` is intended to run while Tally is running.

## Build from source

Requirements:

- Go 1.27.1+
- Node.js 22+

Build the frontend:

```sh
cd frontend
npm ci
npm run build
cd ..
```

Build Tally:

```sh
go test ./...
go build -o tally .
```

Run it:

```sh
./tally
```
