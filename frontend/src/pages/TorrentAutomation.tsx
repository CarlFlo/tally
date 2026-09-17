import { useQueryClient } from "@tanstack/react-query";
import { Bot, RotateCcw, Save, ShieldCheck, SlidersHorizontal, Star } from "lucide-react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { api, Busy, ErrorState, useApp, useLocal } from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { TorrentTabs } from "./TorrentTabs";
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
};

type SavedAutomation = { data: AutomationConfig; revision: number };

const RULE_DEFAULTS = {
  include_keywords: "",
  exclude_keywords: "cam telesync hardsub dubbed",
  allowed_groups: [] as string[],
  preferred_groups: [] as string[],
  allowed_uploaders: [] as string[],
  preferred_uploaders: [] as string[],
  preferred_providers: [] as string[],
};

function listValue(values: string[]) {
  return values.join(", ");
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
  const [revision, setRevision] = useState(0);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!query.data) return;
    if (!data || JSON.stringify(data) === JSON.stringify(previous.current?.data)) {
      setData({ ...query.data.data, high_confidence_only: true, rules_version: 1 });
      setRevision(query.data.revision);
    }
    previous.current = query.data;
  }, [data, query.data]);

  if (query.error)
    return <ErrorState error={query.error} retry={() => query.refetch()} />;
  if (!data || !query.data) return <Busy />;

  function change(next: Partial<AutomationConfig>) {
    setData((current) => (current ? { ...current, ...next } : current));
  }

  function resetRules() {
    change({ ...RULE_DEFAULTS, rules_version: 1 });
  }

  async function save(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      const result = await api<{ revision: number }>(
        "/settings/torrent-automation",
        "PUT",
        {
          data: { ...data, high_confidence_only: true, rules_version: 1 },
          revision,
        },
      );
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
            {t("torrentAutomation.eyebrow", { defaultValue: "VERIFIED BEFORE IT MOVES" })}
          </span>
          <h1>
            {t("search.title")}<span className="accent">.</span>
          </h1>
          <p>
            {t("torrentAutomation.description", {
              defaultValue:
                "Control how Tally chooses and verifies releases before handing them to your torrent client.",
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
                value={listValue(data.allowed_groups)}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => change({ allowed_groups: parseList(event.target.value) })}
              />
            </label>
            <label>
              {t("torrentAutomation.preferredGroups", { defaultValue: "Preferred release groups" })}
              <textarea
                rows={3}
                disabled={busy}
                value={listValue(data.preferred_groups)}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => change({ preferred_groups: parseList(event.target.value) })}
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
                value={listValue(data.allowed_uploaders)}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => change({ allowed_uploaders: parseList(event.target.value) })}
              />
            </label>
            <label>
              {t("torrentAutomation.preferredUploaders", { defaultValue: "Preferred uploaders" })}
              <textarea
                rows={3}
                disabled={busy}
                value={listValue(data.preferred_uploaders)}
                placeholder={t("torrentAutomation.listPlaceholder", { defaultValue: "One per line or comma separated" })}
                onChange={(event) => change({ preferred_uploaders: parseList(event.target.value) })}
              />
            </label>
          </div>
          <label>
            {t("torrentAutomation.preferredProviders", { defaultValue: "Preferred Jackett providers / indexers" })}
            <textarea
              rows={3}
              disabled={busy}
              value={listValue(data.preferred_providers)}
              placeholder={t("torrentAutomation.providerPlaceholder", { defaultValue: "Use the provider names shown in search results" })}
              onChange={(event) => change({ preferred_providers: parseList(event.target.value) })}
            />
          </label>
        </section>

        <section className="panel settings-card">
          <h3>
            <ShieldCheck size={19} />
            {t("torrentAutomation.verification", { defaultValue: "Verification" })}
          </h3>
          <p className="muted">
            {t("torrentAutomation.highVerifiedOnly", {
              defaultValue:
                "Automatic downloads require High confidence and a verified .torrent payload. Medium, Low, Rejected, and unverified candidates are never downloaded automatically.",
            })}
          </p>
          <div className="settings-grid two-fields">
            <label>
              {t("torrentAutomation.releaseDelay", {
                defaultValue: "Release delay (minutes)",
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
                  defaultValue: "Magnet-only releases stay manual",
                })}
              </strong>
              <p>
                {t("torrentAutomation.magnetsManualHelp", {
                  defaultValue:
                    "A magnet does not expose its file list before metadata exchange, so Tally will not choose it for an automatic download.",
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
