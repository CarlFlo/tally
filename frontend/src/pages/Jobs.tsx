import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  Archive,
  Activity,
  Play,
  RefreshCw,
  SlidersHorizontal,
  X,
} from "lucide-react";
import {
  api,
  Busy,
  dateLabel,
  Empty,
  ErrorState,
  useApp,
  useLocal,
} from "../lib";
import { jobName } from "../schedules";
import { queryKeys } from "../queryKeys";
import { invalidateResources } from "../queryInvalidation";

export function JobsPage() {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const prefs = boot.preferences;
  const kind = prefs.job_type_filter || "all",
    status = prefs.job_status_filter || "all";
  const jobs = useQuery<any>({
    queryKey: queryKeys.jobs(kind, status),
    queryFn: ({ signal }) =>
      api(`/jobs?kind=${kind}&status=${status}`, "GET", undefined, signal),
  });
  const settings = useLocal<any>("settings", "/settings");
  const operator = settings.data?.operator;
  const [busy, setBusy] = useState("");
  async function preference(key: string, value: string) {
    try {
      await api("/preferences", "PATCH", { [key]: value });
      await invalidateResources(cache, ["bootstrap"]);
    } catch (e) {
      notify((e as Error).message, true);
    }
  }
  async function action(key: string, resume = false) {
    setBusy(key);
    try {
      await api(`/jobs/${key}${resume ? "/resume" : ""}`, "POST", {});
      notify(resume ? t("jobs.scheduleResumed") : t("jobs.startedNotice"));
      await invalidateResources(cache, ["jobs", "schedules"]);
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy("");
    }
  }
  const preview = prefs.debug_mode
    ? prefs.debug_job_state || "normal"
    : "normal";
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">{t("jobs.eyebrow")}</span>
          <h1>
            {t("jobs.title")}<span className="accent">.</span>
          </h1>
        </div>
      </div>
      {jobs.error && <ErrorState error={jobs.error} />}
      {prefs.debug_mode && (
        <div className="panel debug-panel">
          <strong>{t("jobs.appearancePreview")}</strong>
          <span className="muted small-text">
{t("jobs.previewHelp")}
          </span>
          <div className="debug-actions">
            {["failed", "paused", "disabled", "normal"].map((state) => (
              <button
                key={state}
                className={
                  "button small " + (preview === state ? "active" : "")
                }
                onClick={() => preference("debug_job_state", state)}
              >
                {state === "normal" ? t("jobs.resetPreview") : t("jobs.previewState", { state })}
              </button>
            ))}
          </div>
        </div>
      )}
      <div className="job-cards">
        {jobs.data?.schedules.map((real: any) => {
          const job =
            real.key === "metadata" && preview !== "normal"
              ? {
                  ...real,
                  enabled: preview === "disabled" ? 0 : 1,
                  paused: preview === "paused" ? 1 : 0,
                  failures: preview === "paused" ? 3 : 1,
                  last_status: preview === "disabled" ? "idle" : "failed",
                }
              : real;
          return (
            <section
              className={`panel job-card ${!job.enabled ? "is-disabled" : ""} ${job.last_status === "failed" || job.paused ? "has-failed" : ""}`}
              key={job.key}
            >
              <div className="job-card-head">
                <span
                  className={
                    "metric-icon " +
                    (job.key === "metadata"
                      ? "purple"
                      : job.key === "backup"
                        ? "mint"
                        : "amber")
                  }
                >
                  {job.key === "metadata" ? (
                    <RefreshCw size={21} />
                  ) : job.key === "backup" ? (
                    <Archive size={21} />
                  ) : (
                    <SlidersHorizontal size={21} />
                  )}
                </span>
                <span className={"badge " + (job.paused ? "failed" : "")}>
                  {!job.enabled
                    ? t("jobs.off")
                    : job.paused
                      ? t("jobs.paused")
                      : job.last_status === "running"
                        ? t("jobs.running")
                        : t("jobs.scheduled")}
                </span>
              </div>
              <h3>{jobName(job.key)}</h3>
              <p>
                {job.key === "metadata"
                  ? t("jobs.metadataCardHelp")
                  : job.key === "backup"
                    ? t("jobs.backupCardHelp")
                    : t("jobs.maintenanceCardHelp")}
              </p>
              <dl>
                <div>
                  <dt>{t("jobs.schedule")}</dt>
                  <dd title={job.schedule}>
                    {job.description || job.schedule}
                  </dd>
                </div>
                <div>
                  <dt>{t("jobs.nextRun")}</dt>
                  <dd>
                    {!job.enabled
                      ? t("jobs.turnedOff")
                      : job.paused
                        ? t("jobs.paused")
                        : dateLabel(job.next_run)}
                  </dd>
                </div>
              </dl>
              <p className="job-mini-stats">
{t("jobs.last7", {
                  successes: real.successes_7d,
                  failures: real.failures_7d,
                })}
              </p>
              {!!job.paused && (
                <p className="job-pause-notice">
                  {t("jobs.pausedFailures", { count: job.failures })}
                </p>
              )}
              <div className="job-actions">
                <button
                  className="button"
                  disabled={
                    busy === job.key ||
                    real.last_status === "running" ||
                    (job.key !== "metadata" && !operator)
                  }
                  onClick={() => action(job.key)}
                >
                  {busy === job.key ? <Busy /> : <Play size={15} />}
                  {real.paused ? t("jobs.retryNow") : t("jobs.runNow")}
                </button>
                {!!real.paused && operator && (
                  <button
                    className="button"
                    onClick={() => action(job.key, true)}
                  >
                    {t("jobs.resumeSchedule")}
                  </button>
                )}
              </div>
            </section>
          );
        })}
      </div>
      <div className="section-heading run-heading">
        <h2>{t("jobs.history")}</h2>
        <div className="history-filters">
          <label>
            {t("jobs.job")}
            <select
              aria-label={t("jobs.filterJob")}
              value={kind}
              onChange={(e) => preference("job_type_filter", e.target.value)}
            >
              <option value="all">{t("jobs.all")}</option>
              <option value="metadata">{t("jobs.metadata")}</option>
              <option value="backup">{t("jobs.backup")}</option>
              <option value="maintenance">{t("jobs.maintenance")}</option>
            </select>
          </label>
          <label>
            {t("common.result")}
            <select
              aria-label={t("jobs.filterResult")}
              value={status}
              onChange={(e) => preference("job_status_filter", e.target.value)}
            >
              {[
                "all",
                "success",
                "failed",
                "running",
                "cancelled",
                "interrupted",
              ].map((value) => (
                <option key={value} value={value}>
                  {value === "all"
                    ? t("jobs.allResults")
                    : t(`jobs.${value}`)}
                </option>
              ))}
            </select>
          </label>
        </div>
      </div>
      <p className="muted small-text">{t("jobs.latest")}</p>
      <div className="panel table-scroll">
        {jobs.data?.runs.length ? (
          <table>
            <thead>
              <tr>
                <th>{t("jobs.jobTrigger")}</th>
                <th>{t("jobs.started")}</th>
                <th>{t("jobs.duration")}</th>
                <th>{t("jobs.processed")}</th>
                <th>{t("jobs.apiCache")}</th>
                <th>{t("jobs.changes")}</th>
                <th>{t("common.result")}</th>
              </tr>
            </thead>
            <tbody>
              {jobs.data.runs.map((run: any) => (
                <tr key={run.id}>
                  <td>
                    <strong>
                      {run.show_name || jobName(run.job_key.split(":")[0])}
                    </strong>
                    <small>{run.trigger.replaceAll("_", " ")}</small>
                    {run.error && (
                      <span className="table-error">{run.error}</span>
                    )}
                  </td>
                  <td>{dateLabel(run.started_at)}</td>
                  <td>
                    {run.status === "running" ? (
                      <Busy />
                    ) : (
                      `${(run.duration_ms / 1000).toFixed(1)}s`
                    )}
                  </td>
                  <td>
                    {run.processed}/{run.candidates}
                    <small>
{t("jobs.skippedAttempt", {
                        skipped: run.skipped,
                        attempt: run.attempt,
                      })}
                    </small>
                  </td>
                  <td>
                    {run.api_calls} / {run.cache_hits}
                  </td>
                  <td>{run.changes}</td>
                  <td>
                    <span className={"badge " + run.status}>{run.status}</span>
                    {run.status === "running" && operator && (
                      <button
                        className="icon-button"
                        aria-label={t("jobs.cancel")}
                        onClick={async () => {
                          try {
                            await api("/jobs/runs/" + run.id, "DELETE");
                            notify(t("jobs.cancelRequested"));
                            await invalidateResources(cache, ["jobs"]);
                          } catch (e) {
                            notify((e as Error).message, true);
                          }
                        }}
                      >
                        <X size={15} />
                      </button>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <Empty icon={<Activity size={28} />} title={t("jobs.empty")}>
            {t("jobs.emptyHelp")}
          </Empty>
        )}
      </div>
    </div>
  );
}
