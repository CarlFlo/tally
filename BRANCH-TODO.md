# Branch TODO — RAM optimization

## Requested changes
- [x] Create a dedicated optimization branch.
- [x] Change user-facing authentication wording from “incorrect password or PIN” to “incorrect password”.
- [x] Remove obsolete PIN-named localization keys and update bundled translations.
- [x] Reduce Argon2id working memory from 64 MiB to 16 MiB while keeping a conservative CPU/memory trade-off.
- [x] Serialize expensive password hashing operations and return temporary hash memory to the OS after use.
- [x] Preserve verification of existing 64 MiB hashes and migrate them to the new parameters after a successful login.
- [x] Investigate and reduce RAM growth while navigating, especially image/provider buffering and retained caches.
- [x] Add/update tests for authentication parameters, legacy migration, wording, localization contracts, and streamed downloads.
- [ ] Run the required validation suite and inspect the final branch diff for regressions.

## Findings
- Sign-in’s ~64 MiB jump matched the previous Argon2id `m=65536` working set. New hashes use 16 MiB with three iterations and one lane; legacy hashes remain verifiable and are upgraded after successful login.
- Go can retain freed heap pages after a memory-hard hash. Authentication now serializes Argon2 work and explicitly returns those temporary heap pages to the OS afterward.
- Image proxy cache misses previously buffered each remote image in memory (up to 4 MiB per request). Cache misses now stream to bounded temporary files, validate the image from disk, and atomically move it into the cache.
- Current OWASP guidance lists Argon2id profiles including 19 MiB / t=2 / p=1 and 12 MiB / t=3 / p=1. The branch uses 16 MiB / t=3 / p=1.
- React Query’s cache was reviewed as part of the navigation investigation. Its normal browser-side cache lifecycle is not a plausible explanation for the server RSS jump observed during authentication; the largest identified server-side contributors were password hashing and image-response buffering.
