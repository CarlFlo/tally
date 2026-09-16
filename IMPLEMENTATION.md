# Localization implementation

Branch: `feature/profile-localization`

## Requirements
- [ ] Replace the legacy `APP_LANGUAGE` setting with per-profile localization.
- [ ] Use the latest stable `i18next` and `react-i18next`.
- [ ] Keep localization server-owned under the persistent config volume (`/config/locales` by default).
- [ ] Bundle canonical English and populate/update the config locale on startup using a translation catalog version.
- [ ] Only localize web UI. TV metadata, DB values, logs, internal server messages, and raw backend errors remain English/original.
- [ ] Localize known API error codes in the UI; unknown/raw server errors remain English.
- [ ] Store locale per profile, default/fallback to `en`, and migrate existing profiles to English.
- [ ] New-profile creation allows locale selection and previews the chosen locale immediately before save.
- [ ] Existing-profile locale changes apply immediately without logout, page reload, or server restart.
- [ ] Different sessions/profiles remain isolated.
- [ ] Discover locale files dynamically from the config locale directory.
- [ ] Watch locale files using filesystem events rather than polling.
- [ ] Hot reload valid changes and notify connected clients through the existing SSE/live-update system.
- [ ] Invalid locale files remain visible but disabled in selectors with a concise reason; log diagnostics to the server console.
- [ ] If an active locale becomes invalid/missing, render English temporarily without overwriting the stored profile preference; automatically recover when fixed.
- [ ] Embedded English remains the ultimate fallback so a broken/missing config file cannot brick the UI.
- [ ] Missing non-English translation keys fall back to English rather than invalidating a locale.
- [ ] Set document language metadata and prepare text direction metadata for future RTL locales.
- [ ] Use proper pluralization/interpolation and localize human-visible date/number formatting where appropriate.
- [ ] Remove the existing hard-coded frontend `en` translation object.

## Design decisions
- Canonical locale IDs use stable language/locale codes such as `en`, `sv`, `es`.
- English is authoritative for the translation key contract.
- Each locale file contains metadata including locale, display name, direction, and independent catalog version.
- Catalog versioning is independent of the Tally application version.
- The locale watcher monitors the directory (not individual files), debounces editor write bursts, validates candidates before swapping the in-memory registry, and never exposes partially written data.
- Existing SSE `/api/events` infrastructure is reused for locale-registry invalidation.
- Browser language detection is not authoritative and no language-detector dependency is used.
- No external translation service or custom translation tooling.

## Implementation checklist

### Branch / tracking
- [x] Create feature branch.
- [x] Add `IMPLEMENTATION.md`.

### Dependencies
- [ ] Add `i18next@26.4.2`.
- [ ] Add `react-i18next@17.0.14`.
- [ ] Add Go filesystem watcher dependency if required and pin the latest stable compatible version.

### Backend localization
- [ ] Add embedded canonical English locale.
- [ ] Add locale file schema/metadata model.
- [ ] Add localization registry with safe concurrent reads/atomic replacement.
- [ ] Add validation with concise public errors and detailed console diagnostics.
- [ ] Create/sync config locale directory on startup.
- [ ] Restore/update bundled English based on catalog version.
- [ ] Add event-based directory watcher with debounce and rename/create/delete handling.
- [ ] Add locale list/catalog API endpoints.
- [ ] Add cache revision/ETag behavior where useful.
- [ ] Publish locale changes through the existing live event hub.
- [ ] Gracefully stop watcher during server shutdown.

### Profile persistence
- [ ] Add DB migration for `profiles.locale` defaulting to `en`.
- [ ] Update fresh schema.
- [ ] Update profile repository/model and all profile queries/API responses.
- [ ] Validate requested locale on create/update.
- [ ] Preserve unavailable stored locale preferences while rendering through English fallback.
- [ ] Default OIDC-created profiles to English.
- [ ] Remove `APP_LANGUAGE` from config, examples, docs, and validation.

### Frontend runtime
- [ ] Initialize i18next/react-i18next.
- [ ] Load available locales and current catalog from the server.
- [ ] Use profile locale as authoritative active locale.
- [ ] Add temporary locale preview state to unsaved profile creation.
- [ ] Hot reload locale catalog/availability after SSE invalidation.
- [ ] Switch active locale immediately.
- [ ] Set `<html lang>` and `dir`.
- [ ] Ensure unavailable active locale renders English and automatically recovers.

### UI conversion
- [ ] Replace hard-coded `en` object.
- [ ] Convert navigation/document titles/breadcrumbs.
- [ ] Convert profile/login/registration UI.
- [ ] Convert calendar/show/search UI.
- [ ] Convert settings/system/jobs/logs UI chrome.
- [ ] Convert dialogs/buttons/placeholders/empty states.
- [ ] Convert frontend-generated toasts/validation messages.
- [ ] Convert accessibility labels/tooltips.
- [ ] Add known API error-code localization mapping.
- [ ] Localize human-visible date/number labels while keeping canonical internal formatting stable.
- [ ] Audit string concatenation and use interpolation/pluralization.

### Documentation
- [ ] Add `docs/LOCALIZATION.md` with translation-file format and contributor guidance.
- [ ] Document fallback/hot-reload behavior.

### Tests / validation
- [ ] Backend locale registry validation tests.
- [ ] Startup sync/version tests.
- [ ] File watcher hot-reload tests.
- [ ] Invalid/add/remove/repair locale tests.
- [ ] DB migration tests.
- [ ] Profile locale create/update/isolation tests.
- [ ] Active-invalid-locale fallback/recovery tests.
- [ ] SSE locale invalidation tests.
- [ ] Frontend/E2E language selector and preview tests.
- [ ] E2E immediate switch tests.
- [ ] Final scan for unintended hard-coded user-facing UI strings.
- [ ] Run Go tests.
- [ ] Run TypeScript build.
- [ ] Run Playwright.
- [ ] Record final validation results below.

## Issues / discoveries
- Tally already has a profile-aware SSE live-update hub; localization should reuse it rather than add WebSockets or another stream.
- Tally currently has a small hard-coded frontend `en` object; it should be removed as part of this work.
- Tally currently has `APP_LANGUAGE=en`; this will be removed rather than coexist with profile localization.

## Validation results
_Not run yet._
