import { useEffect, useRef, useState, type FormEvent } from "react";
import { Trans, useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { api, ErrorState, useApp } from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { UnsavedChangesBar, useUnsavedChangesWarning } from "../UnsavedChangesBar";
import "../unsaved-changes.css";

export function BackupRetention({ saved }: { saved: any }) {
  const { t } = useTranslation();
  const [keep, setKeep] = useState(saved.data.keep);
  const [revision, setRevision] = useState(saved.revision);
  const previousSaved = useRef(saved);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const cache = useQueryClient();
  const { notify } = useApp();
  const hasChanges = keep !== previousSaved.current.data.keep;
  useUnsavedChangesWarning(hasChanges, busy, t("common.unsavedNavigation"));
  useEffect(() => {
    if (keep === previousSaved.current.data.keep) {
      setKeep(saved.data.keep);
      setRevision(saved.revision);
    }
    previousSaved.current = saved;
  }, [keep, saved]);
  async function save(event?: FormEvent) {
    event?.preventDefault();
    setBusy(true);
    setError(null);
    try {
      const result = await api("/settings/backups", "PUT", {
        data: { keep },
        revision,
      });
      setRevision(result.revision);
      previousSaved.current = { ...previousSaved.current, data: { keep }, revision: result.revision };
      await invalidateResources(cache, ["editable-settings"]);
      notify(t("backups.saved"));
    } catch (e) {
      setError(e as Error);
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
    <form className="panel settings-card compact-retention">
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
      </div>
      {error && <ErrorState error={error} />}
    </form>
    <UnsavedChangesBar
      hasChanges={hasChanges}
      busy={busy}
      onRevert={() => { setKeep(previousSaved.current.data.keep); setError(null); }}
      onSave={() => void save()}
      statusLabel={t("common.unsavedChanges")}
      revertLabel={t("common.revertChanges")}
      saveLabel={t("common.saveChanges")}
    />
    </>
  );
}
