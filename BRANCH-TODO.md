# Branch TODO — Architecture Consistency Audit

Goal: improve correctness, security, consistency, reuse, and maintainability without unnecessary abstraction.

## Discovery
- [ ] Map repository architecture, shared infrastructure, request lifecycle, frontend state/fetching, background jobs, integrations, and database boundaries.
- [ ] Identify duplicated implementations and competing conventions.
- [ ] Identify security checks that depend on individual handlers/pages remembering to apply them.
- [ ] Identify business rules implemented in multiple layers.
- [ ] Identify API/error/validation/date-time/configuration inconsistencies.
- [ ] Identify important missing tests before changing shared behavior.

## Backend
- [ ] Centralize authentication/authorization where practical; verify object-level authorization and admin/system endpoints fail closed.
- [ ] Consolidate reusable request parsing, validation, API errors, and response conventions where they represent the same responsibility.
- [ ] Move reusable business rules out of HTTP/job-specific implementations where appropriate.
- [ ] Ensure HTTP handlers and background jobs share authoritative business logic.
- [ ] Review session/cookie/header/CORS/CSRF assumptions, secret handling, log safety, URL/path handling, and external-service SSRF boundaries.
- [ ] Review database constraints, transactions, race-prone check-then-write flows, indexes, and migration safety.
- [ ] Review external clients for shared configuration, timeouts, cancellation, response validation, and safe logging.
- [ ] Review scheduled jobs for idempotency, concurrency protection, timeouts, retries, cancellation, and observability.

## Frontend
- [ ] Consolidate repeated API/error/loading/notification/modal/form behavior only where responsibilities are genuinely shared.
- [ ] Review route/permission UX while keeping backend authorization authoritative.
- [ ] Review async request cancellation/races, duplicate fetching, refresh/polling/listener cleanup, and sources of truth.
- [ ] Standardize reusable loading/empty/error/destructive-confirmation behavior where practical.
- [ ] Review date/time/locale handling and shared formatting.
- [ ] Review reusable component accessibility and focus/keyboard behavior.

## Configuration / Dependencies / Observability
- [ ] Consolidate duplicated constants/defaults/timeouts/status values where appropriate.
- [ ] Remove obsolete/unused dependencies or duplicate approaches where clearly safe.
- [ ] Review structured logging and sensitive-data redaction.

## Tests / Validation
- [ ] Add or update tests around consolidated and security-sensitive behavior.
- [ ] Run repository lint, type-check, unit/integration tests, build, static analysis, and Playwright checks available in the repo.
- [ ] Fix failures caused by this branch and proactively update brittle tests affected by intentional behavior changes.

## Final pass
- [ ] Re-scan for old duplicate implementations and dead code.
- [ ] Verify all intended consumers use authoritative shared implementations.
- [ ] Verify backend authz and validation remain authoritative and fail closed.
- [ ] Verify error handling, shared components, async cleanup, and security behavior are consistent.
- [ ] Review the complete diff for accidental behavior changes and unnecessary abstraction.
- [ ] Remove this temporary file before merge readiness.
