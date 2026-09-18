import { ErrorState, useLocal } from "../lib";
import { useTranslation } from "react-i18next";
import { BackupArchives } from "./BackupArchives";
import { BackupRetention } from "./BackupRetention";
import { ScheduleEditor, type Schedule } from "./ScheduleEditor";

const scheduleOrder = ["metadata", "torrent_automation", "maintenance", "backup"];

export function SchedulingBackupsSettings() {
  const { t } = useTranslation();
  const settings = useLocal<any>("editable-settings", "/settings/backups");
  const deployment = useLocal<any>("settings", "/settings");
  const schedules = useLocal<Schedule[]>("schedules", "/settings/scheduling");
  const orderedSchedules = schedules.data?.slice().sort(
    (a, b) => scheduleOrder.indexOf(a.key) - scheduleOrder.indexOf(b.key),
  );
  return (
    <div className="scheduling-backups">
      <section className="settings-group" aria-labelledby="schedule-group-title">
        <div className="section-heading settings-group-heading">
          <div>
            <h2 id="schedule-group-title">{t("backups.scheduling")}</h2>
            <p className="muted">
{t("backups.timezoneHelp", {
                timezone: deployment.data?.timezone || "UTC",
              })}
            </p>
          </div>
        </div>
        {schedules.error && <ErrorState error={schedules.error} retry={() => schedules.refetch()} />}
        <div className="settings-card-list">
          {orderedSchedules?.map((job) => <ScheduleEditor key={job.key} job={job} />)}
        </div>
      </section>
      <section className="settings-group backup-settings-group" aria-labelledby="backup-group-title">
        <div className="section-heading settings-group-heading">
          <div>
            <h2 id="backup-group-title">{t("backups.retentionArchives")}</h2>
            <p className="muted">{t("backups.scheduleHelp")}</p>
          </div>
        </div>
        {settings.error && <ErrorState error={settings.error} retry={() => settings.refetch()} />}
        <div className="settings-card-list">
          {settings.data && <BackupRetention saved={settings.data} />}
          <BackupArchives />
        </div>
      </section>
    </div>
  );
}
