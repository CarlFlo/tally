# Localization

Tally localizes the web interface per profile. Translation catalogs are owned by the server and loaded from the persistent data directory.

## Location

Locale files live in:

```text
/config/locales/
```

More generally, this is `<APP_DATA_DIR>/locales/`.

On startup Tally creates the directory when needed and installs the bundled English catalog as `en.json`. English is the canonical translation-key contract and the final fallback.

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
    "save": "Spara",
    "cancel": "Avbryt"
  },
  "calendar": {
    "today": "Idag"
  }
}
```

Metadata fields:

- `locale`: stable locale ID. It must match the filename without `.json`.
- `name`: human-readable name shown in language selectors.
- `direction`: `ltr` or `rtl`.
- `catalogVersion`: positive integer identifying that translation file's revision. It is independent of the Tally application version.

Translation values must be non-empty strings or nested objects containing strings. Arrays, numbers, booleans, empty strings, and empty translation groups are rejected.

## Creating a translation

1. Use `internal/localization/en.json` from the same Tally release as the reference key set.
2. Create a new file in `/config/locales`.
3. Change `_meta.locale`, `_meta.name`, `_meta.direction`, and set a translation catalog version.
4. Translate as many keys as desired.
5. Save the file. Tally reloads it automatically; no server restart is required.

A translation does **not** need to contain every English key. Missing keys fall back to English through i18next. This makes partial translations usable and lets older translations continue working when new UI text is added.

Keep interpolation variables unchanged, including their braces. For example:

```json
{
  "settings": {
    "usedProfiles": "{{used}} of {{max}} profiles used."
  }
}
```

Use i18next plural keys where English does, such as `_one` and `_other`. Other languages may add the plural forms required by their locale.

## Validation and hot reload

Tally watches the locale directory with filesystem events. It does not continuously poll the directory.

When a JSON file is created, changed, renamed, or removed, Tally debounces the filesystem events and reloads the locale registry. Connected browsers receive a live-update hint through the existing SSE connection and refresh their locale index/catalog automatically.

Invalid locale files:

- are logged to the server console with the detailed validation error;
- remain visible in language selectors where possible;
- are disabled so users cannot newly select them;
- include a concise reason in the UI.

If a profile is already configured to use a locale that later becomes missing or invalid, Tally keeps the stored profile preference but temporarily renders the interface in English. When the locale is repaired, the profile automatically returns to it.

The embedded English catalog is always available as the final safety fallback, even if the config copy of `en.json` is damaged while Tally is running.

## English catalog versioning

The bundled English catalog has its own `catalogVersion`.

On startup:

- if `/config/locales/en.json` is missing, Tally installs the bundled copy;
- if the file is invalid or has an older catalog version, Tally replaces it with the bundled copy;
- if its catalog version is equal to or newer than the bundled version, Tally leaves it untouched.

When the canonical English key set is intentionally changed for a release, increment `_meta.catalogVersion` in `internal/localization/en.json`.

Non-English locale files are never overwritten by Tally.

## What is localized

The localization layer is for the web interface: navigation, settings, dialogs, validation copy, frontend-generated notifications, accessibility labels, human-visible dates/numbers, and similar UI text.

The following remain original/English by design:

- TV/provider metadata such as show names, episode names, summaries, genres, networks, and provider status text;
- database values and protocol/internal identifiers;
- server logs and activity/log message payloads;
- raw backend/provider errors that do not have a stable API error code.

Known API errors may expose stable error codes so the frontend can show localized copy while retaining the original server message as a fallback.

## Locale metadata in the browser

When a locale becomes active, Tally updates the document `lang` and `dir` attributes. This keeps browser accessibility behavior correct and prepares the UI for future RTL translations.

Profile locale is authoritative. Browser language detection is intentionally not used, so different Tally profiles and concurrent sessions can use different languages independently.
