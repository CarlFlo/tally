import { useEffect, useRef, useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { api, Busy, ErrorState, useApp } from "../lib";

export function BackupRetention({ saved }: { saved: any }) {
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
      await cache.invalidateQueries({ queryKey: ["editable-settings"] });
      notify("Backup settings saved");
    } catch (e) {
      setError(e as Error);
    } finally {
      setBusy(false);
    }
  }
  return (
    <form className="panel settings-card compact-retention" onSubmit={save}>
      <h3>Retention</h3>
      <p className="muted">
        Older automatic backups are removed after the next successful backup.
        Manual backups are kept until you remove them.
      </p>
      <div className="schedule-fields">
        <label>
          Automatic backups to keep
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
          {busy && <Busy />}Save settings
        </button>
      </div>
      {error && <ErrorState error={error} />}
    </form>
  );
}
