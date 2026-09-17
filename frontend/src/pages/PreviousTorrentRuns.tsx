import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertTriangle,
  CheckCircle2,
  ChevronRight,
  FileVideo,
  History,
  Search,
  ShieldCheck,
  XCircle,
} from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  api,
  Busy,
  Empty,
  ErrorState,
  bytes,
  dateLabel,
  useApp,
} from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { queryKeys } from "../queryKeys";
import { TorrentTabs } from "./TorrentTabs";

type DecisionStep = {
  stage: string;
  status?: string;
  summary?: string;
  data?: Record<string, any>;
  occurred_at: number;
  duration_ms?: number;
};

type Feedback = {
  profile_id?: string;
  reason: string;
  note?: string;
  created_at: number;
};

type AutomationRun = {
  id: string;
  show_id: string;
  episode_id: string;
  show_name: string;
  season: number;
  episode: number;
  query: string;
  status: string;
  confidence?: string;
  verification?: string;
  selected_name?: string;
  selected_infohash?: string;
  settings_snapshot: Record<string, any>;
  decision_log: DecisionStep[];
  engine_version: string;
  started_at: number;
  ended_at?: number;
  duration_ms: number;
  feedback?: Feedback;
};

type RunsResponse = { runs: AutomationRun[] };

const feedbackReasons = [
  "wrong_show",
  "wrong_episode",
  "wrong_language",
  "poor_quality",
  "corrupt",
  "suspicious_files",
  "other",
] as const;

function episodeLabel(run: AutomationRun) {
  return `S${String(run.season).padStart(2, "0")}E${String(run.episode).padStart(2, "0")}`;
}

function titleCase(value: string) {
  return value.replaceAll("_", " ").replace(/\b\w/g, (letter) => letter.toUpperCase());
}

function StatusIcon({ status }: { status?: string }) {
  if (status === "success" || status === "selected") return <CheckCircle2 size={17} />;
  if (status === "failed" || status === "rejected") return <XCircle size={17} />;
  if (status === "skipped" || status === "no_verified_candidate") return <AlertTriangle size={17} />;
  return <ChevronRight size={17} />;
}

function ConfidenceBadge({ run }: { run: AutomationRun }) {
  if (!run.confidence) return null;
  return (
    <span className={`badge confidence-${run.confidence}`}>
      {titleCase(run.confidence)}
      {run.verification ? ` · ${titleCase(run.verification)}` : ""}
    </span>
  );
}

export function PreviousTorrentRunsPage() {
  const { t } = useTranslation();
  const runs = useQuery<RunsResponse>({
    queryKey: queryKeys.torrentAutomationRuns(),
    queryFn: ({ signal }) =>
      api("/torrents/automation/runs?limit=100", "GET", undefined, signal),
  });
  const [selectedID, setSelectedID] = useState("");
  const [filter, setFilter] = useState("");

  useEffect(() => {
    if (!selectedID && runs.data?.runs.length) setSelectedID(runs.data.runs[0].id);
    if (selectedID && runs.data && !runs.data.runs.some((run) => run.id === selectedID))
      setSelectedID(runs.data.runs[0]?.id || "");
  }, [runs.data, selectedID]);

  const detail = useQuery<AutomationRun>({
    queryKey: queryKeys.torrentAutomationRun(selectedID),
    queryFn: ({ signal }) =>
      api(`/torrents/automation/runs/${selectedID}`, "GET", undefined, signal),
    enabled: !!selectedID,
  });

  const visibleRuns = useMemo(() => {
    const needle = filter.trim().toLowerCase();
    if (!needle) return runs.data?.runs ?? [];
    return (runs.data?.runs ?? []).filter((run) =>
      `${run.show_name} ${episodeLabel(run)} ${run.status} ${run.selected_name || ""}`
        .toLowerCase()
        .includes(needle),
    );
  }, [filter, runs.data]);

  if (runs.error) return <ErrorState error={runs.error} retry={() => runs.refetch()} />;

  return (
    <div className="page torrent-runs-page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">
            {t("torrentRuns.eyebrow", { defaultValue: "EVERY DECISION, EXPLAINED" })}
          </span>
          <h1>
            {t("search.title")}<span className="accent">.</span>
          </h1>
          <p>
            {t("torrentRuns.description", {
              defaultValue:
                "Review what Tally searched, rejected, verified, and selected during previous automated runs.",
            })}
          </p>
        </div>
      </div>
      <TorrentTabs />

      {runs.isPending ? (
        <Busy />
      ) : !runs.data?.runs.length ? (
        <div className="panel">
          <Empty icon={<History size={30} />} title={t("torrentRuns.empty", { defaultValue: "No automated runs yet" })}>
            {t("torrentRuns.emptyHelp", {
              defaultValue: "Runs will appear here after Torrent automation checks released episodes.",
            })}
          </Empty>
        </div>
      ) : (
        <div className="torrent-run-layout">
          <aside className="panel torrent-run-list">
            <div className="torrent-run-list-head">
              <h3>{t("torrentRuns.previousRuns", { defaultValue: "Previous Runs" })}</h3>
              <span className="muted small-text">{runs.data.runs.length}</span>
            </div>
            <div className="search-input torrent-run-filter">
              <Search size={16} />
              <input
                value={filter}
                onChange={(event) => setFilter(event.target.value)}
                placeholder={t("torrentRuns.search", { defaultValue: "Search runs…" })}
                aria-label={t("torrentRuns.search", { defaultValue: "Search runs" })}
              />
            </div>
            <div className="torrent-run-items">
              {visibleRuns.map((run) => (
                <button
                  type="button"
                  key={run.id}
                  className={`torrent-run-item ${selectedID === run.id ? "selected" : ""}`}
                  onClick={() => setSelectedID(run.id)}
                >
                  <span className="torrent-run-item-title">
                    <strong>{run.show_name}</strong>
                    {run.feedback && (
                      <span className="badge failed">
                        {t("torrentRuns.markedBad", { defaultValue: "Marked bad" })}
                      </span>
                    )}
                  </span>
                  <span>{episodeLabel(run)}</span>
                  <span className="muted small-text">{dateLabel(run.started_at)}</span>
                  <span className="torrent-run-item-badges">
                    <span className={`badge ${run.status === "downloaded" ? "success" : run.status === "failed" ? "failed" : ""}`}>
                      {titleCase(run.status)}
                    </span>
                  </span>
                </button>
              ))}
            </div>
          </aside>

          <main className="panel torrent-run-inspection">
            {detail.error ? (
              <ErrorState error={detail.error} retry={() => detail.refetch()} />
            ) : detail.isPending || !detail.data ? (
              <Busy />
            ) : (
              <RunInspection run={detail.data} />
            )}
          </main>

          <aside className="panel torrent-run-details">
            {detail.data ? <RunDetails run={detail.data} /> : <Busy />}
          </aside>
        </div>
      )}
    </div>
  );
}

function RunInspection({ run }: { run: AutomationRun }) {
  const { t } = useTranslation();
  return (
    <>
      <div className="torrent-run-inspection-head">
        <span className="eyebrow">
          {t("torrentRuns.inspection", { defaultValue: "RUN INSPECTION" })}
        </span>
        <h2>{run.show_name} · {episodeLabel(run)}</h2>
        <p className="muted">{dateLabel(run.started_at)}</p>
        <div className="torrent-run-statuses">
          <span className={`badge ${run.status === "downloaded" ? "success" : run.status === "failed" ? "failed" : ""}`}>
            {titleCase(run.status)}
          </span>
          {run.feedback && (
            <span className="badge failed">
              {t("torrentRuns.markedBad", { defaultValue: "Marked bad" })}
            </span>
          )}
          <ConfidenceBadge run={run} />
        </div>
      </div>

      {run.feedback && (
        <div className="torrent-run-feedback-banner">
          <AlertTriangle size={18} />
          <div>
            <strong>
              {t("torrentRuns.feedback", { defaultValue: "User feedback" })}: {titleCase(run.feedback.reason)}
            </strong>
            {run.feedback.note && <p>{run.feedback.note}</p>}
          </div>
        </div>
      )}

      <div className="decision-timeline">
        {run.decision_log.map((step, index) => (
          <section className="decision-step" key={`${step.stage}-${index}`}>
            <div className="decision-marker">{index + 1}</div>
            <div className="decision-card">
              <div className="decision-step-head">
                <h3><StatusIcon status={step.status} />{titleCase(step.stage)}</h3>
                {step.duration_ms ? <span>{(step.duration_ms / 1000).toFixed(1)}s</span> : null}
              </div>
              {step.summary && <p>{step.summary}</p>}
              <DecisionData step={step} />
            </div>
          </section>
        ))}
        {run.feedback && (
          <section className="decision-step">
            <div className="decision-marker">{run.decision_log.length + 1}</div>
            <div className="decision-card decision-feedback">
              <div className="decision-step-head">
                <h3><AlertTriangle size={17} />{t("torrentRuns.postFeedback", { defaultValue: "Post-run feedback" })}</h3>
              </div>
              <p>{t("torrentRuns.markedBadReason", {
                defaultValue: "User marked this result as bad: {{reason}}",
                reason: titleCase(run.feedback.reason),
              })}</p>
            </div>
          </section>
        )}
      </div>

      {run.status === "downloaded" && !run.feedback && <MarkBad run={run} />}
    </>
  );
}

function DecisionData({ step }: { step: DecisionStep }) {
  const data = step.data || {};
  const candidates = Array.isArray(data.candidates) ? data.candidates : [];
  const payload = data.payload as any;
  return (
    <>
      {step.stage === "search" && typeof data.candidate_count === "number" && (
        <p className="muted small-text">{data.candidate_count} candidates returned</p>
      )}
      {step.stage === "filter" && (
        <div className="decision-metrics">
          {typeof data.rejected === "number" && <span>{data.rejected} rejected</span>}
          {typeof data.magnet_only === "number" && <span>{data.magnet_only} magnet-only</span>}
          {typeof data.previously_bad === "number" && <span>{data.previously_bad} previously marked bad</span>}
          {typeof data.shortlisted === "number" && <span>{data.shortlisted} shortlisted</span>}
        </div>
      )}
      {!!candidates.length && (
        <div className="table-scroll decision-candidates">
          <table>
            <thead><tr><th>#</th><th>Release</th><th>Seeds</th><th>Size</th><th>Confidence</th></tr></thead>
            <tbody>
              {candidates.map((candidate: any) => (
                <tr key={`${candidate.rank}-${candidate.name}`}>
                  <td>{candidate.rank}</td>
                  <td>{candidate.name}</td>
                  <td>{candidate.seeders}</td>
                  <td>{candidate.size ? bytes(candidate.size) : "—"}</td>
                  <td>{titleCase(candidate.confidence || "")}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {payload && (
        <div className="torrent-payload-summary">
          <span><FileVideo size={15} />{payload.video_files ?? 0} video</span>
          <span>{payload.subtitle_files ?? 0} subtitles</span>
          <span>{payload.executable_files ?? 0} executable</span>
          {payload.total_size ? <span>{bytes(payload.total_size)}</span> : null}
          {Array.isArray(payload.files) && payload.files.length > 0 && (
            <details>
              <summary>Files ({payload.files.length})</summary>
              <div className="torrent-file-tree">
                {payload.files.map((file: any) => (
                  <div key={`${file.path}-${file.size}`}>
                    <code>{file.path}</code><span>{bytes(file.size)}</span>
                  </div>
                ))}
              </div>
            </details>
          )}
        </div>
      )}
      {data.name && step.stage !== "filter" && <p className="muted small-text"><code>{data.name}</code></p>}
      {Array.isArray(data.hard_rejections) && data.hard_rejections.length > 0 && (
        <div className="decision-rejections">
          {data.hard_rejections.map((reason: any, index: number) => (
            <span key={`${reason.code}-${index}`}>{titleCase(reason.code || "rejected")}{reason.detail ? ` · ${reason.detail}` : ""}</span>
          ))}
        </div>
      )}
    </>
  );
}

function RunDetails({ run }: { run: AutomationRun }) {
  const { t } = useTranslation();
  const snapshot = run.settings_snapshot || {};
  const automation = snapshot.automation || {};
  return (
    <>
      <div className="torrent-run-list-head">
        <h3>{t("torrentRuns.runDetails", { defaultValue: "Run details" })}</h3>
        <span className="muted small-text">#{run.id.slice(-6)}</span>
      </div>
      <dl className="torrent-run-detail-list">
        <div><dt>{t("torrentRuns.query", { defaultValue: "Query" })}</dt><dd>{run.query}</dd></div>
        <div><dt>{t("torrentRuns.showPolicy", { defaultValue: "Show policy" })}</dt><dd>{titleCase(snapshot.show_policy || "default")}</dd></div>
        <div><dt>{t("torrentRuns.minSeeders", { defaultValue: "Minimum seeders" })}</dt><dd>{automation.min_seeders ?? "—"}</dd></div>
        <div><dt>{t("torrentRuns.quality", { defaultValue: "Preferred quality" })}</dt><dd>{automation.preferred_quality || "—"}</dd></div>
        <div><dt>{t("torrentRuns.releaseDelay", { defaultValue: "Release delay" })}</dt><dd>{automation.release_delay_minutes != null ? `${automation.release_delay_minutes} min` : "—"}</dd></div>
        <div><dt>{t("torrentRuns.engine", { defaultValue: "Decision engine" })}</dt><dd>v{run.engine_version}</dd></div>
        <div><dt>{t("torrentRuns.infohash", { defaultValue: "Infohash" })}</dt><dd><code>{run.selected_infohash ? `${run.selected_infohash.slice(0, 10)}…${run.selected_infohash.slice(-6)}` : "—"}</code></dd></div>
        <div><dt>{t("torrentRuns.duration", { defaultValue: "Duration" })}</dt><dd>{(run.duration_ms / 1000).toFixed(1)}s</dd></div>
      </dl>
      <details className="raw-run-data">
        <summary>{t("torrentRuns.rawData", { defaultValue: "View raw run data" })}</summary>
        <pre>{JSON.stringify(run, null, 2)}</pre>
      </details>
    </>
  );
}

function MarkBad({ run }: { run: AutomationRun }) {
  const { t } = useTranslation();
  const { notify } = useApp();
  const cache = useQueryClient();
  const [open, setOpen] = useState(false);
  const [reason, setReason] = useState<(typeof feedbackReasons)[number]>("wrong_episode");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit() {
    setBusy(true);
    try {
      await api(`/torrents/automation/runs/${run.id}/bad`, "POST", { reason, note });
      await invalidateResources(cache, ["torrent-automation-runs"]);
      await cache.invalidateQueries({ queryKey: queryKeys.torrentAutomationRun(run.id) });
      notify(t("torrentRuns.badSaved", { defaultValue: "Run marked as bad" }));
      setOpen(false);
    } catch (error) {
      notify((error as Error).message, true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="mark-bad-run">
      {!open ? (
        <button className="button" type="button" onClick={() => setOpen(true)}>
          <AlertTriangle size={16} />
          {t("torrentRuns.markBad", { defaultValue: "Mark as bad" })}
        </button>
      ) : (
        <div className="panel mark-bad-form">
          <h3>{t("torrentRuns.whyBad", { defaultValue: "Why was this result bad?" })}</h3>
          <label>
            {t("common.result")}
            <select value={reason} onChange={(event) => setReason(event.target.value as (typeof feedbackReasons)[number])}>
              {feedbackReasons.map((value) => (
                <option value={value} key={value}>{titleCase(value)}</option>
              ))}
            </select>
          </label>
          <label>
            {t("torrentRuns.note", { defaultValue: "Note (optional)" })}
            <textarea maxLength={500} value={note} onChange={(event) => setNote(event.target.value)} />
          </label>
          <div className="settings-actions">
            <button className="button" type="button" disabled={busy} onClick={() => setOpen(false)}>
              {t("common.cancel")}
            </button>
            <button className="button primary" type="button" disabled={busy} onClick={() => void submit()}>
              {busy ? <Busy /> : <ShieldCheck size={16} />}
              {t("torrentRuns.markBad", { defaultValue: "Mark as bad" })}
            </button>
          </div>
        </div>
      )}
    </section>
  );
}
