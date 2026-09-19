import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, ErrorState, useApp } from "../lib";
import { commonSchedules, isExperimentalJob, jobDescription, jobName } from "../schedules";
import { useDebouncedValue } from "../useDebouncedValue";
import { SchedulePreview, type Preview } from "./SchedulePreview";
import { queryKeys } from "../queryKeys";
import { invalidateResources } from "../queryInvalidation";
import { ExperimentalJobConfirmation } from "../ExperimentalJobConfirmation";
import { UnsavedChangesBar, useUnsavedChangesWarning } from "../UnsavedChangesBar";
import "../unsaved-changes.css";

export type Schedule = {
  key: string;
  schedule: string;
  enabled: number | boolean;
  revision: number;
  paused: number;
  failures: number;
  preview?: Preview;
};

export function ScheduleEditor({ job }: { job: Schedule }) {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [saved, setSaved] = useState(job);
  const [spec, setSpec] = useState(job.schedule);
  const [enabled, setEnabled] = useState(!!job.enabled);
  const [busy, setBusy] = useState(false);
  const [saveError, setSaveError] = useState<Error>();
  const [confirmExperimental, setConfirmExperimental] = useState(false);
  const hasChanges = spec !== saved.schedule || enabled !== !!saved.enabled;
  useUnsavedChangesWarning(hasChanges, busy, t("common.unsavedNavigation"));
  const previewSpec = useDebouncedValue(spec, 350);
  const preview = useQuery<Preview>({
    queryKey: queryKeys.schedulePreview(previewSpec),
    queryFn: ({ signal }) =>
      api(
        "/settings/scheduling/preview",
        "POST",
        { schedule: previewSpec },
        signal,
      ),
    enabled: previewSpec.trim().length > 0,
    retry: false,
    staleTime: 10_000,
  });
  useEffect(() => {
    if (spec === saved.schedule && enabled === !!saved.enabled && job.revision > saved.revision) {
      setSaved(job);
      setSpec(job.schedule);
      setEnabled(!!job.enabled);
    }
  }, [enabled, job, saved, spec]);

  async function save(schedule: string, automatic: boolean) {
    setBusy(true);
    setSaveError(undefined);
    setEnabled(automatic);
    try {
      const result = await api("/settings/scheduling", "PUT", {
        key: job.key, schedule, enabled: automatic, revision: saved.revision,
      });
      setSaved({ ...saved, schedule, enabled: automatic, revision: result.revision });
      await invalidateResources(cache, ["schedules", "jobs"]);
      notify(
        automatic !== !!saved.enabled
          ? t(automatic ? "schedule.enabled" : "schedule.disabled", {
              name: jobName(job.key),
            })
          : t("schedule.saved"),
      );
      return true;
    } catch (error) {
      setEnabled(!!saved.enabled);
      setSaveError(error as Error);
      return false;
    } finally {
      setBusy(false);
    }
  }

  const settled = previewSpec === spec;
  const currentPreview = settled ? preview.data : undefined;
  const initialPreview = spec === job.schedule ? job.preview : undefined;
  const presets = commonSchedules[job.key] || [];
  const selectedPreset = presets.find(([, value]) => value === spec)?.[1] || "custom";
  return (
    <>
    <form autoComplete="off" className={`panel settings-card schedule-editor ${!enabled ? "is-disabled" : ""}`} onSubmit={(event) => event.preventDefault()}>
      <fieldset disabled={busy}>
        <div className="schedule-heading">
          <div><h3>{jobName(job.key)}</h3><p>{jobDescription(job.key)}</p></div>
          <label className="toggle-setting"><input type="checkbox" checked={enabled} onChange={(event) => {
            const nextEnabled = event.target.checked;
            if (nextEnabled && !saved.enabled && isExperimentalJob(job.key)) {
              setConfirmExperimental(true);
              return;
            }
            setEnabled(nextEnabled);
          }} />{t("schedule.automatic")}</label>
        </div>
        <div className="schedule-editor-body">
          <div className="schedule-fields">
            <label>{t("schedule.common")}<select value={selectedPreset} aria-label={t("schedule.commonFor", { name: jobName(job.key) })} onChange={(event) => event.target.value !== "custom" && setSpec(event.target.value)}><option value="custom">{t("schedule.custom")}</option>{presets.map(([label, value]) => <option key={value} value={value}>{t(label)}</option>)}</select><small aria-hidden="true">&nbsp;</small></label>
            <label>{t("schedule.expression")}<input name={`cron-${job.key}`} value={spec} required maxLength={100} spellCheck={false} autoComplete="off" data-1p-ignore="true" data-lpignore="true" data-bwignore="true" data-protonpass-ignore="true" data-form-type="other" onChange={(event) => setSpec(event.target.value)} /><small>{t("schedule.fields")}</small></label>
          </div>
          <SchedulePreview preview={currentPreview || initialPreview} loading={!settled || preview.isFetching} error={settled ? preview.error as Error | undefined : undefined} timeFormat={boot.preferences.time_format} />
        </div>
        {!!job.paused && <p className="error-box">{t("schedule.paused", { count: job.failures })}</p>}
      </fieldset>
      {saveError && <ErrorState error={saveError} />}
      {confirmExperimental && (
        <ExperimentalJobConfirmation
          jobKey={job.key}
          onClose={() => setConfirmExperimental(false)}
          onConfirm={async () => {
            setEnabled(true);
            return true;
          }}
        />
      )}
    </form>
    <UnsavedChangesBar
      hasChanges={hasChanges}
      busy={busy}
      onRevert={() => { setSpec(saved.schedule); setEnabled(!!saved.enabled); setSaveError(undefined); }}
      onSave={() => void save(spec, enabled)}
      statusLabel={t("common.unsavedChanges")}
      revertLabel={t("common.revertChanges")}
      saveLabel={t("common.saveChanges")}
    />
    </>
  );
}
