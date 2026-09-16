import { useEffect, useRef, useState, type FormEvent } from "react";
import { Trans, useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { api, Busy, ErrorState, useApp } from "../lib";
import { invalidateResources } from "../queryInvalidation";

export function BackupRetention({ saved }: { saved: any }) {
  const { t } = useTranslation();
  const [keep, setKeep] = useState(saved.data.keep);
  const [revision, setRevision] = useState(saved.revision);
  const previousSaved = useRef(saved);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const cache = useQueryClient();
  const { notify } = useApp();
  useEffect(() => {
    if (keep === previousSaved.current.data.keep) {
      setKeep(saved.data.keep);
      setRevision(saved.revision);
    }
    previousSaved.current = saved;
  }, [keep, saved]);
  async function save(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const result = await api("/settings/backups", "PUT", {
        data: { keep },
        revision,
      });
      setRevision(result.revision);
      await invalidateResources(cache, ["editable-settings"]);
      notify(t("backups.saved"));
    } catch (e) {
      setError(e as Error);
    } finally {
      setBusy(false);
    }
  }
  return (
    <form className="panel settings-card compact-retention" onSubmit={save}>
      <h3>{t("backups.retention")}</h3>
      <p className="muted">
        <Trans
          i18nKey="backups.retentionHelp"
          components={{ strong: <strong /> }}
        />
      </p>
      <div className="schedule-fields">
        <label>
          {t("backups.automaticKeep")}
          <input
            type="number"
            min={1}
            max={1000}
            required
            value={keep}
            onChange={(e) => setKeep(Number(e.target.value))}
          />
        </label>
        <button className="button small" disabled={busy}>
          {busy && <Busy />}{t("backups.saveSettings")}
        </button>
      </div>
      {error && <ErrorState error={error} />}
    </form>
  );
}
