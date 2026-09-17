# Localization

Tally localizes the web interface per profile. Locale catalogs are loaded from the persistent application data directory.

## Location and bundled locales

Locale files live in `<APP_DATA_DIR>/locales/` (normally `/config/locales/`). Tally creates the directory when needed.

Bundled catalogs currently include:

- `en.json` — canonical key contract and final fallback
- `uk.json` — bundled Ukrainian translation

Bundled filenames are managed by Tally. On startup, Tally reconciles each bundled file with the embedded catalog:

- a missing, invalid, older, or same-version-modified file is replaced atomically;
- an exact current copy is left untouched, avoiding unnecessary disk writes;
- a valid file with a newer `catalogVersion` is preserved so temporarily running an older Tally release does not destroy a newer catalog.

Do not customize a bundled filename. Custom translations must use their own unique locale filename, such as `sv.json`; Tally does not overwrite those files.

## File format

Each locale is one UTF-8 JSON file named after its locale code, for example `sv.json`, `de.json`, or `pt-BR.json`.

```json
{
  "_meta": {
    "locale": "sv",
    "name": "Svenska",
    "direction": "ltr",
    "catalogVersion": 1
  },
  "common": {
    "save": "Spara"
  },
  "calendar": {
    "today": "Idag"
  }
}
```

Metadata:

- `locale` must match the filename without `.json`.
- `name` is the human-readable language name shown in selectors.
- `direction` is `ltr` or `rtl`.
- `catalogVersion` is a positive translation-catalog revision, independent of the Tally application version.

Translation values must be non-empty strings or nested objects containing strings. Arrays, numbers, booleans, empty strings, and empty groups are rejected.

## Adding or editing a locale

Use `internal/localization/en.json` from the same release as the reference key set. A translation may be partial; missing keys fall back to English.

Keep interpolation variables unchanged, including braces:

```json
{
  "settings": {
    "usedProfiles": "{{used}} of {{max}} profiles used."
  }
}
```

Use i18next plural keys where the canonical catalog does, and add any additional plural forms required by the target language.

For every meaningful change to a bundled catalog — translated text, keys, placeholders, or metadata — increment that file's `_meta.catalogVersion`. Bundled catalog versions must only move forward; never reuse or decrease a version after changing its contents.

Tally watches the locale directory with filesystem events. Valid created/changed/renamed/removed files are reloaded and connected browsers are prompted through the existing live-update channel. There is no background polling fallback for unusual network filesystems. Runtime edits to bundled filenames can therefore appear temporarily, but startup reconciliation restores the managed copy unless the on-disk catalog is from a newer version.

Invalid custom locale files are logged, remain visible where practical, and are disabled for new selection with a concise UI reason. If a profile already references a custom locale that later becomes missing or invalid, the saved preference remains intact while the interface falls back to English. Repairing the file automatically restores that locale.

## Profile behavior

The profile's saved locale is authoritative. Browser language detection is intentionally not used, so different Tally profiles can use different languages on the same deployment.

Changing the language selector edits only the profile draft. The active interface locale changes after **Save profile** succeeds. Leaving the page without saving does not apply the draft language.

When a locale becomes active, Tally updates the document `lang` and `dir` attributes.

Feedback produced by an action that changes locale must resolve against the newly active locale. Do not capture a translated string before the save and display it afterward; keep semantic data such as a translation key and translate at render time when the UI object can survive a locale change.

## What is localized

Localize navigation, settings, dialogs, validation copy, frontend notifications, accessibility labels, and user-visible formatting.

The following remain original/English by design unless a stable application-level mapping exists:

- provider metadata such as show names, episode names, summaries, genres, and networks;
- database values and protocol/internal identifiers;
- server activity/log payloads;
- unstable raw provider/backend error text.

Known API errors may expose stable error codes so the frontend can show localized copy while retaining the original server message as a fallback.

## English key contract

English is the canonical key set and final runtime fallback. When its keys or copy change meaningfully, increment `_meta.catalogVersion` in `internal/localization/en.json`. Bundled translations updated to match those changes must increment their own catalog version as well.

English remains available from the embedded catalog even if the persistent `en.json` becomes damaged while Tally is running.

For broader state/localization lessons, see `LESSONS.md`.
