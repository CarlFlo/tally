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

Backups can be created and restored from the web interface or CLI.

Operator commands that Tally can apply safely at runtime are forwarded to the running server through a private Unix socket under `/config`. This means password resets and live restores do not require stopping the container:

```bash
docker exec -it tally tally reset-password <profile-id-or-name>
docker exec -it tally tally restore /config/backups/<backup>.zip
```

The password reset prompts for the temporary password without echoing it, revokes existing sessions, and requires the profile to replace it after signing in. A live restore uses the same validated restore path as the web interface and refreshes server/browser state afterward.

Commands that require exclusive offline access will still refuse to run while the main server owns the deployment lock. Run `tally help` to see the available commands.

## Command-line tools

Tally includes operator commands for common maintenance and recovery tasks. When using Docker, run them inside the existing container:

```bash
docker exec -it tally tally <command>
```

Use `tally help` to display the available commands.

### Reset a profile password

```bash
docker exec -it tally tally reset-password <profile-id-or-name>
```

Tally prompts for the temporary password without echoing it to the terminal. The command can be run while the server is online. Existing sessions for the profile are revoked, and the user must replace the temporary password after signing in.

### Restore a backup

```bash
docker exec -it tally tally restore /config/backups/<backup>.zip
```

Restore can be performed while Tally is running. It uses the same validated live-restore path as the web interface and refreshes application state after the restore completes.

### Create a backup

```bash
docker exec -it tally tally backup
```

Creates a manual backup and prints the resulting archive path.

### Verify a backup

```bash
docker exec -it tally tally verify-backup /config/backups/<backup>.zip
```

Validates and extracts the archive in a temporary staging directory without changing the live application state.

### Delete a backup

```bash
docker exec -it tally tally delete-backup <filename>.zip
```

Deletes a backup from Tally's backup directory. Supply only the archive filename, not a path.

### Check server readiness

```bash
docker exec tally tally healthcheck
```

The command exits successfully when the running server reports that it is ready. It normally produces no output on success.

### Show command help

```bash
docker exec tally tally help
```

The equivalent `tally -h` and `tally --help` forms are also supported.

Commands that cannot safely share the live application state still use Tally's deployment lock and will refuse to run concurrently with the server.

## Localization

English and Ukrainian are bundled. Language is selected per profile, and missing translation keys fall back to English.

Bundled locale files in `/config/locales` are managed by Tally and reconciled on startup. Custom translations should use their own unique locale filename; those files are validated and hot-reloaded without restarting Tally.

See [docs/LOCALIZATION.md](docs/LOCALIZATION.md) for the locale format and translation workflow.
