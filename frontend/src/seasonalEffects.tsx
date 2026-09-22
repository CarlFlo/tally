import { useEffect, useState, useSyncExternalStore, type ReactNode } from "react";

type SeasonalDateRule = {
  month: number;
  startDay: number;
  endDay?: number;
};

type SeasonalEffectDefinition = {
  id: string;
  labelKey: string;
  dates: readonly SeasonalDateRule[];
  renderOverlay: () => ReactNode;
};

export const SEASONAL_EFFECTS = [
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
      <svg viewBox="0 0 28 24" focusable="false">
        <path d="M5 15.5C8 7.6 13.2 3.2 22.4 3.1c-3 2.4-4.7 5.5-5.1 9.4Z" fill="#D64045" />
        <path d="M4.2 14.4h14.6a2.1 2.1 0 0 1 2.1 2.1v1.7H2.1v-1.7a2.1 2.1 0 0 1 2.1-2.1Z" fill="#FFF9F2" />
        <circle cx="23.2" cy="3.5" r="2.8" fill="#FFF9F2" />
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

function matchesDate(effect: SeasonalEffectDefinition, date: Date) {
  const month = date.getMonth() + 1;
  const day = date.getDate();
  return effect.dates.some(
    (rule) =>
      rule.month === month &&
      day >= rule.startDay &&
      day <= (rule.endDay ?? rule.startDay),
  );
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
