import { ErrorState, useLocal } from "../lib";
import { BackupArchives } from "./BackupArchives";
import { BackupRetention } from "./BackupRetention";
import { ScheduleEditor, type Schedule } from "./ScheduleEditor";

const scheduleOrder = ["metadata", "maintenance", "backup"];

export function SchedulingBackupsSettings() {
  const settings = useLocal<any>("editable-settings", "/settings/backups");
  const schedules = useLocal<Schedule[]>("schedules", "/settings/scheduling");
  const orderedSchedules = schedules.data?.slice().sort(
    (a, b) => scheduleOrder.indexOf(a.key) - scheduleOrder.indexOf(b.key),
  );
  return (
    <div className="scheduling-backups">
      <section className="settings-group" aria-labelledby="schedule-group-title">
        <div className="section-heading settings-group-heading">
          <div>
            <h2 id="schedule-group-title">Scheduling</h2>
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
            <h2 id="backup-group-title">Backup retention and archives</h2>
            <p className="muted">These controls belong to the Automatic backup schedule above.</p>
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
