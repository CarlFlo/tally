# Working on Tally

Tally is a self-hosted TV-show tracking application. Keep changes focused, maintainable, secure, and compatible with the existing architecture.

## Before making changes

* Read `docs/TODO.md` before modifying the application.
* Continue current planned work before starting unrelated work unless the task explicitly requires otherwise.
* Use `docs/ARCHITECTURE.md` to understand ownership, boundaries, and where code belongs.
* Read `docs/LESSONS.md` when working in an area covered by an existing lesson, especially navigation/lifecycle, async work, migrations/backups, authorization, integrations, localization, or testing.
* Read other relevant files in `docs/` when working in an area they cover.
* Use `docs/TODO.md` as the single work tracker. For larger, multi-step changes with several requirements or acceptance criteria, add a focused checklist there before implementation and keep it updated with planned, completed, and newly discovered work throughout the change. Small or straightforward changes do not need a detailed checklist.
* Keep `docs/TODO.md` focused on current and future work. When a larger change is complete, remove its completed implementation checklist or condense it to any genuinely remaining follow-up work instead of keeping a completion diary.

## Architecture

* SQLite is the permanent database.
* Shared metadata never belongs to a profile.
* Profiles own follows, preferences, localization, and episode state.
* Authentication maps to immutable profile IDs; authorization is role-based.
* External IDs are mappings, never general application IDs.
* Preserve API contracts, SQL transaction boundaries, database invariants, and provider-coordinator policies when refactoring.
* Prefer existing abstractions and patterns over introducing parallel implementations.

## External services and configuration

* Infrastructure configuration comes from environment variables.
* Torrent search/client settings, webhooks, notifications, and job schedules are configured through the UI and persisted in SQLite.
* Route provider-related outbound work through the appropriate coordinator and keep background work bounded and observable.
* Torrent searches and torrent sends are always user-initiated/manual.
* Connection secrets may be exposed only through authorized operator settings where intentionally supported. Do not expose secrets through general APIs, logs, errors, or telemetry.

## Data and backups

* Validate migrations and preserve user data across upgrades.
* Backups must be validated before restore.
* Caches and environment-provided secrets do not belong in backups.
* UI-managed credentials are durable application state and belong in protected backups.

## Security

* Enforce authorization and feature availability on the backend; never rely on the UI as the security or capability boundary.
* Use parameterized SQL for all untrusted values.
* Validate paths, uploaded data, remote responses, and user-controlled input at trust boundaries.
* Preserve CSRF, session, profile-isolation, request-size, and security-header protections.
* Do not weaken security controls simply to make a test or implementation easier.

## Go

* Keep packages and files focused on a clear responsibility.
* Prefer domain packages under `internal/`, narrow dependency interfaces, concrete constructors, and instance-owned runtime state.
* Avoid generic `utils` or `helpers` packages.
* Split files when doing so improves ownership or readability; roughly 150–250 lines is a guideline, not a hard limit.
* Prefer standard-library functionality where practical and avoid unnecessary dependencies.
* Do not use deprecated APIs.

## Frontend

* Preserve existing UX and visual behavior unless the task explicitly changes it.
* Avoid duplicated authoritative state, unnecessary effects, polling, listeners, timers, or network requests.
* Clean up subscriptions, observers, timers, dialogs, and asynchronous work when components unmount.
* Keep navigation and rapidly repeated interactions safe and responsive without cooldowns or global interaction locks.
* Prefer reusable domain components over premature generic abstractions.
* When adding or changing user-facing text, use the project's localization system instead of hardcoding strings.
* Keep staged form values distinct from saved state; do not visually apply unsaved settings unless the control is intentionally immediate-save.

## Documentation and lessons

Keep documentation separated by purpose:

* `docs/TODO.md` — the single tracker for current and future work, including temporary detailed checklists for larger active changes; not a completion diary.
* `docs/ARCHITECTURE.md` — durable current architecture, ownership, and boundaries; not milestone history.
* `docs/VALIDATION.md` — current verification expectations and latest meaningful baseline; not CI transcripts.
* `docs/LOCALIZATION.md` — locale format and localization-specific behavior.
* `docs/LESSONS.md` — generalized engineering lessons that can prevent future bugs or unnecessary complexity.

When a meaningful fix, investigation, refactor, or production issue reveals a reusable principle, update `docs/LESSONS.md` before finishing. Generalize the lesson so it remains useful in future projects: describe the failure pattern, the durable rule, and how to prevent or test for it. Do not copy incident-specific narratives, dates, branch names, or one-off implementation details into the lessons file. Prefer refining an existing lesson over adding a duplicate. Trivial wording/style edits do not need a new lesson.

Periodically remove or rewrite stale documentation when the architecture changes. Git history is the source for historical implementation detail.

## Verification

Run checks appropriate to the change before considering work complete.

For backend changes:

* `go test ./...`
* `go vet ./...`

For frontend changes:

* frontend tests relevant to the change
* production frontend build

For changes spanning both, run both sets of checks.

Use `go test -race ./...` for concurrency/lifecycle/provider changes. Run the full browser suite when shared navigation, dialogs, localization, live updates, settings, or reusable frontend state changes. Build/container checks apply when deployment behavior changes.

Treat tests as part of the implementation. When behavior changes, review affected tests, update outdated assertions/fixtures, and add regression coverage where needed. Before a branch is considered ready to merge, perform a final test review and run the relevant checks, including GitHub Actions and browser/end-to-end coverage where applicable.

Resolve failures caused by the change before finishing. Do not silently bypass, disable, or weaken tests to make a change pass. If an apparently unrelated failure occurs, inspect it before rerunning; repeated failures must be explained or fixed.

## Finishing work

Before ending:

* Confirm the requested behavior is implemented.
* Check for regressions in nearby functionality.
* Remove temporary/debug code.
* Update `docs/TODO.md`: leave only unfinished or future work, and remove completed detailed implementation checklists.
* Update `docs/LESSONS.md` when the work produced a reusable lesson.
* Keep documentation consistent with architecture, behavior, configuration, localization, and API changes.
