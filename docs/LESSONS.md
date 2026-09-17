# Engineering lessons

This file captures reusable engineering lessons discovered while building and fixing Tally. The goal is prevention: preserve techniques that apply beyond the incident that exposed them.

Keep entries general. Do not turn this into a changelog, bug diary, branch history, or list of one-off implementation details. Refine existing lessons when a better rule emerges and remove advice that becomes obsolete.

## State and ownership

### Make the backend authoritative

UI visibility is a usability feature, not an enforcement boundary. Permissions, feature toggles, validation, replay protection, and state invariants must be enforced where the operation is executed. The frontend should mirror those capabilities so users are not offered actions that cannot succeed.

Useful check: for every hidden or disabled action, ask what happens if the API is called directly.

### Keep one source of truth for durable state

Avoid maintaining parallel client and server interpretations of persisted settings. Let the server return authoritative state and invalidate/reload the relevant query after writes. Local component state should represent drafts or transient UI state, not a second database.

The same rule applies to inventories. If the filesystem is the authoritative set of archives or assets, do not maintain a second database registry that can drift from it. Derive management operations from the authoritative inventory and cache only information that can be safely rebuilt.

### Give refresh/invalidation one owner

A completed mutation should have one authoritative path that publishes or invalidates the affected state. If the mutation handler, filesystem watcher, background job, and UI all independently trigger the same refresh, duplicate requests and ordering races follow.

Publish state changes after the authoritative operation has actually completed. For asynchronous jobs, completion is the meaningful invalidation point, not merely the button click that started the work.

### Distinguish draft state from applied state

A form value is not saved merely because the UI displays it. Staged forms should apply only after an explicit save succeeds; immediate toggles should be used only when the toggle itself is intentionally the save action.

This avoids the common UX failure where a setting appears active and later reverts when the user navigates away.

### Derived feedback must follow the new state

Do not precompute user-visible feedback from state that an action is about to change. If saving changes locale, theme, identity, or another rendering context, feedback should be resolved after the authoritative new state is active.

For localization, storing a translation key and translating at render time is safer than storing a translated string that may outlive the locale that produced it.

## Navigation and frontend lifecycle

### Fix lifecycle bugs instead of masking navigation

Cooldowns, global pointer locks, forced reloads, and temporary navigation disabling can hide symptoms while creating new failure modes. A route change should not require defensive blocking to remain safe.

Investigate leaked listeners, timers, observers, stale callbacks, unbounded work, expensive rerenders, and incorrect cleanup first. Navigation should remain responsive under repeated same-document route changes.

### Test navigation without reloading the document

A browser test that reloads between routes cannot reveal many lifecycle defects. Stress tests for navigation should keep one document alive while repeatedly changing routes, opening/closing overlays, resizing, and exercising delayed requests.

Track invariants such as active listeners, subscriptions, intervals, request concurrency, and post-stress input responsiveness when practical. Also test revisiting a route: initial-load layout can be correct while retained measurements or stale state break after navigation away and back.

### Give overlays explicit navigation semantics

If opening an overlay materially changes what the user is viewing, integrate it with browser history. Back should close the top-level overlay before leaving the underlying page. Escape and explicit Close should produce the same final UI state without leaving stale history entries.

### Cleanup must be owned by the component that created the work

Listeners, observers, timers, dialogs, subscriptions, and asynchronous operations need deterministic cleanup. Avoid relying on a later render or unrelated route effect to repair abandoned work.

### Keep accessible names stable

Labels are user-facing behavior and part of the testable interface. Dynamic validation, localization, helper text, or loading state should not unexpectedly change a control's accessible name. Prefer stable semantic labels and expose transient state separately.

Stable accessibility contracts improve assistive technology support and make browser tests less brittle at the same time.

## Async work, networking, and performance

### Bound concurrency and queues

Fast user interaction can generate more work than the backend or browser should execute at once. Bound active requests and queued work, remove cancelled queue entries, and release capacity on every success and failure path.

A bounded system degrades predictably; an unbounded one often looks fine until rapid navigation or a slow provider exposes it.

### Cancellation needs clear ownership

Latest-only UI requests should cancel superseded work. Shared requests should continue while at least one waiter still needs them and cancel when the last waiter leaves. Do not reuse already-aborted controllers.

Durable writes may be allowed to finish after navigation, but their stale callbacks must not navigate or mutate the newly active page.

### Do not cache cancellation as a result

Cancellation says that a caller stopped needing the answer; it does not prove the underlying resource is invalid or absent. Do not store cancelled, context-expired, or otherwise incomplete work as a normal cached result.

Be similarly cautious with transient failures. Preserve a valid last-known-good value when appropriate and allow later requests to recover rather than poisoning the cache with a temporary condition.

### Put deadlines around the whole operation

Timeouts should cover queue admission, the network request, and response-body consumption rather than only the socket call. Otherwise work can wait indefinitely before the timeout even starts.

### Live event streams require reconciliation

An event stream is a notification mechanism, not the durable source of truth. There is always a possible gap around initial connection or reconnection, so subscribe/register local consumers before opening the stream where possible and reconcile authoritative state after connection or recovery.

Reconnect handling should not blindly cancel healthy in-flight queries. Coalesce invalidation and let the normal query owner decide whether work should be replaced.

### Watchers need a degradation path

Filesystem watchers and similar long-lived event sources can fail, close their channels, or miss changes. Handle channel closure cleanly without spinning, and treat watcher errors as a reason to reconcile from the authoritative inventory.

A watcher is an optimization for freshness, not a second source of truth. If watching is unavailable, the application should remain usable even if freshness requires a later explicit revisit or refresh.

### Isolate frequently changing state

Do not rerender a large page subtree for a one-second countdown or similarly small changing value. Keep high-frequency state close to the component that displays it.

Cache expensive formatters and other pure helpers with a bounded strategy when repeated allocation becomes measurable, but avoid unbounded memoization caches.

### Prefer events and watchers over polling

When the application already has server events or filesystem notifications, use them to invalidate relevant state. Polling adds repeated work, race windows, and form-overwrite risks.

Do not add a polling fallback merely to support unusual filesystems unless that compatibility is an explicit product requirement.

## External integrations

### Implement the real protocol, not an assumed web-API shape

A successful endpoint may return plain text, an empty body, or a status code that differs from a typical JSON API. Validate success according to the external system's documented contract, not according to generic client assumptions.

Protocol tests should assert exact request method, headers, body, redirects, status handling, and secret behavior using local fixtures.

### Keep provider work behind one coordination boundary

Centralize admission, retries, Retry-After handling, rate limiting, circuit state, response-size limits, cancellation, and telemetry. Bypassing the coordinator for one feature creates inconsistent resource behavior that becomes difficult to reason about later.

### Feature toggles should gate capabilities, not configuration access

Put the feature toggle outside the configuration section it controls. If disabling a capability hides or disables all related configuration including its own switch, users can trap themselves.

Independent capabilities deserve independent toggles. Hiding unavailable navigation should be paired with backend rejection of the disabled operation.

### Preserve saved secrets intentionally

Editing a URL or non-secret field should not force a secret to be re-entered. Treat a blank concealed secret input as "retain existing" unless the UI explicitly offers removal. Never expose secrets in general APIs, logs, errors, telemetry, or activity records.

## Data, migrations, and backups

### Put critical invariants in the database when possible

UI and API checks improve error messages, but database constraints/triggers protect invariants against races and alternate code paths. Examples include uniqueness, ownership, and role invariants.

### Snapshot before migration and validate before commit

A schema migration should have a recovery point, run transactionally, validate both data and expected schema, and refuse unsupported downgrade scenarios. Migration tests should cover sequential upgrades from supported older versions.

### Keep migration validation version-aware

The base schema and historical fixtures represent the schema version they claim to represent; they should not silently acquire columns or invariants introduced years later. Validation that depends on a newer field or table must only run after the migration that creates it.

Build genuine legacy fixtures rather than taking the current schema and merely lowering its version number. Otherwise upgrade tests can pass while real old deployments fail.

### Restore tests must model relational side effects

Foreign-key cascades can destroy rows that were inserted earlier in the same restore procedure. Restore order matters. A backup test should verify complete relationships after restore, not merely that individual tables contain rows.

Stage and validate restore input before changing live state. Failed restore application should leave the current deployment usable.

### Back up durable state, not caches

UI-managed credentials and settings are durable application state and belong in protected backups. Rebuildable caches and environment-provided secrets should not.

### Treat filesystem inventories as untrusted input

Directories used as application inventories can contain symlinks, special files, partial files, or manually copied content. Management operations should accept only the file types they intentionally support, reject paths that escape the owned root, and avoid following symlinks accidentally.

Filesystem metadata used for ordering or cache validation should be precise enough to distinguish rapid changes; coarse timestamps can make stale entries appear current.

## Security and identity

### Authorize roles, not special identities

Do not encode administrator power into a magic account ID such as `user0`. Use generated stable identities plus explicit roles, and enforce authorization against those roles on the backend.

### Re-authenticate the actor for sensitive changes

When Administrator A changes Administrator B, verification should prove the identity of Administrator A, not ask for B's credential. This preserves a clear security model and audit trail.

### Keep internal identifiers out of the UI unless useful

Opaque IDs are implementation details. Prefer display names and domain labels in normal user-facing interfaces while retaining IDs internally for stable references.

When operator commands accept a friendly name as well as an ID, resolve names only when the match is exact and unambiguous; never guess which identity an operator intended.

### Preserve underlying errors when adding friendly lookup behavior

A lookup returning "not found" is different from the database itself failing. Convenience resolution layers should preserve infrastructure errors instead of collapsing every failure into an absent-record result.

## Time and scheduling

### Separate execution time from display time

A user's preferred timezone is presentation state; it should not silently redefine when server-owned schedules execute. Choose one explicit execution timezone for jobs and notifications, then convert resulting timestamps for each user's display preferences.

This separation avoids inconsistent scheduling across profiles and makes daylight-saving behavior testable. Preview calculations should use the same execution semantics as the scheduler itself.

## Notifications and observability

### Logs and notifications serve different purposes

Logs should preserve broad operational history. User notifications should be selective and actionable. Routine actions initiated by the same user usually do not need attention-grabbing notifications, while failed scheduled/background work often does.

Offer useful categories with sensible defaults rather than exposing every possible event as a separate preference.

### Preserve actor context in privileged activity

Security-sensitive activity is much more useful when it records who performed the action as well as what target changed. Keep stable human-readable actor/target context where appropriate so later identity deletion or migration does not make the audit trail meaningless.

### Long operations should explain shutdown progress

Graceful shutdown can legitimately spend time draining work, checkpointing data, or stopping services. Emit concise phase-level logs so operators can distinguish expected shutdown work from a hung process.

## Localization

### Localize at the rendering boundary when context can change

Translated strings have a lifecycle. If a toast, dialog, queued message, or other UI object can survive a locale change, store stable semantic data such as a translation key and interpolate when rendered.

### Keep the canonical key contract stable

Use one canonical locale as the complete key contract and allow partial translations to fall back to it. Validate catalog structure and interpolation variables so a damaged translation does not break the application.

### Keep infrastructure independent of localization startup

Low-level transport, request pools, caches, and bootstrap code should expose stable typed errors or codes rather than requiring the translation system to already be loaded. Translate at the API/UI boundary.

This avoids dependency cycles where localization needs networking while networking error handling itself needs localization.

### Hot reload should replace state and preserve last-known-good data

When a catalog or other runtime resource is hot-reloaded, make the replacement semantics explicit. Merging new content into old content can leave deleted keys behind; replace the authoritative bundle when the source changes.

If a refetch fails transiently, keep a previously valid catalog available rather than blanking the UI. Cache keys or resource revisions must change when the underlying mutable resource changes so stale data cannot remain indefinitely.

## Testing and debugging

### A regression test should fail for the original reason

Write the test so the previous bug would actually reproduce. Avoid tests that accidentally reset state, reload the page, wait away a race, or assert only the final database value when the defect was visual/lifecycle-related.

### Treat tests as implementation work

When behavior changes, review related tests and fixtures immediately. Updating tests only at merge time is useful as a final audit, but it should not be the first time changed behavior is represented in coverage.

### Prefer observable synchronization over sleeps

Browser and integration tests should wait for the state transition that proves the operation finished—an API response, row appearance, event, or stable UI condition—rather than relying on arbitrary delays. Fixed sleeps often become flaky as CI load changes and can also hide sequencing bugs.

### Keep fixtures semantically real

Fixtures should preserve the invariants of the version or role they claim to represent. Use generated opaque identities when production does; use real historical schemas for migration coverage; use artificial locale/provider data that is intentionally distinct enough for assertions.

A fixture that depends on obsolete magic IDs or current-schema shortcuts can make tests pass for behavior that no longer exists in production.

### Investigate flaky-looking failures before rerunning

Use traces, logs, and the changed-file surface to determine whether a failure is plausibly related. A single rerun can help classify a timeout as transient, but repeated failure must be treated as a real issue until explained.

### Build the assets the test will actually execute

Browser tests should exercise freshly built production assets when the application embeds or serves generated frontend output. Testing stale assets can produce misleading passes and failures.

## Maintenance and dependencies

### Remove superseded paths once the replacement is authoritative

Old scripts, styles, components, configuration references, and unused dependencies add cognitive load and can accidentally remain as a second implementation path. After a replacement is proven, remove the obsolete code and update documentation, lockfiles, and tests that referenced it.

Do not keep dead tooling merely because it once participated in validation; the current validation path should be explicit and maintained.

## Documentation

### Keep current documentation current

Roadmaps should describe current/future work, validation docs should describe the present verification contract, and architecture docs should describe durable design. Git already stores historical implementation detail.

When a fix teaches a reusable lesson, capture the generalized prevention rule here rather than appending another dated incident paragraph elsewhere.
