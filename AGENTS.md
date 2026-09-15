# Working on Tally

Tally is a self-hosted TV-show tracking application. Keep changes focused, maintainable, secure, and compatible with the existing architecture.

## Before making changes

* Read `docs/TODO.md` before modifying the application.
* Continue the current milestone before starting unrelated work unless the task explicitly requires otherwise.
* Use `docs/ARCHITECTURE.md` to understand ownership, boundaries, and where code belongs.
* Read other relevant files in `docs/` when working in an area they cover.
* Update `docs/TODO.md` when work changes the state, scope, or completion of planned tasks.

## Architecture

* SQLite is the permanent database.
* Shared metadata never belongs to a profile.
* Profiles own follows, preferences, and episode state.
* Authentication maps to immutable profile IDs.
* External IDs are mappings, never general application IDs.
* Preserve API contracts, SQL transaction boundaries, database invariants, and provider-coordinator policies when refactoring.
* Prefer existing abstractions and patterns over introducing parallel implementations.

## External services and configuration

* Infrastructure configuration comes from environment variables.
* Torrent clients, Torznab providers, webhooks, and job schedules are configured through the UI and persisted in SQLite.
* Route provider-related outbound work through the appropriate coordinator and keep background work bounded and observable.
* Torrent searches and torrent sends are always user-initiated/manual.
* Connection secrets may be exposed only through authorized operator settings where intentionally supported. Do not expose secrets through general APIs, logs, errors, or telemetry.

## Data and backups

* Validate migrations and preserve user data across upgrades.
* Backups must be validated before restore.
* Caches and environment-provided secrets do not belong in backups.
* UI-managed credentials are durable application state and belong in protected backups.

## Security

* Enforce authorization on the backend; never rely on the UI to protect privileged operations.
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
* Avoid duplicated state, unnecessary effects, polling, listeners, timers, or network requests.
* Clean up subscriptions, observers, timers, and asynchronous work when components unmount.
* Keep navigation and rapidly repeated interactions safe and responsive.
* Prefer reusable domain components over premature generic abstractions.

## Verification

Run checks appropriate to the change before considering work complete.

For backend changes:

* `go test ./...`
* `go vet ./...`

For frontend changes:

* frontend tests relevant to the change
* production frontend build

For changes spanning both, run both sets of checks.

Treat tests as part of the implementation. Whenever code or behavior changes, update or add relevant automated tests in the same change whenever possible, including tests exercised by GitHub Actions and browser/end-to-end coverage.

If an intentional change invalidates existing assertions, fixtures, snapshots, or workflow expectations, update them alongside the code. Run the same checks used by the GitHub workflow when feasible and resolve workflow failures caused by the change before finishing.

Fix failures caused by the change. Do not silently bypass, disable, or weaken existing tests.

## Finishing work

Before ending:

* Confirm the requested behavior is implemented.
* Check for regressions in nearby functionality.
* Remove temporary/debug code.
* Update `docs/TODO.md` where applicable.
* Keep documentation consistent with any architecture, behavior, configuration, or API changes.
