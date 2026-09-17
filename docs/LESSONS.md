# Engineering lessons

This file captures reusable engineering lessons discovered while building and fixing Tally. The goal is prevention: preserve techniques that apply beyond the incident that exposed them.

Keep entries general. Do not turn this into a changelog, bug diary, branch history, or list of one-off implementation details. Refine existing lessons when a better rule emerges and remove advice that becomes obsolete.

## State and ownership

### Make the backend authoritative

UI visibility is a usability feature, not an enforcement boundary. Permissions, feature toggles, validation, replay protection, and state invariants must be enforced where the operation is executed. The frontend should mirror those capabilities so users are not offered actions that cannot succeed.

Useful check: for every hidden or disabled action, ask what happens if the API is called directly.

### Keep one source of truth for durable state

Avoid maintaining parallel client and server interpretations of persisted settings. Let the server return authoritative state and invalidate/reload the relevant query after writes. Local component state should represent drafts or transient UI state, not a second database.

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

Track invariants such as active listeners, subscriptions, intervals, request concurrency, and post-stress input responsiveness when practical.

### Give overlays explicit navigation semantics

If opening an overlay materially changes what the user is viewing, integrate it with browser history. Back should close the top-level overlay before leaving the underlying page. Escape and explicit Close should produce the same final UI state without leaving stale history entries.

### Cleanup must be owned by the component that created the work

Listeners, observers, timers, dialogs, subscriptions, and asynchronous operations need deterministic cleanup. Avoid relying on a later render or unrelated route effect to repair abandoned work.

## Async work, networking, and performance

### Bound concurrency and queues

Fast user interaction can generate more work than the backend or browser should execute at once. Bound active requests and queued work, remove cancelled queue entries, and release capacity on every success and failure path.

A bounded system degrades predictably; an unbounded one often looks fine until rapid navigation or a slow provider exposes it.

### Cancellation needs clear ownership

Latest-only UI requests should cancel superseded work. Shared requests should continue while at least one waiter still needs them and cancel when the last waiter leaves. Do not reuse already-aborted controllers.

Durable writes may be allowed to finish after navigation, but their stale callbacks must not navigate or mutate the newly active page.

### Put deadlines around the whole operation

Timeouts should cover queue admission, the network request, and response-body consumption rather than only the socket call. Otherwise work can wait indefinitely before the timeout even starts.

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

### Restore tests must model relational side effects

Foreign-key cascades can destroy rows that were inserted earlier in the same restore procedure. Restore order matters. A backup test should verify complete relationships after restore, not merely that individual tables contain rows.

Stage and validate restore input before changing live state. Failed restore application should leave the current deployment usable.

### Back up durable state, not caches

UI-managed credentials and settings are durable application state and belong in protected backups. Rebuildable caches and environment-provided secrets should not.

## Security and identity

### Authorize roles, not special identities

Do not encode administrator power into a magic account ID such as `user0`. Use generated stable identities plus explicit roles, and enforce authorization against those roles on the backend.

### Re-authenticate the actor for sensitive changes

When Administrator A changes Administrator B, verification should prove the identity of Administrator A, not ask for B's credential. This preserves a clear security model and audit trail.

### Keep internal identifiers out of the UI unless useful

Opaque IDs are implementation details. Prefer display names and domain labels in normal user-facing interfaces while retaining IDs internally for stable references.

## Notifications and observability

### Logs and notifications serve different purposes

Logs should preserve broad operational history. User notifications should be selective and actionable. Routine actions initiated by the same user usually do not need attention-grabbing notifications, while failed scheduled/background work often does.

Offer useful categories with sensible defaults rather than exposing every possible event as a separate preference.

### Long operations should explain shutdown progress

Graceful shutdown can legitimately spend time draining work, checkpointing data, or stopping services. Emit concise phase-level logs so operators can distinguish expected shutdown work from a hung process.

## Localization

### Localize at the rendering boundary when context can change

Translated strings have a lifecycle. If a toast, dialog, queued message, or other UI object can survive a locale change, store stable semantic data such as a translation key and interpolate when rendered.

### Keep the canonical key contract stable

Use one canonical locale as the complete key contract and allow partial translations to fall back to it. Validate catalog structure and interpolation variables so a damaged translation does not break the application.

## Testing and debugging

### A regression test should fail for the original reason

Write the test so the previous bug would actually reproduce. Avoid tests that accidentally reset state, reload the page, wait away a race, or assert only the final database value when the defect was visual/lifecycle-related.

### Treat tests as implementation work

When behavior changes, review related tests and fixtures immediately. Updating tests only at merge time is useful as a final audit, but it should not be the first time changed behavior is represented in coverage.

### Investigate flaky-looking failures before rerunning

Use traces, logs, and the changed-file surface to determine whether a failure is plausibly related. A single rerun can help classify a timeout as transient, but repeated failure must be treated as a real issue until explained.

### Build the assets the test will actually execute

Browser tests should exercise freshly built production assets when the application embeds or serves generated frontend output. Testing stale assets can produce misleading passes and failures.

## Documentation

### Keep current documentation current

Roadmaps should describe current/future work, validation docs should describe the present verification contract, and architecture docs should describe durable design. Git already stores historical implementation detail.

When a fix teaches a reusable lesson, capture the generalized prevention rule here rather than appending another dated incident paragraph elsewhere.
