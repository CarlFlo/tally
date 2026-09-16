# Localization implementation

Branch: `feature/profile-localization`

## Requirements
- [x] Replace the legacy `APP_LANGUAGE` setting with per-profile localization.
- [x] Use the latest stable `i18next` and `react-i18next`.
- [x] Keep localization server-owned under the persistent config volume (`/config/locales` by default).
- [x] Bundle canonical English and populate/update the config locale on startup using a translation catalog version.
- [x] Only localize web UI. TV metadata, DB values, logs, internal server messages, and raw backend errors remain English/original.
- [x] Localize known API error codes in the UI; unknown/raw server errors remain English.
- [x] Store locale per profile, default/fallback to `en`, and migrate existing profiles to English.
- [x] New-profile creation allows locale selection and previews the chosen locale immediately before save.
- [x] Existing-profile locale changes apply immediately without logout, page reload, or server restart.
- [x] Different sessions/profiles remain isolated.
- [x] Discover locale files dynamically from the config locale directory.
- [x] Watch locale files using filesystem events rather than polling.
- [x] Hot reload valid changes and notify connected clients through the existing SSE/live-update system.
- [x] Invalid locale files remain visible but disabled in selectors with a concise reason; log diagnostics to the server console.
- [x] If an active locale becomes invalid/missing, render English temporarily without overwriting the stored profile preference; automatically recover when fixed.
- [x] Embedded English remains the ultimate fallback so a broken/missing config file cannot brick the UI.
- [x] Missing non-English translation keys fall back to English rather than invalidating a locale.
- [x] Set document language metadata and prepare text direction metadata for future RTL locales.
- [x] Use proper pluralization/interpolation and localize human-visible date/number formatting where appropriate.
- [x] Remove the existing hard-coded frontend `en` translation object.

## Design decisions
- Canonical locale IDs use stable language/locale codes such as `en`, `sv`, `es`.
- English is authoritative for the translation key contract.
- Each locale file contains metadata including locale, display name, direction, and independent catalog version.
- Catalog versioning is independent of the Tally application version.
- The locale watcher monitors the directory (not individual files), debounces editor write bursts, validates candidates before swapping the in-memory registry, and never exposes partially written data.
- Existing SSE `/api/events` infrastructure is reused for locale-registry invalidation.
- Browser language detection is not authoritative and no language-detector dependency is used.
- No external translation service or custom translation tooling.
- Only `en` is bundled/supported by Tally initially. Any additional locale used by automated tests must be clearly artificial/test-only and must not ship as a supported translation.
- Filesystem watching is event-driven only. Localization and backup archives use the same fsnotify/debounce pattern while keeping their validation/domain logic separate; no polling fallback is used.

## Implementation checklist

### Branch / tracking
- [x] Create feature branch.
- [x] Add `IMPLEMENTATION.md`.

### Dependencies
- [x] Add `i18next@26.4.2`.
- [x] Add `react-i18next@17.0.14`.
- [x] Add `github.com/fsnotify/fsnotify@v1.10.1` (latest stable).

### Backend localization
- [x] Add embedded canonical English locale.
- [x] Add locale file schema/metadata model.
- [x] Add localization registry with safe concurrent reads/atomic replacement.
- [x] Add validation with concise public errors and detailed console diagnostics.
- [x] Create/sync config locale directory on startup.
- [x] Restore/update bundled English based on catalog version and merge embedded fallback keys at runtime.
- [x] Add event-based directory watcher with debounce and rename/create/delete handling.
- [x] Add locale list/catalog API endpoints.
- [x] Add registry revision-based client cache invalidation and `Cache-Control: no-store` for locale APIs.
- [x] Publish locale changes through the existing live event hub.
- [x] Gracefully stop watcher during server shutdown.

### Profile persistence
- [x] Add DB migration for `profiles.locale` defaulting to `en`.
- [x] Keep `schema.sql` as the v1 baseline; fresh installs receive locale through migration 007.
- [x] Update profile repository/model and all profile queries/API responses.
- [x] Validate requested locale on create/update.
- [x] Preserve unavailable stored locale preferences while rendering through English fallback.
- [x] Default OIDC-created profiles to English through the database default.
- [x] Remove `APP_LANGUAGE` from config, settings API, examples, docs, and validation.

### Frontend runtime
- [x] Initialize i18next/react-i18next.
- [x] Load available locales and current catalog from the server.
- [x] Use profile locale as authoritative active locale.
- [x] Add temporary locale preview state to unsaved profile creation.
- [x] Hot reload locale catalog/availability after SSE invalidation.
- [x] Switch active locale immediately.
- [x] Set `<html lang>` and `dir`.
- [x] Ensure unavailable active locale renders English and automatically recovers.

### UI conversion
- [x] Replace hard-coded `en` object.
- [x] Convert navigation/document titles/breadcrumbs.
- [x] Convert profile/login/registration UI.
- [x] Convert calendar/show/search UI.
- [x] Convert settings/system/jobs/logs UI chrome.
- [x] Convert dialogs/buttons/placeholders/empty states.
- [x] Convert frontend-generated toasts/validation messages.
- [x] Convert accessibility labels/tooltips.
- [x] Add known API error-code localization mapping.
- [x] Localize human-visible date/number labels while keeping canonical internal formatting stable.
- [x] Audit string concatenation and use interpolation/pluralization.

### Backup archive live refresh
- [x] Add event-driven filesystem watching for the backup archive directory; do not poll.
- [x] Reuse the localization watch pattern/helper where practical, while keeping backup and localization validation/domain logic separate.
- [x] Watch the backup directory rather than individual archive files so create/rename/remove workflows are handled reliably.
- [x] Filter filesystem events aggressively to relevant completed backup archive files (primarily `.zip`) and ignore temporary/staging files and unrelated directory noise.
- [x] Debounce bursts of filesystem events caused by copy/rename/write operations.
- [x] Publish the existing global `backups` SSE resource when an external backup archive is created, renamed into the directory, changed, or removed.
- [x] Ensure Tally-created manual/scheduled backups continue to publish `backups` after successful completion without duplicate or excessive UI refreshes.
- [x] Ensure the frontend invalidates/refetches the active backup archive query when a `backups` live event arrives.
- [x] Remove the Backup archives manual Refresh button once event-driven refresh is verified.
- [x] Gracefully stop the backup directory watcher during server shutdown.
- [x] Test external archive copy/create detection.
- [x] Test archive rename/move-into-directory detection.
- [x] Test external archive deletion detection.
- [x] Test that temporary/non-archive files do not trigger unnecessary backup-list refreshes.
- [x] Test manual backup completion appears automatically without page reload or manual refresh.
- [x] Test the `/settings` Backup archives list updates automatically while the page is open.

### Post-review hardening
- [x] Make localization watcher shutdown safe when fsnotify channels close and wait for its goroutine to exit.
- [x] Treat localization filesystem watching as optional at startup: keep serving loaded/embedded locales if fsnotify cannot start; do not add polling.
- [x] Reconcile localization state after watcher errors/overflow so missed filesystem events do not leave stale catalogs.
- [x] Limit locale JSON file size before reading/parsing to avoid accidental excessive memory use.
- [x] Avoid redundant backup archive refreshes on Tally-initiated delete while preserving immediate local UI refresh and watcher-based updates for other clients.
- [x] Add focused regression tests for the watcher lifecycle, oversized locale handling, and backup delete refresh behavior.
- [ ] Re-run the full validation workflow after these fixes.

### Documentation
- [x] Add `docs/LOCALIZATION.md` with translation-file format and contributor guidance.
- [x] Document fallback/hot-reload behavior.

### Remaining cleanup before merge
- [x] Replace the current Swedish `sv` browser/E2E fixture with an obviously artificial test-only locale (for example `zz-Test`) so the repository does not imply Swedish is bundled or supported.
- [x] Fix immediate unsaved locale-preview rerender in the profile settings screen.
- [x] Align the low-level request-pool test with the localized API/UI error boundary while preserving the internal typed overload error.
- [x] Resolve the current Backup archives Playwright failures by ensuring the `backups` SSE invalidation reliably refreshes the visible archive list.
- [x] Re-run the complete Playwright suite after the backup watcher/live-refresh implementation.
- [x] Run Docker image build and Trivy after Playwright is green.
- [x] Perform one final branch diff/remnant review against `master`.
- [x] Update this file with the final green workflow/run results before merge.

### Tests / validation
- [x] Backend locale registry validation tests.
- [x] Startup sync/version tests.
- [x] File watcher hot-reload tests.
- [x] Invalid/add/remove/repair locale tests.
- [x] DB migration tests.
- [x] Profile locale create/update/isolation tests.
- [x] Active-invalid-locale fallback/recovery tests.
- [x] SSE locale invalidation tests.
- [x] Frontend/E2E language selector and preview tests.
- [x] E2E immediate switch/persistence/fallback tests.
- [x] Final scan for unintended hard-coded user-facing UI strings.
- [x] Run Go tests.
- [x] Run TypeScript build.
- [x] Run Playwright.
- [x] Record final validation results below.

## Issues / discoveries
- Tally's existing profile-aware SSE live-update hub is reused for localization and backup archive invalidation; no parallel WebSocket/event system was added.
- The former hard-coded frontend `en` object has been removed and replaced by the server-owned catalog.
- `APP_LANGUAGE` has been removed completely rather than coexisting with profile localization.
- `schema.sql` is intentionally the v1 baseline; adding `locale` there would break fresh installs when migration 007 runs.
- Locale catalog query keys include the server registry revision so editing an already-active locale reloads the actual catalog, not only the locale index.
- Invalid locale metadata uses the filename as identity so a broken file cannot shadow another valid locale.
- A same/newer custom English catalog is preserved on disk but merged over embedded canonical English in memory, keeping English a complete final fallback.

## Validation results
- Post-review hardening implemented on branch head; GitHub Actions run `35121826498` is queued/pending validation.
- Dependency versions verified: i18next 26.4.2, react-i18next 17.0.14, fsnotify v1.10.1.
- Repository remnant scan found no remaining `APP_LANGUAGE`, `c.Language`, or legacy frontend `export const en` references.
- npm install/audit, TypeScript/Vite build, Go vet + race tests, and govulncheck have passed on recent localization branch heads.
- Full GitHub Actions run `35111968905` passed on implementation commit `94e84c5d074bb303ce6a84673a8784661e4980f0`.
- Passed: npm ci, npm high-severity audit, TypeScript/Vite build, Go vet, Go race tests, govulncheck, complete Playwright suite, Docker image build, and Trivy scan.
- Backup archive live refresh is event-driven only. There is no polling fallback; if filesystem notifications are unavailable, Tally logs a warning and the archive list updates on normal page reload.
- The browser/E2E secondary locale is the artificial test-only `zz-Test` fixture. Only English (`en`) is bundled/supported in production.
