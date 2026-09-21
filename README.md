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
- Per-profile Password or No authentication access
- Per-profile localization, including English and Ukrainian, with custom locale support
- Responsive light/dark web interface

Torrent search and torrent downloading are separate features and can be enabled independently. Tally does not automatically choose or download torrents.

## Quick start

```yaml
services:
  tally:
    image: lappenhappen/tally:latest
    container_name: tally
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=Etc/UTC
    stop_grace_period: 30s
    security_opt:
      - no-new-privileges:true
    cap_drop:
      - ALL
    cap_add:
      - CHOWN
      - SETGID
      - SETUID
    volumes:
      - tally-config:/config

volumes:
  tally-config:
```

## Integrations

Tally works without torrent integrations. When wanted, configure them under **Settings**:

- **Torrent search:** Jackett base URL and API key
- **Torrent download:** qBittorrent connection details
- **Notifications:** webhook or Discord destinations
- **Scheduling:** background job schedules
- **Backups:** schedule and retention policy

## Recommended deployment

Tally is intended for a trusted self-hosted environment or local network.

- Prefer Docker Compose and keep `/config` on persistent storage.
- Do not port-forward Tally directly to the public internet.
- For remote access, prefer a VPN/private network or an HTTPS reverse proxy with authentication enabled.
- Use Password authentication for profiles when access extends beyond a fully trusted network.
- Configure automatic backups and occasionally verify that your backup storage is available.
- Keep Tally, Jackett, and qBittorrent on trusted network paths where possible.

A profile configured with No authentication can be entered by anyone who can reach Tally.

## Data and backups

Tally stores its SQLite database, settings, locales, and backups below `/config` in the container. Application-managed credentials stored in Tally are included in protected backups; transient caches and environment-provided secrets are not.

Backups can be created and restored from the web interface or CLI. See **Command-line tools** below for CLI usage.

## Command-line tools

Run maintenance commands inside the existing container:

```bash
docker exec -it tally tally reset-password
docker exec -it tally tally reset-password <profile-id-or-unique-name>
docker exec -it tally tally backup
docker exec -it tally tally restore /config/backups/<backup>.zip
docker exec -it tally tally verify-backup /config/backups/<backup>.zip
docker exec -it tally tally delete-backup <filename>.zip
```

Running `reset-password` without a profile lists the available profiles and IDs. Run `docker exec tally tally help` for the full command list and usage.

## Localization

English and Ukrainian are bundled. Language is selected per profile, and missing translation keys fall back to English.

Bundled locale files in `/config/locales` are managed by Tally and reconciled on startup. Custom translations should use their own unique locale filename; those files are validated and hot-reloaded without restarting Tally.

See [docs/LOCALIZATION.md](docs/LOCALIZATION.md) for the locale format and translation workflow.
