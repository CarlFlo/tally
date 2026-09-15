import { useEffect, useState, type FormEvent } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Save } from "lucide-react";
import { api, Busy, ErrorState, useApp } from "../lib";
import { commonSchedules, jobDescription, jobName } from "../schedules";
import { useDebouncedValue } from "../useDebouncedValue";
import { SchedulePreview, type Preview } from "./SchedulePreview";
import { queryKeys } from "../queryKeys";
import { invalidateResources } from "../queryInvalidation";

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
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [saved, setSaved] = useState(job);
  const [spec, setSpec] = useState(job.schedule);
  const [enabled, setEnabled] = useState(!!job.enabled);
  const [busy, setBusy] = useState(false);
  const [saveError, setSaveError] = useState<Error>();
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
      await invalidateResources(cache, ["schedules"]);
      notify(automatic !== !!saved.enabled
        ? `${jobName(job.key)} ${automatic ? "enabled" : "disabled"}`
        : "Schedule saved");
    } catch (error) {
      setEnabled(!!saved.enabled);
      setSaveError(error as Error);
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
    <form autoComplete="off" className={`panel settings-card schedule-editor ${!saved.enabled ? "is-disabled" : ""}`} onSubmit={(event: FormEvent) => { event.preventDefault(); void save(spec, !!saved.enabled); }}>
      <fieldset disabled={busy}>
        <div className="schedule-heading">
          <div><h3>{jobName(job.key)}</h3><p>{jobDescription(job.key)}</p></div>
          <label className="toggle-setting"><input type="checkbox" checked={enabled} onChange={(event) => void save(saved.schedule, event.target.checked)} />Run automatically</label>
        </div>
        <div className="schedule-editor-body">
          <div className="schedule-fields">
            <label>Common schedule<select value={selectedPreset} aria-label={`${jobName(job.key)} common schedule`} onChange={(event) => event.target.value !== "custom" && setSpec(event.target.value)}><option value="custom">Custom cron</option>{presets.map(([label, value]) => <option key={value} value={value}>{label}</option>)}</select><small aria-hidden="true">&nbsp;</small></label>
            <label>Cron expression<input name={`cron-${job.key}`} value={spec} required maxLength={100} spellCheck={false} autoComplete="off" data-1p-ignore="true" data-lpignore="true" data-bwignore="true" data-protonpass-ignore="true" data-form-type="other" onChange={(event) => setSpec(event.target.value)} /><small>Minute · hour · day-of-month · month · weekday</small></label>
            <button className="button primary small">{busy ? <Busy /> : <Save size={16} />}Save schedule</button>
          </div>
          <SchedulePreview expression={spec} preview={currentPreview || initialPreview} loading={!settled || preview.isFetching} error={settled ? preview.error as Error | undefined : undefined} timeFormat={boot.preferences.time_format} timezone={boot.preferences.timezone} />
        </div>
        {!!job.paused && <p className="error-box">Paused after {job.failures} consecutive failures. Resume from System → Jobs when ready.</p>}
      </fieldset>
      {saveError && <ErrorState error={saveError} />}
    </form>
  );
}
