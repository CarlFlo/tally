# Development guide

This file contains implementation rules that apply while changing Tally. `AGENTS.md` should stay small and route coding agents here rather than duplicating these details.

## Planning and scope

- Read `TODO.md` before changing the application. It is the single work tracker.
- For larger multi-step work, keep a concise active checklist in `TODO.md` and update it as requirements, discoveries, and completion state change. Remove or condense completed implementation detail when the work is finished.
- Start from the owning domain or route and its matching tests. Prefer existing patterns over parallel implementations.
- Keep changes focused. Preserve existing UX, API contracts, data compatibility, and behavior unless the task explicitly changes them.

## Architecture and security

- Follow `ARCHITECTURE.md` for ownership, data boundaries, provider coordination, migrations, backups, and frontend lifecycle rules.
- Keep authorization, feature availability, validation, replay protection, and durable invariants backend-authoritative. UI visibility is not enforcement.
- Preserve CSRF, session, profile-isolation, request-size, security-header, and secret-redaction protections.
- Use parameterized SQL for untrusted values and validate paths, uploads, remote responses, and user input at trust boundaries.
- Do not weaken security controls to simplify an implementation or make a test pass.

## Go

- Keep code in the owning domain under `internal/`; prefer narrow dependency interfaces and concrete service types.
- Keep files focused. Roughly 150–250 lines is a readability guideline, not a hard limit.
- Avoid generic `utils` or `helpers` packages and global mutable runtime state.
- Prefer the standard library when practical and avoid unnecessary dependencies.
- Do not use deprecated APIs.

## Frontend

- Avoid duplicated authoritative state, unnecessary effects, polling, listeners, timers, or network requests.
- Clean up dialogs, subscriptions, observers, timers, listeners, and asynchronous work when components unmount.
- Keep rapid navigation and repeated interactions safe without cooldowns, global pointer locks, forced reloads, or similar symptom-masking workarounds.
- Prefer reusable domain components over premature generic abstractions.
- Keep staged form values separate from saved state unless a control is intentionally immediate-save.
- Use the localization system for user-facing text; follow `LOCALIZATION.md` for locale-specific rules.

## External services and configuration

- Infrastructure configuration is environment-owned. Operator-editable application settings and credentials are persisted in SQLite.
- Route provider-related outbound work through the existing coordinator and keep background work bounded, cancellable, and observable.
- Torrent discovery, inspection, automation, and manual search must share the backend release evaluator rather than creating frontend-specific or scheduler-specific acceptance logic.
- Manual search may display confidence only when Tally has authoritative episode context. Keep confidence separate from quality/size preference.
- Automatic torrent submission requires High confidence, a locally verified `.torrent` payload, and no hard rejection. Magnet-only candidates remain manual because their file tree cannot be verified before qBittorrent receives them.
- Deep-inspect only a bounded shortlist or explicitly selected result; never fetch every Jackett result merely to compute UI confidence.
- Re-check search/download/automation capabilities immediately before external side effects. A per-show override must never bypass a disabled global capability.
- Treat ambiguous downloader responses as reconciliation problems. When an infohash is known, check the client before retrying so a successful-but-uncertain submission does not become a duplicate.
- Keep automation/run history credential-free. Do not store provider API keys, authenticated URLs, cookies, client tokens, or secrets in decision snapshots.
- Secrets may appear only in explicitly authorized settings views where intentional. Do not expose them through general APIs, logs, errors, notifications, activity records, telemetry, or automation history.
- When editing a connection, preserve an existing secret when the concealed field is intentionally left blank unless the UI explicitly requests removal.

## Data and backups

- Preserve user data across migrations and validate schema changes before commit.
- Validate backups before restore and stage restore input before changing live state.
- UI-managed credentials and durable torrent automation state are application state and belong in protected backups. Rebuildable caches and environment-provided secrets do not.
- Keep automation decisions global and profile notifications profile-owned. Multiple followers of one show must not create multiple download decisions.

## Documentation

Keep each document focused on one purpose:

- `TODO.md` — current/future work and active implementation checklists.
- `ARCHITECTURE.md` — durable architecture, ownership, and boundaries.
- `VALIDATION.md` — required verification and the latest meaningful baseline.
- `LOCALIZATION.md` — locale format and localization behavior.
- `LESSONS.md` — generalized engineering lessons and prevention patterns.

Git history is the source for historical implementation detail. Do not turn the docs into chronological completion logs.

When a meaningful fix, investigation, refactor, or production problem reveals a reusable principle, refine `LESSONS.md`. Generalize the failure pattern, durable rule, and prevention/test strategy; avoid dates, branch names, and incident-specific narration. Prefer improving an existing lesson over adding a duplicate.

## Finishing work

- Treat tests as part of the implementation. Update affected assertions/fixtures and add regression coverage when behavior changes.
- For torrent automation changes, include deterministic coverage for release parsing/evaluation, payload inspection, candidate fallback, bad-infohash handling, retry gating, duplicate prevention, capability re-checks, ambiguous client reconciliation, and disabled/no-op states as applicable.
- Browser coverage should exercise manual confidence, Automation settings, Previous Runs explainability/feedback, feature-disabled navigation, and responsive behavior when those surfaces change.
- Run the checks required by `VALIDATION.md` for the changed surface and inspect unexplained failures before rerunning them.
- Update `TODO.md` to reflect the remaining state of work.
- Update the relevant documentation when architecture, behavior, configuration, localization, validation expectations, or reusable lessons change.
- Remove temporary/debug code before finishing.
