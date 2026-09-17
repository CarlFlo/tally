# Tally

**Your little TV universe.**

Tally is a self-hosted TV show tracker for following series, seeing what is coming up, tracking episode progress, and optionally searching for and sending torrents to a download client.

## Features

- Calendar for upcoming and recently aired episodes
- Personal show library, favorites, and episode watch state
- Multiple profiles with role-based administration
- TV metadata and discovery powered by TVmaze
- Manual torrent search through Jackett/Torznab
- Optional torrent downloads through qBittorrent, with download monitoring
- Background schedules, activity logs, statistics, and configurable bell notifications
- Manual and scheduled backups with restore validation
- Local authentication, OIDC, or trusted-network mode
- Per-profile localization, including English and Ukrainian, with custom locale support
- Responsive light/dark web interface

Torrent search and torrent downloading are separate features and can be enabled independently. Tally does not automatically choose or download torrents.

## Quick start

Docker Compose is the recommended way to run Tally.

```sh
git clone https://github.com/CarlFlo/tally.git
cd tally
cp .env.example .env
docker compose up -d --build
```

Open **http://localhost:8080**.

The included Compose setup stores persistent data in the `tally-config` volume and listens on localhost by default.

For access from other devices on your LAN, change this in `.env`:

```env
APP_BIND=0.0.0.0
```

You should also set the deployment timezone, for example:

```env
TZ=Europe/Stockholm
```

Most application settings are managed from the Tally web interface rather than environment variables.

## Integrations

Tally works without torrent integrations. When wanted, configure them under **Settings**:

- **Torrent search:** Jackett base URL and API key
- **Torrent download:** qBittorrent connection details
- **Notifications:** webhook or Discord destinations
- **Scheduling:** background job schedules
- **Backups:** schedule and retention policy

TVmaze metadata does not require an API key.

## Recommended deployment

Tally is intended for a trusted self-hosted environment or local network.

- Prefer Docker Compose and keep `/config` on persistent storage.
- Do not port-forward Tally directly to the public internet.
- For remote access, prefer a VPN/private network or an HTTPS reverse proxy with authentication enabled.
- Use local authentication or OIDC when access is shared beyond a fully trusted network.
- Configure automatic backups and occasionally verify that your backup storage is available.
- Keep Tally, Jackett, and qBittorrent on trusted network paths where possible.

When authentication is disabled, anyone who can reach Tally can use the available profiles.

## Data and backups

Tally stores its SQLite database, settings, locales, and backups below `/config` in the container. Application-managed credentials stored in Tally are included in protected backups; transient caches and environment-provided secrets are not.

Backups can be created and restored from the web interface or CLI. Commands that directly modify Tally's data should normally be run while the main container is stopped.

## Localization

English and Ukrainian are bundled. Language is selected per profile, and missing translation keys fall back to English.

Bundled locale files in `/config/locales` are managed by Tally and reconciled on startup. Custom translations should use their own unique locale filename; those files are validated and hot-reloaded without restarting Tally.

See [docs/LOCALIZATION.md](docs/LOCALIZATION.md) for the locale format and translation workflow.

## Build from source

Requirements:

- Go 1.27.1+
- Node.js 24+

```sh
cd frontend
npm ci
npm run build
cd ..
go test ./...
go build -o tally .
./tally
```

For development details, see [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md). Architecture and validation notes are available in [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) and [docs/VALIDATION.md](docs/VALIDATION.md).
