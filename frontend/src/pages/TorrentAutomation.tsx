import { useQueryClient } from "@tanstack/react-query";
import { Bot, Save, ShieldCheck } from "lucide-react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { api, Busy, ErrorState, useApp, useLocal } from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { TorrentTabs } from "./TorrentTabs";

type AutomationConfig = {
  enabled: boolean;
  preferred_quality: "best" | "720p" | "1080p" | "2160p";
  min_seeders: number;
  high_confidence_only: boolean;
  prefer_smaller: boolean;
  release_delay_minutes: number;
  retry_window_hours: number;
  max_candidates: number;
};

type SavedAutomation = { data: AutomationConfig; revision: number };

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
      setData({ ...query.data.data, high_confidence_only: true });
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

  async function save(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      const result = await api<{ revision: number }>(
        "/settings/torrent-automation",
        "PUT",
        {
          data: { ...data, high_confidence_only: true },
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
