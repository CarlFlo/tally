import { scheduleRunLabel } from "../schedules";
export type Preview = {
  expression: string;
  description: string;
  next_runs: number[];
  timezone: string;
};

export function SchedulePreview({
  expression,
  preview,
  loading,
  error,
  timeFormat,
  timezone,
}: {
  expression: string;
  preview?: Preview;
  loading: boolean;
  error?: Error;
  timeFormat: string;
  timezone: string;
}) {
  const description = loading
    ? "Checking schedule…"
    : error?.message || preview?.description || "Enter a cron schedule.";
  return (
    <div
      className={`schedule-preview ${error ? "is-invalid" : ""}`}
      aria-live="polite"
      aria-busy={loading}
    >
      <dl>
        <div>
          <dt>Cron expression</dt>
          <dd><code>{expression || "—"}</code></dd>
        </div>
        <div className="schedule-description-value">
          <dt>Description</dt>
          <dd>{description}</dd>
        </div>
        <div>
          <dt>Timezone</dt>
          <dd>{preview?.timezone || "UTC"}</dd>
        </div>
      </dl>
      <strong>Next 3 runs · {timezone}</strong>
      <ol>
        {[0, 1, 2].map((index) => (
          <li key={index}>
            {preview?.next_runs[index]
              ? scheduleRunLabel(preview.next_runs[index], timeFormat, timezone)
              : "—"}
          </li>
        ))}
      </ol>
    </div>
  );
}
