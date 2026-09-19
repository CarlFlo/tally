# Working on Tally

Tally is a self-hosted TV-show tracking application. Keep changes focused, secure, maintainable, and compatible with the existing architecture unless the task explicitly requires otherwise.

## Start here

Read `docs/TODO.md` before modifying the application. It is the single work tracker. For larger multi-step work, keep a concise active checklist there and remove or condense completed implementation detail when finished.

Load only the documentation relevant to the task:

- `docs/DEVELOPMENT.md` — implementation workflow, coding conventions, security rules, external-service handling, documentation maintenance, and finishing work.
- `docs/ARCHITECTURE.md` — system ownership, data boundaries, frontend lifecycle, providers, migrations, backups, torrent capabilities, and durable design decisions.
- `docs/LOCALIZATION.md` — locale format, user-facing text, fallback behavior, and localization workflow.
- `docs/VALIDATION.md` — required tests, builds, browser coverage, CI expectations, and validation by change type.
- `docs/LESSONS.md` — reusable failure patterns and prevention techniques. Read relevant sections before working in an area they cover.

Prefer current repository documentation and code over historical assumptions. Git history is the source for historical implementation detail.

## Non-negotiables

- Follow existing ownership and patterns rather than creating parallel systems.
- Keep authorization, feature availability, validation, and durable invariants backend-authoritative.
- Preserve existing security protections; never weaken security or tests to simplify a change.
- Use the localization system for user-facing text.
- Increment a bundled locale's `_meta.catalogVersion` whenever its keys, translated text, placeholders, or metadata change meaningfully; never reuse or decrease a bundled catalog version.
- Do not use deprecated APIs.
- Treat tests and regression coverage as part of the implementation.

## Before finishing

For branch work, treat merge readiness as a final gate: finish the task's active `docs/TODO.md` items (or explicitly record intentionally deferred scope), review affected tests and fixtures against the final diff, and require the full authoritative CI run for the exact branch HEAD to succeed. Do not merge based on an earlier green commit, a skipped same-repository pull-request run, or a partial rerun.

Run the checks required by `docs/VALIDATION.md` and investigate unexplained failures rather than bypassing them.

Update `docs/TODO.md` to the remaining current/future state and update any other documentation whose architecture, behavior, configuration, localization, or validation contract changed.

If meaningful work reveals a reusable engineering principle, refine `docs/LESSONS.md` with the generalized failure pattern, rule, and prevention/test strategy. Do not turn it into an incident log or changelog.

Remove temporary/debug code before finishing.
