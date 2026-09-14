import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
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
      notify(resume ? "Schedule resumed" : "Job started");
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
          <span className="eyebrow">QUIETLY KEEPING THINGS CURRENT</span>
          <h1>
            Jobs<span className="accent">.</span>
          </h1>
        </div>
      </div>
      {jobs.error && <ErrorState error={jobs.error} />}
      {prefs.debug_mode && (
        <div className="panel debug-panel">
          <strong>Appearance preview</strong>
          <span className="muted small-text">
            Preview the metadata card. Real jobs and their history are
            unchanged.
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
                {state === "normal" ? "Reset preview" : `Preview ${state}`}
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
                    ? "Off"
                    : job.paused
                      ? "Paused"
                      : job.last_status === "running"
                        ? "Running"
                        : "Scheduled"}
                </span>
              </div>
              <h3>{jobName(job.key)}</h3>
              <p>
                {job.key === "metadata"
                  ? "Refresh only the shows that are due."
                  : job.key === "backup"
                    ? "A verified snapshot of everything that matters."
                    : "Keep caches, sessions, and history tidy."}
              </p>
              <dl>
                <div>
                  <dt>Schedule · UTC</dt>
                  <dd title={job.schedule}>
                    {job.description || job.schedule}
                  </dd>
                </div>
                <div>
                  <dt>Next run</dt>
                  <dd>
                    {!job.enabled
                      ? "Turned off"
                      : job.paused
                        ? "Paused"
                        : dateLabel(job.next_run)}
                  </dd>
                </div>
              </dl>
              <p className="job-mini-stats">
                Last 7 days · {real.successes_7d} succeeded · {real.failures_7d}{" "}
                failed
              </p>
              {!!job.paused && (
                <p className="job-pause-notice">
                  Paused after {job.failures} consecutive failures.
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
                  {real.paused ? "Retry now" : "Run now"}
                </button>
                {!!real.paused && operator && (
                  <button
                    className="button"
                    onClick={() => action(job.key, true)}
                  >
                    Resume schedule
                  </button>
                )}
              </div>
            </section>
          );
        })}
      </div>
      <div className="section-heading run-heading">
        <h2>Run history</h2>
        <div className="history-filters">
          <label>
            Job
            <select
              aria-label="Filter history by job"
              value={kind}
              onChange={(e) => preference("job_type_filter", e.target.value)}
            >
              <option value="all">All jobs</option>
              <option value="metadata">Metadata sync</option>
              <option value="backup">Backup</option>
              <option value="maintenance">Maintenance</option>
            </select>
          </label>
          <label>
            Result
            <select
              aria-label="Filter history by result"
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
                    ? "All results"
                    : value[0].toUpperCase() + value.slice(1)}
                </option>
              ))}
            </select>
          </label>
        </div>
      </div>
      <p className="muted small-text">Latest 100 matching runs</p>
      <div className="panel table-scroll">
        {jobs.data?.runs.length ? (
          <table>
            <thead>
              <tr>
                <th>Job / trigger</th>
                <th>Started</th>
                <th>Duration</th>
                <th>Processed</th>
                <th>API / cache</th>
                <th>Changes</th>
                <th>Result</th>
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
                      {run.skipped} skipped · attempt {run.attempt}
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
                        aria-label="Cancel job"
                        onClick={async () => {
                          try {
                            await api("/jobs/runs/" + run.id, "DELETE");
                            notify("Cancellation requested");
                            await jobs.refetch();
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
          <Empty icon={<Activity size={28} />} title="No matching runs">
            Completed and running jobs appear here. Try changing the filters.
          </Empty>
        )}
      </div>
    </div>
  );
}
