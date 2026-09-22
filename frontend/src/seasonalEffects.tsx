import { useEffect, useState, useSyncExternalStore, type ReactNode } from "react";

type SeasonalDateRule = {
  month: number;
  startDay: number;
  endDay?: number;
};

type SeasonalDateRange = {
  start: Date;
  end?: Date;
};

type SeasonalEffectDefinition = {
  id: string;
  labelKey: string;
  dates?: readonly SeasonalDateRule[];
  getDateRanges?: (year: number) => readonly SeasonalDateRange[];
  renderOverlay: () => ReactNode;
};

function easterSunday(year: number): Date {
  const a = year % 19;
  const b = Math.floor(year / 100);
  const c = year % 100;
  const d = Math.floor(b / 4);
  const e = b % 4;
  const f = Math.floor((b + 8) / 25);
  const g = Math.floor((b - f + 1) / 3);
  const h = (19 * a + b - d - g + 15) % 30;
  const i = Math.floor(c / 4);
  const k = c % 4;
  const l = (32 + 2 * e + 2 * i - h - k) % 7;
  const m = Math.floor((a + 11 * h + 22 * l) / 451);
  return new Date(year, Math.floor((h + l - 7 * m + 114) / 31) - 1, ((h + l - 7 * m + 114) % 31) + 1);
}

function swedishMidsummer(year: number): SeasonalDateRange {
  const eve = new Date(year, 5, 19);
  eve.setDate(eve.getDate() + ((5 - eve.getDay() + 7) % 7));
  const day = new Date(eve);
  day.setDate(day.getDate() + 1);
  return { start: eve, end: day };
}

export const SEASONAL_EFFECTS = [
  {
    id: "valentines-day",
    labelKey: "debug.seasonalEffects.valentinesDay",
    dates: [{ month: 2, startDay: 14 }],
    renderOverlay: () => (
      <svg viewBox="0 0 24 24" focusable="false">
        <path d="M12 20.2 4.1 12.8A5 5 0 0 1 11.2 5.7L12 6.5l.8-.8a5 5 0 0 1 7.1 7.1Z" fill="#E64B6B" />
        <path d="M7.1 8.2a3 3 0 0 1 3.2-.6" fill="none" opacity=".55" stroke="#FFF4F6" strokeLinecap="round" strokeWidth="1.4" />
      </svg>
    ),
  },
  {
    id: "april-fools-day",
    labelKey: "debug.seasonalEffects.aprilFoolsDay",
    dates: [{ month: 4, startDay: 1 }],
    renderOverlay: () => (
      <svg viewBox="0 0 26 25" focusable="false">
        <path d="m4 17.6 8.5-14 8.7 14Z" fill="#5C82E6" />
        <path d="m7 12.7 4.6-7.6 2.3 4Z" fill="#F4C84D" />
        <path d="m11.7 5.1 3.4 5.8 2.6-4.2" fill="#E15F82" />
        <path d="M3.2 17.2h18.9c1.3 0 2.3 1 2.3 2.3v1.1H2v-1.1c0-1.3 1-2.3 2.3-2.3Z" fill="#FFF7E9" />
        <circle cx="21.5" cy="5.3" r="1.7" fill="#E15F82" />
      </svg>
    ),
  },
  {
    id: "easter",
    labelKey: "debug.seasonalEffects.easter",
    getDateRanges: (year) => {
      const sunday = easterSunday(year);
      const friday = new Date(sunday);
      friday.setDate(friday.getDate() - 2);
      const monday = new Date(sunday);
      monday.setDate(monday.getDate() + 1);
      return [{ start: friday, end: monday }];
    },
    renderOverlay: () => (
      <svg viewBox="0 0 22 25" focusable="false">
        <path d="M11 2.2c4.7 0 7.7 6.4 7.7 12.5 0 5.1-2.8 8.1-7.7 8.1s-7.7-3-7.7-8.1C3.3 8.6 6.3 2.2 11 2.2Z" fill="#81C7D9" />
        <path d="M5.2 11.3c3.7 1.9 7.9 1.9 11.6 0M4.3 16.1c4.3 2.1 9.1 2.1 13.4 0" fill="none" stroke="#FFF8E7" strokeLinecap="round" strokeWidth="2" />
        <path d="M7.5 4.9c1.2 1 2.5 1.3 3.8 1.3" fill="none" opacity=".6" stroke="#FFF8E7" strokeLinecap="round" strokeWidth="1.4" />
      </svg>
    ),
  },
  {
    id: "halloween",
    labelKey: "debug.seasonalEffects.halloween",
    dates: [{ month: 10, startDay: 31 }],
    renderOverlay: () => (
      <svg viewBox="0 0 24 24" focusable="false">
        <path d="M11 5.2c.1-1.4.7-2.4 1.9-3.2" fill="none" stroke="#5B7C43" strokeLinecap="round" strokeWidth="2.2" />
        <ellipse cx="12" cy="13.1" rx="9.1" ry="7.7" fill="#F97316" />
        <path d="M8 10.5 10.1 12 7.6 12.6ZM16 10.5 13.9 12l2.5.6ZM8.4 15.5c2.2 1.9 5 1.9 7.2 0" fill="none" stroke="#2B1A0E" strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" />
        <path d="M12 5.7c-1.5 1.6-2 4.1-2 7.3s.5 5.5 2 7.4M12 5.7c1.5 1.6 2 4.1 2 7.3s-.5 5.5-2 7.4" fill="none" opacity=".18" stroke="#7C2D12" strokeWidth="1" />
      </svg>
    ),
  },
  {
    id: "christmas",
    labelKey: "debug.seasonalEffects.christmas",
    dates: [{ month: 12, startDay: 24, endDay: 26 }],
    renderOverlay: () => (
      <svg viewBox="0 0 32 26" focusable="false">
        <path d="M5.1 16.9C7.1 8.6 13.9 3.1 24 3c-3.2 2.5-5.2 6.5-5.1 11-4.3-1.3-9.3-.4-13.8 2.9Z" fill="#D64045" />
        <path d="M4.6 15.1c5.6-1.7 12.4-1.6 18.3.2 1.1.4 1.8 1.5 1.5 2.7l-.4 2H2.7l-.5-1.9c-.3-1.3.7-2.6 2.4-3Z" fill="#FFF9F2" />
        <path d="M4.7 18.8c5.1-1.1 11.1-1.1 17.2.2" fill="none" opacity=".42" stroke="#D8CABE" strokeLinecap="round" strokeWidth="1.2" />
        <circle cx="25.4" cy="3.5" r="3.1" fill="#FFF9F2" />
      </svg>
    ),
  },
  {
    id: "new-year",
    labelKey: "debug.seasonalEffects.newYear",
    dates: [
      { month: 12, startDay: 31 },
      { month: 1, startDay: 1 },
    ],
    renderOverlay: () => (
      <svg viewBox="0 0 24 24" focusable="false">
        <path d="m12 1.8 1.4 5.4 4.8-2.8-2.8 4.8 5.4 1.4-5.4 1.4 2.8 4.8-4.8-2.8-1.4 5.4-1.4-5.4-4.8 2.8 2.8-4.8-5.4-1.4 5.4-1.4-2.8-4.8 4.8 2.8Z" fill="#F5C451" />
        <circle cx="19.6" cy="4.3" r="1.4" fill="#F08A5D" />
        <circle cx="4.2" cy="18.8" r="1.2" fill="#8BD3DD" />
      </svg>
    ),
  },
  {
    id: "sweden-national-day",
    labelKey: "debug.seasonalEffects.swedenNationalDay",
    dates: [{ month: 6, startDay: 6 }],
    renderOverlay: () => (
      <svg viewBox="0 0 28 20" focusable="false">
        <rect width="28" height="20" rx="3" fill="#006AA7" />
        <path d="M9 0h4v20H9zM0 8h28v4H0z" fill="#FECC00" />
      </svg>
    ),
  },
  {
    id: "swedish-midsummer",
    labelKey: "debug.seasonalEffects.swedishMidsummer",
    getDateRanges: (year) => [swedishMidsummer(year)],
    renderOverlay: () => (
      <svg viewBox="0 0 28 23" focusable="false">
        <path d="M3.5 16.6C8.4 20.2 19.6 20.2 24.5 16.6" fill="none" stroke="#5C934B" strokeLinecap="round" strokeWidth="3.2" />
        <g fill="#F5E9FF" stroke="#C876A5" strokeWidth=".8">
          <path d="m7 14.3-1.8-2.1 2.8.6.8-2.7 1 2.7 2.7-.5-1.8 2.1Z" />
          <path d="m14 16-1.8-2.1 2.8.6.8-2.7 1 2.7 2.7-.5-1.8 2.1Z" />
          <path d="m21 14.3-1.8-2.1 2.8.6.8-2.7 1 2.7 2.7-.5-1.8 2.1Z" />
        </g>
        <circle cx="7.9" cy="13.1" r="1.3" fill="#F4C84D" />
        <circle cx="14.9" cy="14.8" r="1.3" fill="#F4C84D" />
        <circle cx="21.9" cy="13.1" r="1.3" fill="#F4C84D" />
      </svg>
    ),
  },
  {
    id: "st-lucia-day",
    labelKey: "debug.seasonalEffects.stLuciaDay",
    dates: [{ month: 12, startDay: 13 }],
    renderOverlay: () => (
      <svg viewBox="0 0 27 27" focusable="false">
        <path d="M5.2 17.2c3-2.8 13.6-2.8 16.6 0" fill="none" stroke="#4E8A47" strokeLinecap="round" strokeWidth="2.8" />
        <path d="M9.2 15.5V7.7M13.5 14.8V5M17.8 15.5V7.7" stroke="#F5C451" strokeLinecap="round" strokeWidth="1.7" />
        <path d="M7.8 8.8c0-1 .6-1.8 1.4-1.8s1.4.8 1.4 1.8c0 .8-.5 1.4-1.4 1.4s-1.4-.6-1.4-1.4ZM12.1 6.1c0-1 .6-1.8 1.4-1.8s1.4.8 1.4 1.8c0 .8-.5 1.4-1.4 1.4s-1.4-.6-1.4-1.4ZM16.4 8.8c0-1 .6-1.8 1.4-1.8s1.4.8 1.4 1.8c0 .8-.5 1.4-1.4 1.4s-1.4-.6-1.4-1.4Z" fill="#FFF4A8" />
      </svg>
    ),
  },
  {
    id: "ukraine-independence-day",
    labelKey: "debug.seasonalEffects.ukraineIndependenceDay",
    dates: [{ month: 8, startDay: 24 }],
    renderOverlay: () => (
      <svg viewBox="0 0 28 20" focusable="false">
        <path d="M3 0h22a3 3 0 0 1 3 3v7H0V3a3 3 0 0 1 3-3Z" fill="#0057B7" />
        <path d="M0 10h28v7a3 3 0 0 1-3 3H3a3 3 0 0 1-3-3Z" fill="#FFD700" />
      </svg>
    ),
  },
] as const satisfies readonly SeasonalEffectDefinition[];

export type SeasonalEffect = (typeof SEASONAL_EFFECTS)[number];
export type SeasonalEffectId = SeasonalEffect["id"];

export type SeasonalEffectOverride = {
  enabled: boolean;
  effectId: SeasonalEffectId;
};

const OVERRIDE_ENABLED_KEY = "tally-seasonal-effect-override";
const OVERRIDE_EFFECT_KEY = "tally-seasonal-effect-id";
const DEFAULT_EFFECT_ID = SEASONAL_EFFECTS[0].id;

function defaultOverride(): SeasonalEffectOverride {
  return { enabled: false, effectId: DEFAULT_EFFECT_ID };
}

function loadSeasonalEffectOverride(): SeasonalEffectOverride {
  try {
    const storedEffectId = sessionStorage.getItem(OVERRIDE_EFFECT_KEY);
    return {
      enabled: sessionStorage.getItem(OVERRIDE_ENABLED_KEY) === "true",
      effectId: isSeasonalEffectId(storedEffectId)
        ? storedEffectId
        : DEFAULT_EFFECT_ID,
    };
  } catch {
    return defaultOverride();
  }
}

let volatileOverride = loadSeasonalEffectOverride();
const overrideSubscribers = new Set<() => void>();

function isSeasonalEffectId(value: string | null): value is SeasonalEffectId {
  return SEASONAL_EFFECTS.some((effect) => effect.id === value);
}

function dateRanges(effect: SeasonalEffectDefinition, year: number) {
  if (effect.getDateRanges) return effect.getDateRanges(year);
  return (effect.dates ?? []).map((rule) => ({
    start: new Date(year, rule.month - 1, rule.startDay),
    end: new Date(year, rule.month - 1, rule.endDay ?? rule.startDay),
  }));
}

function matchesDate(effect: SeasonalEffectDefinition, date: Date) {
  const current = new Date(date.getFullYear(), date.getMonth(), date.getDate());
  return dateRanges(effect, date.getFullYear()).some(({ start, end = start }) =>
    current >= start && current <= end,
  );
}

export function formatSeasonalEffectDates(
  effect: SeasonalEffectDefinition,
  locale: string,
  year = new Date().getFullYear(),
): string {
  const format = new Intl.DateTimeFormat(locale, {
    month: "long",
    day: "numeric",
  });
  const ranges = dateRanges(effect, year).map(({ start, end }) => {
    if (!end || start.getTime() === end.getTime()) return format.format(start);
    return format.formatRange(start, end);
  });

  return new Intl.ListFormat(locale, {
    style: "long",
    type: "conjunction",
  }).format(ranges);
}

export function resolveSeasonalEffect(
  date: Date,
  override?: SeasonalEffectOverride | null,
): SeasonalEffect | null {
  if (override?.enabled) {
    return (
      SEASONAL_EFFECTS.find((effect) => effect.id === override.effectId) ?? null
    );
  }
  return SEASONAL_EFFECTS.find((effect) => matchesDate(effect, date)) ?? null;
}

export function readSeasonalEffectOverride(): SeasonalEffectOverride {
  return volatileOverride;
}

export function writeSeasonalEffectOverride(
  override: SeasonalEffectOverride,
): void {
  volatileOverride = override;
  try {
    sessionStorage.setItem(OVERRIDE_ENABLED_KEY, String(override.enabled));
    sessionStorage.setItem(OVERRIDE_EFFECT_KEY, override.effectId);
  } catch {
    // The in-memory value still keeps the preview usable for this document.
  }
  overrideSubscribers.forEach((notify) => notify());
}

function subscribeToSeasonalEffectOverride(notify: () => void) {
  overrideSubscribers.add(notify);
  return () => overrideSubscribers.delete(notify);
}

export function useSeasonalEffectOverride() {
  const override = useSyncExternalStore(
    subscribeToSeasonalEffectOverride,
    readSeasonalEffectOverride,
    readSeasonalEffectOverride,
  );

  return [override, writeSeasonalEffectOverride] as const;
}

function useLocalDateNow() {
  const [now, setNow] = useState(Date.now);

  useEffect(() => {
    const current = new Date();
    const nextDay = new Date(current);
    nextDay.setHours(24, 0, 0, 0);
    const delay = Math.max(1, nextDay.getTime() - current.getTime() + 50);
    const timer = window.setTimeout(() => setNow(Date.now()), delay);
    return () => window.clearTimeout(timer);
  }, [now]);

  return now;
}

export function useActiveSeasonalEffect(): SeasonalEffect | null {
  const [override] = useSeasonalEffectOverride();
  const now = useLocalDateNow();
  return resolveSeasonalEffect(new Date(now), override);
}
