# Branch TODO

- [ ] Make Docker-exec CLI commands run as the configured Tally UID/GID without weakening /config permissions.
- [ ] Add a private local operator channel so reset-password and restore can be executed by the running Tally process.
- [ ] Reuse live restore behavior, settings reconciliation, event publication, and scheduler wake-up for CLI restore.
- [ ] Preserve offline fallback behavior when the server is stopped.
- [ ] Validate CLI arguments before locks or expensive startup work; add help behavior if practical.
- [ ] Add focused tests for online operator commands and existing offline behavior.
- [ ] Run full validation and remove this temporary file.
