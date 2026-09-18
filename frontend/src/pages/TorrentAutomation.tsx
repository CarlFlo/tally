import { useQueryClient } from "@tanstack/react-query";
import { Bot, RotateCcw, Save, ShieldCheck, SlidersHorizontal, Star } from "lucide-react";
import { useEffect, useRef, useState, type CSSProperties, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { api, Busy, ErrorState, useApp, useLocal } from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { TorrentTabs } from "./TorrentTabs";
import { AutomationShowEnrollmentList } from "../AutomationShowEnrollmentList";
import "../torrent-selection.css";

type AutomationConfig = {
  enabled: boolean;
  preferred_quality: "best" | "720p" | "1080p" | "2160p";
  min_seeders: number;
  high_confidence_only: boolean;
  prefer_smaller: boolean;
  release_delay_minutes: number;
  retry_window_hours: number;
  max_candidates: number;
  rules_version: number;
  include_keywords: string;
  exclude_keywords: string;
  allowed_groups: string[];
  preferred_groups: string[];
  allowed_uploaders: string[];
  preferred_uploaders: string[];
  preferred_providers: string[];
  live_min_mb_per_minute: number;
  live_max_mb_per_minute: number;
  animated_min_mb_per_minute: number;
  animated_max_mb_per_minute: number;
  request_policy_version: number;
  discovery_budget: number;
  retry_first_minutes: number;
  retry_second_minutes: number;
  retry_later_minutes: number;
  prioritize_recent: boolean;
};

type SavedAutomation = { data: AutomationConfig; revision: number };
type RuleText = {
  allowed_groups: string;
  preferred_groups: string;
  allowed_uploaders: string;
  preferred_uploaders: string;
  preferred_providers: string;
};

const RULE_DEFAULTS = {
  include_keywords: "",
  exclude_keywords: "cam telesync hardsub dubbed",
  allowed_groups: [] as string[],
  preferred_groups: [] as string[],
  allowed_uploaders: [] as string[],
  preferred_uploaders: [] as string[],
  preferred_providers: [] as string[],
};

function normalizeConfig(data: AutomationConfig): AutomationConfig {
  return { ...data, high_confidence_only: true, rules_version: 1 };
}

function listValue(values: string[]) {
  return values.join(", ");
}

function ruleTextFromConfig(data: AutomationConfig): RuleText {
  return {
    allowed_groups: listValue(data.allowed_groups || []),
    preferred_groups: listValue(data.preferred_groups || []),
    allowed_uploaders: listValue(data.allowed_uploaders || []),
    preferred_uploaders: listValue(data.preferred_uploaders || []),
    preferred_providers: listValue(data.preferred_providers || []),
  };
}

function parseList(value: string) {
  const seen = new Set<string>();
  return value
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter((item) => {
      if (!item) return false;
      const key = item.toLowerCase();
      if (seen.has(key)) return false;
      seen.add(key);
      return true;
    });
}

function appendKeyword(current: string, keyword: string) {
  const words = current.split(/\s+/).filter(Boolean);
  if (words.some((word) => word.toLowerCase() === keyword.toLowerCase())) return current;
  return [...words, keyword].join(" ");
}

function SizeRange({
  label,
  minLabel,
  maxLabel,
  minimum,
  maximum,
  disabled,
  onChange,
}: {
  label: string;
  minLabel: string;
  maxLabel: string;
  minimum: number;
  maximum: number;
  disabled: boolean;
  onChange: (minimum: number, maximum: number) => void;
}) {
  return (
    <div className="size-range-setting">
      <div className="size-range-heading">
        <strong>{label}</strong>
        <span>{minimum}–{maximum} MB/min</span>
      </div>
      <div className="dual-range" style={{ "--range-min": `${(minimum / 500) * 100}%`, "--range-max": `${(maximum / 500) * 100}%` } as CSSProperties}>
        <div className="dual-range-track" />
        <input
          type="range"
          min="1"
          max="500"
          value={minimum}
          disabled={disabled}
          aria-label={minLabel}
          onChange={(event) => onChange(Math.min(Number(event.target.value), maximum - 1), maximum)}
        />
        <input
          type="range"
          min="1"
          max="500"
          value={maximum}
          disabled={disabled}
          aria-label={maxLabel}
          onChange={(event) => onChange(minimum, Math.max(Number(event.target.value), minimum + 1))}
        />
      </div>
    </div>
  );
}

export function TorrentAutomationPage() {
  const { t } = useTranslation();
  const { notify } = useApp();
  const cache = useQueryClient();
  const query = useLocal<SavedAutomation>(
    "editable-settings",
    "/settings/torrent-automation",
    true,
  );
  const previous = useRef<SavedAutomation | null>(null);
  const [data, setData] = useState<AutomationConfig | null>(null);
  const [ruleText, setRuleText] = useState<RuleText | null>(null);
  const [revision, setRevision] = useState(0);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!query.data) return;
    const previousData = previous.current
      ? normalizeConfig(previous.current.data)
      : null;
    const previousRuleText = previousData
      ? ruleTextFromConfig(previousData)
      : null;
    const draftIsClean =
      !data ||
      (!!previousData &&
        JSON.stringify(data) === JSON.stringify(previousData) &&
        JSON.stringify(ruleText) === JSON.stringify(previousRuleText));
    if (draftIsClean) {
      const next = normalizeConfig(query.data.data);
      setData(next);
      setRuleText(ruleTextFromConfig(next));
      setRevision(query.data.revision);
    }
    previous.current = query.data;
  }, [query.data]);

  if (query.error)
    return <ErrorState error={query.error} retry={() => query.refetch()} />;
  if (!data || !ruleText || !query.data) return <Busy />;

  function change(next: Partial<AutomationConfig>) {
    setData((current) => (current ? { ...current, ...next } : current));
  }

  function changeRuleText(next: Partial<RuleText>) {
    setRuleText((current) => (current ? { ...current, ...next } : current));
  }

  function resetRules() {
    change({ ...RULE_DEFAULTS, rules_version: 1 });
    setRuleText({
      allowed_groups: "",
      preferred_groups: "",
      allowed_uploaders: "",
      preferred_uploaders: "",
      preferred_providers: "",
    });
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    const currentData = data;
    const currentRuleText = ruleText;
    if (!currentData || !currentRuleText) return;
    setBusy(true);
    const next: AutomationConfig = {
      ...currentData,
      high_confidence_only: true,
      rules_version: 1,
      allowed_groups: parseList(currentRuleText.allowed_groups),
      preferred_groups: parseList(currentRuleText.preferred_groups),
      allowed_uploaders: parseList(currentRuleText.allowed_uploaders),
      preferred_uploaders: parseList(currentRuleText.preferred_uploaders),
      preferred_providers: parseList(currentRuleText.preferred_providers),
    };
    try {
      const result = await api<{ revision: number }>(
        "/settings/torrent-automation",
        "PUT",
        { data: next, revision },
      );
      setData(next);
      setRuleText(ruleTextFromConfig(next));
      setRevision(result.revision);
      await invalidateResources(cache, ["editable-settings", "settings", "jobs"]);
      notify(
        t("torrentAutomation.saved", {
          defaultValue: "Torrent automation settings saved",
        }),
      );
    } catch (error) {
      notify((error as Error).message, true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="page torrent-automation-page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">
            {t("torrentAutomation.eyebrow", { defaultValue: "STRONG SIGNALS BEFORE IT MOVES" })}
          </span>
          <h1>
            {t("search.title")}<span className="accent">.</span>
          </h1>
          <p>
            {t("torrentAutomation.description", {
              defaultValue:
                "Control how Tally matches, sizes, verifies, and selects releases before handing them to your torrent client.",
            })}
          </p>
        </div>
      </div>
      <TorrentTabs />

      <form onSubmit={save} className="torrent-automation-settings">
        <section className="panel settings-card feature-toggle-setting">
          <label className="toggle-setting">
            <input
              type="checkbox"
              checked={data.enabled}
              disabled={busy}
              onChange={(event) => change({ enabled: event.target.checked })}
            />
            {t("torrentAutomation.enable", {
              defaultValue: "Enable automatic torrent downloads",
            })}
          </label>
          <p className="muted small-text">
            {t("torrentAutomation.enableHelp", {
              defaultValue:
                "This is global for the server. The scheduler can check for releases while this is enabled.",
            })}
          </p>
        </section>

        <AutomationShowEnrollmentList />

        <section className="panel settings-card">
          <h3>
            <Bot size={19} />
            {t("torrentAutomation.selection", { defaultValue: "Release selection" })}
          </h3>
          <p className="muted">
            {t("torrentAutomation.selectionHelp", {
              defaultValue:
                "Quality is a preference between valid releases; it does not increase or reduce confidence.",
            })}
          </p>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.preferredQuality", {
                defaultValue: "Preferred quality",
              })}
              <select
                value={data.preferred_quality}
                disabled={busy}
                onChange={(event) =>
                  change({
                    preferred_quality: event.target.value as AutomationConfig["preferred_quality"],
                  })
                }
              >
                <option value="best">
                  {t("torrentAutomation.bestAvailable", { defaultValue: "Best available" })}
                </option>
                <option value="720p">720p</option>
                <option value="1080p">1080p</option>
                <option value="2160p">2160p</option>
              </select>
            </label>
            <label>
              {t("torrentAutomation.minSeeders", {
                defaultValue: "Minimum seeders",
              })}
              <input
                type="number"
                min="0"
                max="1000000"
                value={data.min_seeders}
                disabled={busy}
                onChange={(event) =>
                  change({ min_seeders: Math.max(0, Number(event.target.value)) })
                }
              />
            </label>
          </div>
          <label className="toggle-setting compact-toggle">
            <input
              type="checkbox"
              checked={data.prefer_smaller}
              disabled={busy}
              onChange={(event) => change({ prefer_smaller: event.target.checked })}
            />
            {t("torrentAutomation.preferSmaller", {
              defaultValue: "Prefer smaller valid releases",
            })}
          </label>
        </section>

        <section className="panel settings-card size-profile-card">
          <h3>
            <SlidersHorizontal size={19} />
            {t("torrentAutomation.sizeProfiles", { defaultValue: "Episode size" })}
          </h3>
          <p className="muted">
            {t("torrentAutomation.sizeProfilesHelp", {
              defaultValue:
                "Filter implausibly small or large releases by MB per minute. Tally uses episode runtime when available, then show runtime; verified torrent contents replace Jackett's reported size when possible.",
            })}
          </p>
          <div className="size-profile-ranges">
            <SizeRange
              label={t("torrentAutomation.liveActionProfile", { defaultValue: "Live-action / normal TV" })}
              minLabel={t("torrentAutomation.liveActionMin", { defaultValue: "Live-action minimum MB per minute" })}
              maxLabel={t("torrentAutomation.liveActionMax", { defaultValue: "Live-action maximum MB per minute" })}
              minimum={data.live_min_mb_per_minute}
              maximum={data.live_max_mb_per_minute}
              disabled={busy}
              onChange={(minimum, maximum) => change({ live_min_mb_per_minute: minimum, live_max_mb_per_minute: maximum })}
            />
            <SizeRange
              label={t("torrentAutomation.animatedProfile", { defaultValue: "Animated / anime" })}
              minLabel={t("torrentAutomation.animatedMin", { defaultValue: "Animated minimum MB per minute" })}
              maxLabel={t("torrentAutomation.animatedMax", { defaultValue: "Animated maximum MB per minute" })}
              minimum={data.animated_min_mb_per_minute}
              maximum={data.animated_max_mb_per_minute}
              disabled={busy}
              onChange={(minimum, maximum) => change({ animated_min_mb_per_minute: minimum, animated_max_mb_per_minute: maximum })}
            />
          </div>
          <p className="muted small-text">
            {t("torrentAutomation.sizeProfilesHint", {
              defaultValue: "Auto-detected animated shows use the animated range. You can override a show's media type from its Show actions menu.",
            })}
          </p>
        </section>

        <section className="panel settings-card automation-rule-card">
          <div className="automation-rule-heading">
            <div>
              <h3>
                <SlidersHorizontal size={19} />
                {t("torrentAutomation.filters", { defaultValue: "Release filters" })}
              </h3>
              <p className="muted">
                {t("torrentAutomation.filtersHelp", {
                  defaultValue:
                    "Include terms must all be present. Any excluded term removes the candidate before torrent inspection.",
                })}
              </p>
            </div>
            <button className="button ghost small" type="button" disabled={busy} onClick={resetRules}>
              <RotateCcw size={15} />
              {t("torrentAutomation.resetRules", { defaultValue: "Reset filters" })}
            </button>
          </div>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.includeKeywords", { defaultValue: "Include keywords" })}
              <input
                value={data.include_keywords}
                disabled={busy}
                placeholder={t("torrentAutomation.includePlaceholder", { defaultValue: "proper repack" })}
                onChange={(event) => change({ include_keywords: event.target.value })}
              />
            </label>
            <label>
              {t("torrentAutomation.excludeKeywords", { defaultValue: "Exclude keywords" })}
              <input
                value={data.exclude_keywords}
                disabled={busy}
                placeholder={t("torrentAutomation.excludePlaceholder", { defaultValue: "cam telesync hardsub dubbed" })}
                onChange={(event) => change({ exclude_keywords: event.target.value })}
              />
            </label>
          </div>
          <div className="automation-suggestion-row">
            <span className="muted small-text">
              {t("torrentAutomation.quickAdd", { defaultValue: "Quick add" })}
            </span>
            {["proper", "repack"].map((word) => (
              <button
                key={`include-${word}`}
                type="button"
                className="rule-chip positive"
                onClick={() => change({ include_keywords: appendKeyword(data.include_keywords, word) })}
              >
                + {word}
              </button>
            ))}
            {["cam", "telesync", "hardsub", "dubbed"].map((word) => (
              <button
                key={`exclude-${word}`}
                type="button"
                className="rule-chip negative"
                onClick={() => change({ exclude_keywords: appendKeyword(data.exclude_keywords, word) })}
              >
                − {word}
              </button>
            ))}
          </div>
        </section>

        <section className="panel settings-card automation-rule-card">
          <h3>
            <Star size={19} />
            {t("torrentAutomation.releaseGroups", { defaultValue: "Release groups" })}
          </h3>
          <p className="muted">
            {t("torrentAutomation.releaseGroupsHelp", {
              defaultValue:
                "An allowlist is strict: when it contains entries, releases from other or unknown groups are skipped. Preferred groups rank ahead of otherwise equivalent candidates.",
            })}
          </p>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.allowedGroups", { defaultValue: "Allowed release groups" })}
              <textarea
                rows={3}
                disabled={busy}
                value={ruleText.allowed_groups}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => changeRuleText({ allowed_groups: event.target.value })}
              />
            </label>
            <label>
              {t("torrentAutomation.preferredGroups", { defaultValue: "Preferred release groups" })}
              <textarea
                rows={3}
                disabled={busy}
                value={ruleText.preferred_groups}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => changeRuleText({ preferred_groups: event.target.value })}
              />
            </label>
          </div>
        </section>

        <section className="panel settings-card automation-rule-card">
          <h3>
            <ShieldCheck size={19} />
            {t("torrentAutomation.sourceTrust", { defaultValue: "Source trust" })}
          </h3>
          <p className="muted">
            {t("torrentAutomation.sourceTrustHelp", {
              defaultValue:
                "Uploader metadata is optional in Jackett. A configured uploader allowlist skips candidates whose uploader is missing or not allowed. Preferred uploaders and providers influence ranking only after identity checks pass.",
            })}
          </p>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.allowedUploaders", { defaultValue: "Allowed uploaders" })}
              <textarea
                rows={3}
                disabled={busy}
                value={ruleText.allowed_uploaders}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => changeRuleText({ allowed_uploaders: event.target.value })}
              />
            </label>
            <label>
              {t("torrentAutomation.preferredUploaders", { defaultValue: "Preferred uploaders" })}
              <textarea
                rows={3}
                disabled={busy}
                value={ruleText.preferred_uploaders}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => changeRuleText({ preferred_uploaders: event.target.value })}
              />
            </label>
          </div>
          <label>
            {t("torrentAutomation.preferredProviders", { defaultValue: "Preferred Jackett providers / indexers" })}
            <textarea
              rows={3}
              disabled={busy}
              value={ruleText.preferred_providers}
              placeholder={t("torrentAutomation.providerPlaceholder", { defaultValue: "Use the provider names shown in search results" })}
              onChange={(event) => changeRuleText({ preferred_providers: event.target.value })}
            />
          </label>
        </section>

        <section className="panel settings-card">
          <h3>
            <Bot size={19} />
            {t("torrentAutomation.searchPacing", { defaultValue: "Search pacing" })}
          </h3>
          <p className="muted">
            {t("torrentAutomation.searchPacingHelp", {
              defaultValue:
                "Limit how much external discovery work one scheduler run can do. Deferred episodes stay queued for later runs instead of creating a burst of Jackett requests.",
            })}
          </p>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.discoveryBudget", {
                defaultValue: "Episode searches per run",
              })}
              <input
                type="number"
                min="1"
                max="25"
                value={data.discovery_budget}
                disabled={busy}
                onChange={(event) =>
                  change({ discovery_budget: Math.max(1, Number(event.target.value)) })
                }
              />
            </label>
            <label>
              {t("torrentAutomation.maxCandidates", {
                defaultValue: "Torrent candidates inspected per episode",
              })}
              <input
                type="number"
                min="1"
                max="20"
                value={data.max_candidates}
                disabled={busy}
                onChange={(event) =>
                  change({ max_candidates: Math.max(1, Number(event.target.value)) })
                }
              />
            </label>
            <label className="toggle-setting compact-toggle">
              <input
                type="checkbox"
                checked={data.prioritize_recent}
                disabled={busy}
                onChange={(event) => change({ prioritize_recent: event.target.checked })}
              />
              {t("torrentAutomation.prioritizeRecent", {
                defaultValue: "Prioritize newly aired episodes",
              })}
            </label>
          </div>
          <p className="muted small-text">
            {t("torrentAutomation.prioritizeRecentHelp", {
              defaultValue:
                "When several episodes are ready, newer releases are searched before older backlog items.",
            })}
          </p>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.retryFirst", {
                defaultValue: "First retry delay (minutes)",
              })}
              <input
                type="number"
                min="5"
                max="1440"
                value={data.retry_first_minutes}
                disabled={busy}
                onChange={(event) =>
                  change({ retry_first_minutes: Math.max(5, Number(event.target.value)) })
                }
              />
            </label>
            <label>
              {t("torrentAutomation.retrySecond", {
                defaultValue: "Second retry delay (minutes)",
              })}
              <input
                type="number"
                min="5"
                max="1440"
                value={data.retry_second_minutes}
                disabled={busy}
                onChange={(event) =>
                  change({ retry_second_minutes: Math.max(5, Number(event.target.value)) })
                }
              />
            </label>
            <label>
              {t("torrentAutomation.retryLater", {
                defaultValue: "Later retry delay (minutes)",
              })}
              <input
                type="number"
                min="5"
                max="1440"
                value={data.retry_later_minutes}
                disabled={busy}
                onChange={(event) =>
                  change({ retry_later_minutes: Math.max(5, Number(event.target.value)) })
                }
              />
            </label>
          </div>
          <p className="muted small-text">
            {t("torrentAutomation.retryPacingHelp", {
              defaultValue:
                "These delays apply after unsuccessful or failed automated searches. Tally does not immediately retry Jackett inside the same background attempt.",
            })}
          </p>
        </section>

        <section className="panel settings-card">
          <h3>
            <ShieldCheck size={19} />
            {t("torrentAutomation.verification", { defaultValue: "Verification" })}
          </h3>
          <p className="muted">
            {t("torrentAutomation.highVerifiedOnly", {
              defaultValue:
                "Automatic downloads require High confidence. Tally prefers a retrievable .torrent and verifies its payload first; when only a magnet is usable, it may fall back to metadata-only selection after all other rules pass.",
            })}
          </p>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.releaseDelay", {
                defaultValue: "Minimum release age (minutes)",
              })}
              <input
                type="number"
                min="0"
                max="1440"
                value={data.release_delay_minutes}
                disabled={busy}
                onChange={(event) =>
                  change({ release_delay_minutes: Math.max(0, Number(event.target.value)) })
                }
              />
            </label>
            <label>
              {t("torrentAutomation.retryWindow", {
                defaultValue: "Retry window (hours)",
              })}
              <input
                type="number"
                min="1"
                max="168"
                value={data.retry_window_hours}
                disabled={busy}
                onChange={(event) =>
                  change({ retry_window_hours: Math.max(1, Number(event.target.value)) })
                }
              />
            </label>
          </div>
          <div className="setup-notice automation-magnet-note">
            <ShieldCheck size={19} />
            <div>
              <strong>
                {t("torrentAutomation.magnetsManual", {
                  defaultValue: "Magnets are a fallback",
                })}
              </strong>
              <p>
                {t("torrentAutomation.magnetsManualHelp", {
                  defaultValue:
                    "Tally asks Jackett for an inspectable .torrent whenever possible. A magnet can be selected automatically only when the release is High confidence and passes identity, trust, history, seed, and MB/min checks. It starts Unverified, then Tally checks qBittorrent's resolved file list on later automation runs and removes unsafe or mismatched payloads.",
                })}
              </p>
            </div>
          </div>
        </section>

        <div className="settings-actions">
          <button className="button primary" type="submit" disabled={busy}>
            {busy ? <Busy /> : <Save size={17} />}
            {t("common.save")}
          </button>
        </div>
      </form>
    </div>
  );
}
