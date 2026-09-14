# Working on Tally

Read `project goal high level.md` and `TODO.md` before changing the application.
Continue the current milestone before unrelated features and update `TODO.md` before ending work.

- SQLite is the permanent database. Shared metadata never belongs to a profile.
- Profiles own follows, preferences, and episode state; authentication maps to immutable profile IDs.
- External IDs are mappings, never general application IDs.
- Route all outbound requests through the provider coordinator. Bound and observe background work.
- Infrastructure configuration comes from ENV; torrent clients, Torznab providers, webhooks, and job schedules are configured in the UI and persisted in SQLite. Connection secrets may be viewed only through operator settings, as requested by the user; general APIs and logs must not expose them. Torrent searches and sends are always manual.
- Back up and validate before schema upgrades. Caches and ENV secrets do not belong in backups. UI-managed client credentials are durable state and are included in protected backups.
- Run `go test ./...`, `go vet ./...`, and the frontend build after relevant changes.
- Keep Go files focused on one responsibility and target about 150-200 lines. Use domain packages under `internal/`, narrow dependency interfaces, concrete constructors, and instance-owned runtime state; avoid generic `utils` or `helpers` packages.
- Use `docs/ARCHITECTURE.md` to locate code. Preserve API contracts, SQL transactions, and coordinator policies when refactoring.
