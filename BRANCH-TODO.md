# Advanced Automation Flows branch

## Implementation

- [x] Define a bounded, typed node registry and validate graphs on the server.
- [x] Persist flow revisions and execution records with a schema migration and backup coverage.
- [x] Execute an initial Tally-specific flow slice, with dry-run replay of historical triggers and no side effects.
- [x] Add administrator flow editor, run list, and debugger with localized UI.
- [x] Complete flow rename and confirmed deletion, and fix canvas sizing so the graph renders in the board.
- [x] Finish local validation and resolve application failures.
- [x] Review the branch diff, address findings, and update durable documentation.
- [ ] Verify authoritative CI for the exact branch HEAD before any merge.

## Design decisions

- Advanced flows are separate from the existing torrent automation scheduler and remain opt-in/experimental. This first slice does not route live scheduled events into user flows.
- Historical replay is dry-run only. The original run record and a new replay are distinct.
- Every save creates a new revision. Run records retain the exact graph, including canvas coordinates, even for unsaved test drafts.

## Coverage and issues

- Backend coverage: graph validity, execution ordering/branching/failure, revision persistence, archive restore, replay access, and no-side-effect behavior.
- Browser coverage: first-node drag/click placement on an empty canvas, original run to debugger, node movement, typed connection feedback, save/reload, dry-run, and recorded graph.
- Full local validation previously passed for dependencies, Go, Docker build/readiness, Trivy scanning, and 48 Playwright tests. The current rename/delete/canvas pass has successful frontend and Go builds; the browser suite was not rerun.
- Windows cannot run the repository's Unix-socket operator tests or reliably tear down Playwright-managed Go test servers. Linux Go race tests and all browser tests passed using separately managed local fixtures.
- CI for the exact branch HEAD remains the final merge-readiness gate. The branch has not been pushed or merged.
