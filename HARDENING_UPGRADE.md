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

- [x] Fix long-lived SSE handling so the server-wide write timeout cannot terminate healthy event streams.
- [x] Replace the one-second scheduled-job SQLite polling loop with a next-deadline timer plus explicit wakeups when schedules change.
- [x] Replace the one-second notification worker polling loop with event/deadline-driven wakeups and a 30-second reconciliation fallback.
- [x] Remove duplicate image-body caching between provider `cache.db` and the filesystem image cache while retaining shared-request coalescing.
- [x] Stream cached image files instead of loading the complete file into memory.
- [x] Publish image cache files through temp-file + atomic rename so concurrent requests cannot observe partial files.
- [x] Make SQLite WAL checkpoint policy explicit for the main and provider-cache databases and checkpoint during maintenance/clean shutdown.
- [x] Split shutdown phase budgets so job cancellation cannot consume the entire HTTP-drain budget.
- [x] Fix scheduled-job next-run recalculation to consistently use deployment `TZ`/DST semantics.

### Frontend efficiency and navigation

- [x] Remove the global `<Routes key={location.pathname}>` remount and preserve normal React Router lifecycle.
- [x] Add route-level lazy loading for System, Settings and Logs while keeping Calendar, Shows and Search eager.
- [x] Remove 500 ms manual-backup job polling and rely on live job/backups invalidation; EventSource recovery already revalidates active server data after a connection failure.
- [x] Reduce repeated backup archive ZIP inspection by caching manifest results keyed by filename, size and nanosecond mtime while preserving manual Refresh semantics.

### Dependencies and image/build

- [x] Upgrade `modernc.org/sqlite` from v1.58.0 to v1.59.0 with verified module checksums committed to `go.sum`.
- [x] Re-check Go and npm dependencies; retain stable/LTS versions rather than perform major framework migrations solely for version-number parity.
- [x] Keep Node 24 LTS for the frontend builder and Alpine 3.24 for the runtime.
- [x] Remove redundant runtime `tzdata`; Tally already embeds Go's timezone database.
- [x] Separate production image build from test execution so normal image builds do not rerun the Go suite.
- [x] Add BuildKit cache mounts for Go modules/build cache and npm package cache.

### Security and operational hardening

- [x] Keep the LAN-only threat model explicit in documentation and warn against direct public-internet exposure.
- [x] Raise the default local password/PIN minimum from 4 to 6 while preserving the environment override.
- [x] Preserve current first-profile bootstrap behavior; do not add setup-token complexity for the trusted-LAN model.
- [x] Add `govulncheck ./...` to CI using `golang.org/x/vuln` v1.8.0, which supports Go 1.27. A staged branch scan reports zero called/imported-package vulnerabilities.
- [x] Add npm dependency audit for high/critical vulnerabilities to CI. Current branch audit reports zero vulnerabilities.
- [~] Add a high/critical Trivy container-image vulnerability scan to CI. The first scan correctly caught fixed OpenSSL CVE-2026-14456 in stale Alpine base packages; the runtime now runs `apk upgrade --no-cache` so `libcrypto3`/`libssl3` receive the fixed 3.5.8-r0 packages. Final rescan is pending.
- [x] Add Dependabot coverage for Go modules, npm, Docker images and GitHub Actions.

### Maintainability

- [-] Replace the low-activity `github.com/lnquy/cron` description dependency. Reassessed and intentionally retained: it has zero transitive dependencies, is not on the execution/security path, and supports a much broader cron-description grammar than a small replacement. Removing it now would trade a cosmetic dependency-age concern for schedule-description regressions. The scheduler itself remains `robfig/cron/v3`.
- [x] Update README and architecture/TODO documentation for lifecycle, scheduler, notifications, image cache, Docker build and security model.
- [x] Add targeted tests for scheduler and notification deadline calculations; existing navigation/browser/race suites cover the changed lifecycle boundaries.

## Findings and decisions

- Tally is intentionally a LAN/self-hosted application. First-profile creation is therefore treated as a local trust-boundary/design concern, not an internet-facing release blocker.
- Direct WAN exposure remains unsupported unless the operator deliberately adds appropriate authentication, TLS/reverse proxy, or a private-network/VPN layer.
- Node 24 LTS and Alpine 3.24 are intentionally preferred over chasing non-LTS/current release trains.
- React Router 8 is not being adopted solely for version parity; the current React Router compatibility package works with the existing architecture and a major router migration is unrelated to this hardening pass.
- The existing non-root runtime, dropped Linux capabilities, no-new-privileges, Argon2id, hashed session tokens, OIDC PKCE/state/nonce, CSRF/origin checks, backup extraction validation, bounded outbound HTTP, and backend authorization are preserved.

## Verification

- [~] `go test ./...` — covered by successful staged branch runs; final head pending.
- [~] `go vet ./...` — covered by successful staged branch runs; final head pending.
- [~] `go test -race ./...` — runtime and navigation batches pass; final head pending.
- [~] frontend production build — passes on staged branch runs; final head pending.
- [~] Playwright E2E/browser regression suite — runtime and navigation batches pass, including same-document navigation stress; final head pending.
- [~] Docker production build — passes before the final dependency/scan additions; final head pending.
- [~] vulnerability scans/audits — npm audit and Go vulnerability analysis pass; Trivy identified CVE-2026-14456 and the image fix is committed; final rescan pending.
- [ ] final diff review against every item above.

## Issues / notes discovered during implementation

- [x] Scheduled job start used the UTC parser directly when advancing `next_run`, bypassing deployment timezone handling. Fixed to use the same timezone-aware scheduler helper used elsewhere.
- [x] `golang.org/x/vuln` v1.1.4 panics under Go 1.27 (`unexpected expr: *ast.KeyValueExpr`) because its bundled `x/tools` is too old. CI now pins the current upstream x/vuln commit from 2026-09-08, which passes under Go 1.27.
- [x] SQLite 1.59.0 checksums were captured from CI, committed to `go.sum`, and the temporary checksum-generation workflow steps were removed.
- [x] Removing the global route remount and adding lazy secondary routes passed the existing same-document navigation stress suite, including bounded listener/stream/request checks.

- [x] Trivy found CVE-2026-14456 in the Alpine runtime's OpenSSL 3.5.7-r0 packages, with 3.5.8-r0 already available. The Docker build now upgrades runtime packages before installing CA certificates; the security gate remains strict for fixed HIGH/CRITICAL findings.
- [x] Final cache review found that a cancelled backup-list request could otherwise cache `context.Canceled` for an unchanged archive. Cancellation/deadline errors are now excluded from the manifest cache and covered by a regression test.
