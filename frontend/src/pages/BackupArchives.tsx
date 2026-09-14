import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, Download, Plus, XCircle } from "lucide-react";
import { useState } from "react";
import { api, Busy, bytes, dateLabel, ErrorState, useApp } from "../lib";
import { queryKeys } from "../queryKeys";
import { invalidateResources } from "../queryInvalidation";

export function BackupArchives() {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [busy, setBusy] = useState(false);
  const archives = useQuery<{ records: any[]; failures: any[] }>({
    queryKey: queryKeys.backups(boot.profile!.id),
    queryFn: ({ signal }) => api("/backups", "GET", undefined, signal),
  });
  const rows = archives.data
    ? [
        ...archives.data.records.map((record) => ({ ...record, time: record.created_at, failed: !record.verified })),
        ...archives.data.failures.map((failure) => ({ ...failure, time: failure.started_at, failed: true })),
      ].sort((a, b) => b.time - a.time)
    : [];
  async function create() {
    setBusy(true);
    try {
      await api("/jobs/backup", "POST", {});
      notify("Backup started. The archive will appear here when ready.");
      await invalidateResources(cache, ["backups"]);
    } catch (error) {
      notify((error as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel settings-card backup-archives">
      <div className="section-heading">
        <div><h3><Archive size={19} />Backup archives</h3><p className="muted">Download a verified archive of your data and saved settings.</p></div>
        <button className="button primary" disabled={busy} onClick={create}>{busy ? <Busy /> : <Plus size={16} />}Create backup</button>
      </div>
      {archives.error && <ErrorState error={archives.error} retry={() => archives.refetch()} />}
      {archives.isPending && <Busy />}
      {rows.map((record) => (
        <div className={`backup-row ${record.failed ? "backup-failed" : ""}`} key={record.id}>
          {record.failed ? <XCircle size={18} /> : <Archive size={18} />}
          <span><strong>{record.filename || "Backup could not be completed"}</strong><small>{dateLabel(record.time)}{record.size ? ` · ${bytes(record.size)}` : ""} · {record.kind || record.trigger}</small></span>
          {record.failed ? <span className="badge failed">Failed</span> : <a className="button small" href={`/api/backups/${record.id}/download`} download><Download size={16} />Download</a>}
        </div>
      ))}
      {archives.data && !rows.length && <p className="muted">No backups yet.</p>}
      <p className="small-text muted">Archives include saved connection credentials. Keep downloads private. Restore an archive using the server's backup restore command.</p>
    </section>
  );
}
