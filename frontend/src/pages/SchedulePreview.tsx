import { scheduleRunLabel } from "../schedules";

export type Preview = {
  description: string;
  next_runs: number[];
  timezone: string;
};

export function SchedulePreview({
  preview,
  loading,
  error,
  timeFormat,
}: {
  preview?: Preview;
  loading: boolean;
  error?: Error;
  timeFormat: string;
}) {
  const description = loading
    ? "Checking schedule…"
    : error?.message || preview?.description || "Enter a cron schedule.";
  const timezone = preview?.timezone || "UTC";
  return (
    <div
      className={`schedule-preview ${error ? "is-invalid" : ""}`}
      aria-live="polite"
      aria-busy={loading}
    >
      <div className="schedule-preview-section">
        <span className="schedule-preview-label">Description</span>
        <p className="schedule-preview-description">{description}</p>
      </div>
      <div className="schedule-preview-section schedule-preview-runs">
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
    </div>
  );
}
