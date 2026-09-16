import { useApp } from "../lib";
import { scheduleRunLabel } from "../schedules";
import { useTranslation } from "react-i18next";

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
  const { t } = useTranslation();
  const { boot } = useApp();
  const description = loading
    ? t("schedule.checking")
    : error?.message || preview?.description || t("schedule.enter");
  return (
    <div
      className={`schedule-preview ${error ? "is-invalid" : ""}`}
      aria-live="polite"
      aria-busy={loading}
    >
      <div className="schedule-preview-section">
        <span className="schedule-preview-label">{t("common.description")}</span>
        <p className="schedule-preview-description">{description}</p>
      </div>
      <div className="schedule-preview-section schedule-preview-runs">
        <strong>{t("schedule.nextRuns")}</strong>
        <ol>
          {[0, 1, 2].map((index) => (
            <li key={index}>
              {preview?.next_runs[index]
                ? scheduleRunLabel(
                    preview.next_runs[index],
                    timeFormat,
                    boot.preferences.timezone,
                  )
                : "—"}
            </li>
          ))}
        </ol>
      </div>
    </div>
  );
}
