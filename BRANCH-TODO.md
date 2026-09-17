# Branch TODO — RAM optimization

## Requested changes
- [x] Create a dedicated optimization branch.
- [x] Change user-facing authentication wording from “incorrect password or PIN” to “incorrect password”.
- [x] Remove obsolete PIN-named localization keys and update bundled translations.
- [ ] Reduce Argon2id working memory from 64 MiB to 16 MiB while keeping an OWASP-aligned CPU/memory trade-off.
- [x] Serialize expensive password hashing operations and return temporary hash memory to the OS after use.
- [ ] Preserve verification of existing 64 MiB hashes and migrate them to the new parameters after a successful login.
- [ ] Investigate and reduce RAM growth while navigating, especially image/provider buffering and retained caches.
- [ ] Add/update tests for authentication parameters, legacy migration, wording, and localization contracts.
- [ ] Run the required validation suite and inspect the final branch diff for regressions.

## Findings
- Sign-in’s ~64 MiB jump matches the current Argon2id `m=65536` working set.
- Go may retain freed heap pages after the hash completes, which can leave RSS near its high-water mark unless memory is explicitly scavenged.
- Current image proxy misses buffer each remote image in memory (up to 4 MiB per request), so several image loads during navigation can create substantial transient heap pressure.
- Current OWASP guidance lists Argon2id profiles including 19 MiB / t=2 / p=1 and 12 MiB / t=3 / p=1. The branch will use 16 MiB / t=3 / p=1.
