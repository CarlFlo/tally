# Hardening and performance upgrade tracker

Branch: `audit/hardening-performance-2026-09-15`
Started: 2026-09-15
Scope: implement the fixes and upgrades from the 2026-09-15 code audit while preserving Tally's LAN/self-hosted deployment model and existing user-visible behavior unless a change is explicitly noted here.

## Status legend

- [ ] Planned
- [~] In progress
- [x] Completed
- [!] Blocked / issue found
- [-] Intentionally deferred

## Planned work

### Runtime and backend efficiency

- [~] Fix long-lived SSE handling so the server-wide write timeout cannot terminate healthy event streams.
- [ ] Replace the one-second scheduled-job SQLite polling loop with a next-deadline timer plus explicit wakeups when schedules change.
- [ ] Replace the one-second notification worker polling loop with event/deadline-driven wakeups and a conservative fallback timer.
- [ ] Remove duplicate image-body caching between provider `cache.db` and the filesystem image cache.
- [ ] Stream cached image files instead of loading the complete file into memory.
- [ ] Make SQLite WAL checkpoint policy explicit for the main and provider-cache databases and checkpoint cleanly during maintenance/shutdown where appropriate.
- [ ] Split shutdown phase budgets so job cancellation cannot consume the entire HTTP-drain budget.

### Frontend efficiency and navigation

- [ ] Remove the global `<Routes key={location.pathname}>` remount and preserve route/component lifecycle normally.
- [ ] Add route-level lazy loading for secondary/admin/settings pages while keeping core navigation responsive.
- [ ] Remove 500 ms manual-backup job polling and rely on live job/backups invalidation; retain a bounded fallback for connection loss.
- [ ] Reduce repeated backup archive ZIP inspection by caching archive metadata keyed by filesystem identity/mtime/size while preserving manual Refresh semantics.

### Dependencies and image/build

- [ ] Upgrade `modernc.org/sqlite` from v1.58.0 to the current v1.59.x release and update sums.
- [ ] Re-check Go and npm dependencies after the implementation; do not perform major framework migrations solely for version-number parity.
- [ ] Keep Node 24 LTS for the frontend builder and Alpine 3.24 for the runtime unless verification shows a better stable choice.
- [ ] Remove redundant runtime `tzdata` package if the embedded Go timezone database fully covers Tally's runtime needs.
- [ ] Separate production image build from test execution so normal image builds do not rerun the Go suite unnecessarily.
- [ ] Add BuildKit cache mounts for Go modules/build cache and npm package cache where safe.

### Security and operational hardening

- [ ] Keep the LAN-only threat model explicit in documentation and warn against direct public-internet exposure.
- [ ] Raise the default local password/PIN minimum from 4 to a safer value while preserving the environment override.
- [ ] Preserve current first-profile bootstrap behavior; do not add setup-token complexity for the trusted-LAN model.
- [ ] Add `govulncheck ./...` to CI.
- [ ] Add npm dependency audit for high/critical vulnerabilities to CI.
- [ ] Add a container-image vulnerability scan to CI with a practical high/critical threshold.
- [ ] Add/confirm automated dependency update configuration for Go, npm, Docker base images, and GitHub Actions.

### Maintainability

- [ ] Reassess the low-activity cron-description dependency; replace it only if a small, tested internal five-field describer can cover Tally's supported expressions without regressions.
- [ ] Update architecture/TODO/README documentation for lifecycle, scheduler, notifications, image cache, Docker build, security model, and verification changes.
- [ ] Add or update targeted tests for every changed lifecycle/concurrency/cache boundary.

## Findings and decisions

- Tally is intentionally a LAN/self-hosted application. First-profile creation is therefore treated as a local trust-boundary/design concern, not an internet-facing release blocker.
- Direct WAN exposure remains unsupported unless the operator deliberately adds appropriate authentication, TLS/reverse proxy, or a private-network/VPN layer.
- Node 24 LTS and Alpine 3.24 are intentionally preferred over chasing non-LTS/current release trains.
- The existing non-root runtime, dropped Linux capabilities, no-new-privileges, Argon2id, hashed session tokens, OIDC PKCE/state/nonce, CSRF/origin checks, backup extraction validation, bounded outbound HTTP, and backend authorization are to be preserved.

## Verification

- [ ] `go test ./...`
- [ ] `go vet ./...`
- [ ] `go test -race ./...`
- [ ] frontend production build
- [ ] Playwright E2E/browser regression suite
- [ ] Docker production build
- [ ] vulnerability scans/audits added by this branch
- [ ] final diff review against every item above

## Issues / notes discovered during implementation

- None yet.
