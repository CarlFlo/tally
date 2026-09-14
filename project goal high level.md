Keep in mind this is the first draft of the project high end goal and the goal has been updated based on new instructions.
This is the version v0.1. and new complimatary changes that conflic with this design doc has been added.

# 1. Product

Build a modern self-hosted web application for:

* Tracking TV shows
* Viewing past/upcoming episodes in a calendar
* Multiple personal profiles
* Manual watched/downloaded episode state
* TV-show and generic torrent search
* Manually copying or sending a selected torrent/magnet to one configured torrent client
* Background metadata synchronization
* Jobs/provider statistics
* Backups and safe upgrades
* Optional local authentication and later OIDC/SSO

Inspired by useful DuckieTV concepts, but implemented from scratch.

## Priorities

The application should be:

* Fast
* Attractive
* Simple to self-host
* Low-resource
* Secure by design
* Predictable
* Observable
* Resistant to request/API storms
* Easy to upgrade and extend

## Non-goals

Do not implement unless requirements change:

* Movies
* Automatic torrent search/download/selection
* Media-library scanning
* File moving/renaming/hardlinking
* Torrent lifecycle management
* Playback
* Automatic duplicate-media detection
* Multiple simultaneous torrent clients
* Runtime plugin loading
* Required Redis/PostgreSQL/microservices

---

# 2. Technology and deployment

## Backend

Use whatever backend you deem would be the best
Suggested:
* Go
* Echo v5
* SQLite
* sqlc
* goose
* `log/slog`
* `robfig/cron` or equivalent
* Go contexts/deadlines
* Go `embed`

Prefer simple standard Go patterns over unnecessary abstractions.

## Frontend

Use whatever backend you deem would be the best:
Suggested:
* React
* TypeScript
* Vite
* Tailwind
* shadcn/ui
* TanStack Query
* React Router
* FullCalendar or equivalent
* i18n-ready string structure

English is initially sufficient.

## Deployment

Normal production deployment:

```text
Browser
  ↓
React SPA
  ↓
Go process
  ├── SQLite
  ├── Scheduler
  ├── TVmaze
  ├── Torrent providers
  ├── Torrent client
  └── Webhooks
```

Target:

```text
1 container
1 Go process
1 SQLite database
1 persistent /config volume
```

React is built during image creation and embedded into the Go binary/application.

SQLite is the permanent intended database.

Default:

```text
/config/app.db
```

Enable WAL and foreign keys.

---

# 3. Configuration

Configuration precedence:

```text
built-in defaults
        ↓
environment variables
```

Users only need to define values they want to override or values that inherently require deployment-specific configuration.

Do not duplicate infrastructure configuration into editable SQLite settings.

Example defaults:

```env
APP_AUTH_MODE=disabled
APP_MAX_PROFILES=8

APP_LANGUAGE=en
APP_THEME_DEFAULT=system

LOCAL_PASSWORD_MIN_LENGTH=4
LOCAL_PASSWORD_MAX_LENGTH=128
LOCAL_PASSWORD_ALLOW_NUMERIC_ONLY=true
LOCAL_PASSWORD_RESET_COOLDOWN=60s

SESSION_IDLE_TIMEOUT=30d
SESSION_ABSOLUTE_TIMEOUT=180d

JOB_METADATA_CRON=*/15 * * * *
JOB_MAINTENANCE_CRON=30 3 * * *
JOB_MAX_CONCURRENCY=4
JOB_MAX_RUNTIME=5m
JOB_MAX_RETRIES=3
JOB_MAX_BATCH_SIZE=50

PROVIDER_MAX_CONCURRENCY=2

BACKUP_ENABLED=true
BACKUP_CRON=0 3 * * *
BACKUP_KEEP=10
BACKUP_PATH=/config/backups
```

Secrets remain server-side.

The UI may show safe effective configuration as read-only:

```text
Authentication       Local
Max profiles         8
Timezone             Europe/Stockholm
Downloader           qBittorrent
Backup retention     10
OIDC secret          Configured
```

Never show actual secret values.

---

# 4. Core data model

Use application-owned internal IDs.

External provider IDs must not become the application's general primary keys.

Core tables/concepts:

```text
profiles
profile_preferences
profile_identities
local_credentials
sessions

shows
seasons
episodes
external_ids

profile_shows
profile_episode_state

jobs
job_runs

provider_state
provider_requests
provider_request_aggregates

alerts
backup_records

torrent_search_history
torrent_send_history
```

Disposable provider/image/search caches should be logically separate from durable state.

## Shared vs personal data

Shared deployment-wide:

* Show metadata
* Seasons
* Episodes
* External IDs
* Search-result cache
* Provider response cache
* Posters/images

Profile-specific:

* Followed shows
* Watched state
* Downloaded state
* Preferences
* Torrent-search selections/preferences

Example:

```text
TVmaze show 123
    ↓
one local show
    ├── user0 follows
    ├── user1 follows
    └── user4 follows
```

Never fetch/store separate TV metadata simply because multiple profiles follow the same show.

---

# 5. Profiles

Profiles use immutable IDs:

```text
user0
user1
user2
...
```

Rules:

* `user0` always exists
* `user0` cannot be deleted
* Other profiles may be deleted
* IDs are never reused
* IDs are independent of display names and authentication

Default maximum:

```env
APP_MAX_PROFILES=8
```

Profiles contain:

* Display name
* Avatar
* Personal preferences
* Followed shows
* Episode state

## Avatars

Support built-in avatars and uploads.

Allow:

* JPEG
* PNG
* WebP

Reject arbitrary SVG.

For uploads:

* Validate actual image type
* Limit file size/dimensions
* Strip metadata
* Resize/re-encode server-side
* Generate safe filenames

Example:

```text
/config/avatars/user2.webp
```

---

# 6. Profile selection and preferences

## Disabled-auth profile selection

If only one profile exists:

```text
enter it automatically
```

If multiple profiles exist:

* Use remembered profile if valid
* Otherwise show a Netflix-style picker

Remember the selected profile with a long-lived browser cookie, approximately one year.

The server must verify that the referenced profile still exists.

This cookie is convenience only, not authentication.

Keep profile switching easy from the current profile/avatar menu.

## Preferences

Store per-profile preferences in SQLite, including:

```text
theme
timezone
date_format
time_format
calendar_view
week_start
torrent search preferences
```

Support:

```text
system
light
dark
```

Default theme:

```text
system
```

Determine initial timezone from browser when possible, otherwise server `TZ`.

Use IANA timezone identifiers.

---

# 7. Authentication

Support:

```env
APP_AUTH_MODE=disabled|local|oidc
```

Authentication must map onto profiles rather than being embedded into domain data.

This allows disabled, local and OIDC modes to share the same profile/show/calendar model.

## Disabled

* No login
* Profiles are convenience boundaries only
* Anyone with app access can choose any profile

Suitable for trusted LAN/Tailscale/reverse-proxy environments.

If the app appears publicly reachable while auth is disabled, show a prominent warning.

Public detection is heuristic; use signals such as:

* `APP_PUBLIC_URL`
* Trusted proxy information
* Observed addresses

## Local

One local account maps to one profile.

A profile-selection screen may still be used:

```text
Who's watching?

Carl
Lisa
```

Selecting a profile opens its password/PIN prompt.

Profile enumeration is acceptable.

### Passwords/PINs

Store with Argon2id and unique random salts.

Passwords are opaque strings; do not trim, lowercase or otherwise modify them.

Default password policy intentionally allows short PINs:

```env
LOCAL_PASSWORD_MIN_LENGTH=4
LOCAL_PASSWORD_MAX_LENGTH=128
LOCAL_PASSWORD_ALLOW_NUMERIC_ONLY=true
```

Optional stricter rules may be configured through ENV.

A value such as:

```text
1234
```

is valid by default.

Because short PINs are low entropy, protect normal login from very rapid automated guessing using modest progressive server-side delay or equivalent.

### Bootstrap

On first local-auth startup with no credentials:

* Map local authentication to `user0`
* Generate a secure temporary bootstrap password
* Print it once to server/container logs
* Require immediate password replacement

### Sessions

Use long-lived, revocable sessions.

Suggested defaults:

```env
SESSION_IDLE_TIMEOUT=30d
SESSION_ABSOLUTE_TIMEOUT=180d
```

Use:

* Random session IDs
* HttpOnly cookies
* SameSite
* Secure under HTTPS
* Rotation/revocation

Users can view/revoke their own sessions and change their password.

### Forgot password

Clicking Forgot Password:

* Generates a secure temporary password
* Writes username + temporary password + expiry to server logs
* Uses a server-enforced cooldown, default 60 seconds
* Does not invalidate the current password

Temporary recovery credentials:

* Expire after 30 minutes
* Expire on application restart
* Are kept only in application memory
* One valid temporary credential per account

When used successfully, the user enters a restricted password-change flow.

Only:

```text
Set new password
Sign out
```

are allowed.

The old permanent password is replaced only after the new password is successfully committed.

## OIDC / SSO

Low priority and should be implemented last.

One OIDC identity maps to one profile.

Use:

```text
issuer + subject
```

as identity.

Do not use email/display name as immutable identity.

Support:

* Discovery
* state
* nonce
* PKCE where appropriate
* Strict redirect validation
* Auto-provisioning up to `APP_MAX_PROFILES`

Typical ENV:

```env
OIDC_ISSUER_URL=
OIDC_CLIENT_ID=
OIDC_CLIENT_SECRET=
OIDC_REDIRECT_URL=
OIDC_SCOPES=openid,profile,email
OIDC_AUTO_CREATE_USERS=true
```

---

# 8. UI and navigation

Primary navigation:

```text
Calendar
Shows
Torrent Search

────────────

Jobs
Statistics

────────────

Settings
```

Only show pages that actually exist.

Stable routes:

```text
/calendar
/shows
/shows/:id
/search
/jobs
/statistics
/settings
```

Browser back/forward/refresh must work normally.

## Design principles

Prefer:

* Strong visual hierarchy
* Clean typography
* Consistent spacing
* Good dark mode
* Minimal clicks
* Persistent useful choices
* Immediate feedback
* Stable layouts
* Responsive desktop/tablet/mobile design

Avoid:

* Excessive animation
* Excessive gradients/shadows
* Full-page loaders for minor operations
* Unnecessary modal dialogs
* Layout shift

Respect `prefers-reduced-motion`.

Use accessible semantic HTML, keyboard navigation, visible focus states, proper dialog behavior, labels for icon-only controls, and adequate touch targets.

---

# 9. Calendar

The calendar is the main/default page.

Default:

```text
current month
```

Desktop/tablet month view should be a real calendar:

```text
7 columns
one row per week
```

Show:

* Clearly highlighted today
* Subdued past dates
* Muted adjacent-month dates
* Compact episode entries

Example:

```text
Severance
S03E04
```

Do not fill cells with large posters.

If a day has many entries:

```text
first few
+ N more
```

Controls:

```text
Previous
Today
Next
```

Views:

```text
Month
Week
Agenda
```

Remember the selected view per profile.

On narrow mobile screens, agenda/list should be particularly usable.

## Episode interaction

Selecting an episode opens a lightweight drawer/popover/modal.

Actions:

```text
Mark watched
Mark downloaded
Search torrents
View show
```

Use optimistic state updates and revert on persistence failure.

Useful bulk actions:

* Mark season watched/unwatched
* Mark all aired watched
* Mark season downloaded

Calendar rendering always uses local SQLite data.

Navigating between months must not trigger TVmaze requests.

---

# 10. TV metadata

Use TVmaze initially.

Keep provider-specific code behind a small interface:

```go
type TVProvider interface {
    SearchShows(ctx context.Context, query string) ([]ShowSearchResult, error)
    GetShow(ctx context.Context, externalID string) (*Show, error)
    GetEpisodes(ctx context.Context, externalID string) ([]Episode, error)
}
```

Do not create a runtime plugin system.

## Search

Use TVmaze's ranked show search rather than assuming one name has one result.

Search should:

* Debounce around 300–500 ms
* Require useful query length
* Cancel obsolete browser requests
* Cache equivalent queries
* Support keyboard use

Results show:

* Poster
* Name
* Premiere year
* Status
* Network/web channel where available
* Short summary
* Whether active profile already follows it

Normalize equivalent cache queries such as:

```text
Severance
 severance
SEVERANCE
```

where sensible.

## Adding a show

Store:

* Show metadata
* Seasons
* All known past episodes
* All known upcoming episodes
* Specials
* External IDs
* Image references

Removing a show removes the profile relationship, not shared metadata used by others.

---

# 11. Provider caching and synchronization

External APIs are synchronization sources, not live frontend backends.

Normal UI path:

```text
Browser
  ↓
Go
  ↓
SQLite/cache
```

not:

```text
Browser
  ↓
Go
  ↓
TVmaze on every page load
```

## Shared caching

Cache globally:

* Shows
* Seasons
* Episodes
* Search responses
* Provider responses
* Posters/images

Use conditional requests where supported:

* ETag
* Last-Modified
* If-None-Match
* If-Modified-Since
* 304

Server-side image cache may live under:

```text
/config/cache/images/
```

The image cache is disposable and excluded from backups.

Only fetch images from expected provider hosts and enforce content/type/size limits.

---

# 12. Provider request coordinator

All outbound provider calls must pass through one central control layer handling:

* Cache
* Single-flight/request coalescing
* Provider concurrency
* Rate limiting
* Timeout
* Response-size limits
* Retry
* `Retry-After`
* Circuit breaker
* Logging
* Metrics/statistics

Nothing should bypass it.

## Retry policy

Typical behavior:

```text
400      no retry
401/403  no repeated retry
404      normally no retry
408      limited retry
429      honor Retry-After
5xx      limited retry
```

Use bounded exponential backoff with jitter.

Example:

```text
5s
30s
2m
stop
```

Manual refresh may bypass freshness TTL, but never:

* Rate limits
* `Retry-After`
* Circuit breaker
* Concurrency limits
* Timeouts
* Response-size limits

## Circuit breaker

A reasonable default:

```text
5 relevant failures within 10 minutes
```

States:

```text
healthy
degraded
open
half-open
```

Suggested recovery sequence:

```text
30m
1h
2h
4h
6h max
```

After cooldown, allow one controlled probe.

Success restores normal operation.

---

# 13. Metadata scheduling and jobs

Do not refresh every show on every scheduler tick.

Scheduler wake:

```env
JOB_METADATA_CRON=*/15 * * * *
```

means:

```text
find due entities
process only due entities
```

Track per entity where useful:

```text
last_checked_at
next_check_at
last_changed_at
provider_updated_at
```

Suggested refresh strategy:

* Episode soon: roughly every 6–12 hours
* Known episode weeks away: less often
* Active/no known next episode: about daily
* Between seasons: every few days
* Ended: about monthly

Use jitter.

## Job records

Track:

```text
job type/key
trigger
scheduled/start/end
duration
status
attempt
error
candidate/processed counts
API calls
cache hits
changes
```

Example logical keys:

```text
metadata:tvmaze:show:123
backup:auto
maintenance:image-cache
```

Do not run the same logical job concurrently.

Every job must have:

* Context
* Deadline
* Retry bound
* Batch bound
* Pagination bound
* Cancellation

On startup, detect stale `running` jobs and mark/reschedule safely.

---

# 14. Jobs and statistics UI

## Jobs

Show:

* Schedule
* Last/next run
* Duration
* Candidates
* Refreshed/skipped
* API calls
* Cache hits
* Changes
* Result

Allow manual triggering where appropriate.

Schedules remain ENV-driven/read-only.

## Statistics

Show useful operational data:

* Requests/provider
* Success/failure
* Retries
* 429s
* Backoff
* Circuit events
* Cache hits
* 304s
* Calls avoided
* Average latency
* Job history
* Next scans

Track reasons calls were avoided:

```text
fresh cache
not due
duplicate/coalesced
conditional cache hit
provider backoff
```

Every external call should be explainable:

```text
why was it made?
what triggered it?
for which entity?
did it hit cache?
did it retry?
how long did it take?
```

Suggested triggers:

```text
user_search
manual_refresh
scheduled_refresh
retry
recovery_probe
torrent_search
connection_test
```

Retention suggestion:

```text
raw detail      30 days
aggregates      1 year
```

configurable.

---

# 15. Alerts and notifications

Important failures should:

1. Be logged
2. Appear in-app
3. Optionally send webhook

Examples:

* Provider circuit opened/recovered
* Backup failed
* Torrent client unavailable
* Migration failed
* Auth disabled on apparently public deployment

Deduplicate repeated alerts while state is unchanged.

Initially support a generic webhook only.

Keep notification code behind a small provider-neutral interface.

---

# 16. Torrent search

Torrent search is always manually initiated.

Support:

1. Episode-based search with pre-filled query
2. Generic free-text torrent search

Example episode query:

```text
Severance S03E04
```

It must remain fully editable.

## Search providers

Use:

```go
type SearchProvider interface {
    ID() string
    Name() string
    Search(ctx context.Context, query SearchQuery) ([]SearchResult, error)
}
```

Prefer stable APIs:

* Torznab
* JSON/XML APIs
* RSS/API feeds

Avoid brittle HTML scraping where possible.

Torznab should be the first generalized implementation because it enables Prowlarr/Jackett-style integrations.

Provider configuration is deployment-level.

Profiles may remember which configured providers they prefer to search.

## Results

Show:

* Name
* Size
* Seeders
* Provider
* Optional leechers/age/category

Default sorting:

```text
seeders descending
```

Actions:

```text
Copy magnet
Send to torrent client
```

Filters:

* Minimum seeders
* Minimum/maximum size
* Include/exclude title keywords

Convenience quality filters:

```text
720p
1080p
2160p
WEB-DL
WEBRip
BluRay
x264
x265
HEVC
HDR
DV
```

These are keyword filters, not authoritative media analysis.

Treat all torrent/provider strings as untrusted.

---

# 17. Torrent client

One deployment has one active configured torrent client shared by all profiles.

Configuration example:

```env
DOWNLOAD_CLIENT=qbittorrent
DOWNLOAD_CLIENT_URL=http://qbittorrent:8080
DOWNLOAD_CLIENT_USERNAME=
DOWNLOAD_CLIENT_PASSWORD=
```

Credentials stay server-side.

Use a minimal interface:

```go
type DownloadClient interface {
    Name() string
    TestConnection(ctx context.Context) error
    AddMagnet(ctx context.Context, magnet string) error
}
```

qBittorrent should be implemented first.

Later adapters may include:

* Transmission
* Deluge
* rTorrent-compatible clients

Do not add lifecycle methods the app does not use.

Protect submission against accidental double-click/network replay with short-lived idempotency.

Sending a torrent does not mark an episode as downloaded.

---

# 18. Backups

Backups are deployment-wide.

Include durable state required to recreate the application:

* SQLite domain state
* Profiles/preferences
* Followed shows
* Watched/downloaded state
* Local credential hashes
* Identity mappings
* Avatars
* Durable job/stat history
* Manifest

Exclude disposable/recoverable data:

* Provider response cache
* Search cache
* Poster/image cache
* Temp files
* Recovery passwords
* ENV secrets

Use SQLite's backup/snapshot mechanism rather than copying the live DB file under WAL.

Validate completed backups.

## Automatic backups

Example:

```env
BACKUP_ENABLED=true
BACKUP_CRON=0 3 * * *
BACKUP_KEEP=10
BACKUP_PATH=/config/backups
```

Retain the newest successful automatic backups.

## Manual backups

Manual backups are not subject to automatic rolling retention.

They remain until deliberately deleted.

Backup archives contain all profiles/auth state and should be treated as deployment-sensitive.

Restore/export should be considered an operator-level capability rather than normal per-profile administration.

---

# 19. Upgrades and migrations

Startup flow:

1. Load/validate configuration
2. Open database
3. Compare schema version
4. If migration required, create and verify pre-upgrade backup
5. Run migrations
6. Verify schema
7. Start application

If backup fails:

```text
do not migrate
```

If migration fails:

* Do not become ready
* Log clearly
* Exit non-zero
* Preserve backup
* Avoid partially running

An optional marker may record repeated migration failure:

```text
/config/migration-failed.json
```

Downgrading to an incompatible schema should fail safely.

---

# 20. Security

## Sessions and authorization

In authenticated modes, derive active profile from the server-side authenticated session.

Do not trust a browser-supplied profile ID for authorization.

One authenticated profile cannot modify another profile's personal state.

Disabled mode intentionally provides no such security boundary.

## Web security

Use:

* HttpOnly cookies
* SameSite
* Secure cookies under HTTPS
* CSRF protection for cookie-authenticated mutations
* Origin validation
* CSP
* `frame-ancestors`
* `X-Content-Type-Options: nosniff`
* Appropriate referrer policy
* Request-body limits
* HTTP read/write/idle/header timeouts

No mutating GET endpoints.

## Untrusted content

Treat as untrusted:

* TV metadata
* HTML summaries
* Torrent titles
* Search queries
* Provider error text
* Image data

Escape output.

Do not inject raw provider HTML.

Sanitize or convert summaries to safe text.

## SSRF

Do not provide a generic arbitrary URL-fetch endpoint.

Configured outbound URLs must use safe parsing, schemes, timeouts, size limits and cautious redirects.

Restrict cached image fetching to expected provider hosts.

## Logging

Use structured logs.

Useful fields include:

```text
provider
job_id
profile_id
show_id
episode_id
status_code
retry_count
duration_ms
```

Never log:

* Permanent passwords
* Password hashes
* API keys
* OIDC tokens
* Authorization headers
* Downloader credentials
* Webhook secrets

Temporary bootstrap/recovery passwords are the intentional exception and should be printed once in isolated recovery messages.

---

# 21. Health and shutdown

Expose:

```text
/healthz
/readyz
```

`/healthz` checks process liveness.

`/readyz` requires:

* Valid configuration
* SQLite available
* Migrations completed
* Application able to serve normal local requests

External providers being unavailable must not make readiness fail.

Prefer a built-in healthcheck command rather than installing curl solely for Docker health checks.

## Graceful shutdown

On SIGTERM:

* Stop accepting new background work
* Stop starting provider calls
* Allow/cancel active work within deadline
* Persist state
* Close SQLite cleanly
* Exit

Transient external-service errors must not crash the process.

---

# 22. Frontend request behavior

Normal pages should be local-data driven and responsive.

Use TanStack Query deliberately.

Avoid:

* Search requests for every keystroke
* React effect loops
* Blind refetch-on-focus that triggers provider work
* Excessive retries
* Unnecessary sequential API calls

Prefer:

* Cached local content
* Skeleton only for initial load
* Existing content retained during refresh
* Inline action loading
* Adjacent calendar prefetch

---

# 23. User feedback

Use concise success messages such as:

```text
Show added
Marked watched
Magnet copied
Sent to qBittorrent
Backup created
```

Known errors should be specific and actionable.

Avoid generic "Something went wrong" if the application knows the cause.

Use confirmation dialogs only for meaningful destructive operations, such as:

* Delete profile
* Remove show
* Restore backup
* Delete manual backup

Prefer Undo for easy reversible actions.

---

# 24. Development tracking

Maintain:

```text
TODO.md
```

at repository root.

It tracks implementation state, not product requirements.

Use statuses such as:

```text
TODO
IN PROGRESS
DONE
BLOCKED
DEFERRED
```

Keep near the top:

* Current feature
* Current milestone/state
* What is being implemented
* What remains for that feature
* Blockers
* Next logical work

The complete implementation roadmap should exist in this file.

Important tasks should include concise acceptance criteria where useful.

Record architectural context when it prevents future refactors, for example:

```text
Show metadata is global.
Do not add profile ownership to shared show rows.
Profiles reference shows through profile_shows.
```

or:

```text
Authentication maps to profiles.
Do not embed authentication fields into show/calendar ownership.
```

`TODO.md` should be updated whenever implementation state changes.

Do not mark a feature done because a stub or partial happy path exists.

The code/tests remain authoritative if `TODO.md` is stale.

A short `AGENTS.md` should instruct coding agents to:

* Read this specification
* Read `TODO.md`
* Continue existing work before starting unrelated features
* Preserve documented architectural boundaries
* Update `TODO.md` before ending work

---

# 25. Important architectural boundaries

These are the decisions most important to preserve:

1. SQLite is permanent, not temporary.
2. Shared metadata and profile state remain separate.
3. Profiles are independent from authentication.
4. External IDs are not the application's universal primary keys.
5. Provider-specific code stays behind small interfaces.
6. All remote requests eventually pass through centralized request control.
7. Frontend rendering normally uses local data.
8. Infrastructure configuration remains ENV-driven.
9. Jobs and provider loops are bounded and observable.
10. Manual provider actions cannot bypass safety controls.
11. Torrent search/download remains user-initiated.
12. One deployment uses one shared torrent client.
13. Backups exclude disposable cache and ENV secrets.
14. Upgrades require safe migrations and pre-upgrade backup.
15. OIDC is deliberately low priority and should be implemented after the core application is stable.

---

# 26. Suggested repository structure

```text
app/
├── cmd/server/
├── internal/
│   ├── api/
│   ├── auth/
│   │   ├── disabled/
│   │   ├── local/
│   │   └── oidc/
│   ├── backup/
│   ├── cache/
│   ├── calendar/
│   ├── config/
│   ├── database/
│   ├── downloadclient/
│   ├── jobs/
│   ├── metadata/tvmaze/
│   ├── notifications/
│   ├── providers/
│   ├── scheduler/
│   ├── security/
│   ├── statistics/
│   ├── torrentsearch/
│   └── service/
├── migrations/
├── queries/
├── frontend/
├── web/
├── TODO.md
├── AGENTS.md
├── Dockerfile
├── docker-compose.yml
└── go.mod
```

Do not create packages/modules before they are useful merely to match the final tree.

---

# 27. Testing priorities

Automate important boundaries and failure paths, including:

* `user0` cannot be deleted
* Profile IDs are never reused
* Profile maximum
* Remembered deleted profile
* Shared show metadata deduplication
* Cross-profile authorization
* Password/PIN hashing and policy
* Recovery expiry/restart behavior
* Forced password change after recovery
* OIDC mapping
* CSRF
* Avatar validation
* Provider request deduplication
* `Retry-After`
* Circuit transitions/recovery
* Job deduplication/interruption
* Migration failure
* Backup consistency/restore validation
* Secret redaction
* Torrent submission idempotency

---

# 28. Target experience

A typical user should be able to:

```text
docker compose up -d
```

open the app and immediately land on the current month's calendar when only one profile exists.

They search for a show, add it, and all known past/upcoming episodes and specials are stored locally.

Another profile following the same show reuses the same metadata.

On return visits, the browser remembers the last profile.

Selecting an episode provides:

```text
Mark watched
Mark downloaded
Search torrents
View show
```

Torrent search pre-fills:

```text
Show Name S03E04
```

but remains freely editable.

The user chooses configured providers, filters/sorts results, then either copies the magnet or sends the exact selected result to the configured torrent client.

Nothing else is automatically downloaded or managed.

Meanwhile the application quietly keeps metadata current using globally shared caching, adaptive scheduling, rate limiting, request deduplication, bounded retries and circuit breaking.

Jobs and Statistics explain what the application is doing and why.

The result should feel like a polished self-hosted product rather than a collection of scripts.
