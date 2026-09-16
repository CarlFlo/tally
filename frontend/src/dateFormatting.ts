// Reuse ICU formatters across rows, countdown ticks and route mounts. Cache
// configuration only (never dates or profile data), with a hard size bound.
const formatters = new Map<string, Intl.DateTimeFormat>();
const capacity = 32;

export function dateTimeFormatter(
  locale: string,
  options: Intl.DateTimeFormatOptions,
): Intl.DateTimeFormat {
  const key = JSON.stringify([locale, Object.entries(options).sort(([a], [b]) => a.localeCompare(b))]);
  let formatter = formatters.get(key);
  if (!formatter) {
    formatter = new Intl.DateTimeFormat(locale, options);
    if (formatters.size >= capacity) formatters.delete(formatters.keys().next().value!);
    formatters.set(key, formatter);
  }
  return formatter;
}


export function displayLocale(locale?: string | null): string {
  if (!locale || locale === "en") return "en-GB";
  return locale;
}
