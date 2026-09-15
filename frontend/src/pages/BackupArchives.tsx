import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Archive, Download, Plus, RotateCcw, Trash2 } from "lucide-react";
import { useState } from "react";
import { api, Busy, bytes, Confirm, dateLabel, ErrorState, useApp } from "../lib";
import { queryKeys } from "../queryKeys";
import { invalidateResources } from "../queryInvalidation";

type BackupRecord = {
  id: string;
  filename: string;
  kind?: string;
  size?: number;
  created_at?: number;
  schema?: number;
  app_version?: string;
  legacy_version?: boolean;
  different_version?: boolean;
  compatible?: boolean;
  archive_error?: string;
};

export function BackupArchives() {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState<{ action: "restore" | "delete"; record: BackupRecord } | null>(null);
  const archives = useQuery<{ records: BackupRecord[] }>({
    queryKey: queryKeys.backups(boot.profile!.id),
    queryFn: ({ signal }) => api("/backups", "GET", undefined, signal),
  });
  const rows = archives.data?.records || [];

  async function waitForBackupJob(id: string) {
    const deadline = Date.now() + 60_000;
    while (Date.now() < deadline) {
      const jobs = await api<any>("/jobs?kind=backup&status=all");
      const run = jobs.runs?.find((item: any) => item.id === id);
      if (run && run.status !== "running") {
        if (run.status !== "success")
          throw new Error(run.error || "Backup could not be completed");
        return;
      }
      await new Promise((resolve) => window.setTimeout(resolve, 500));
    }
    throw new Error("Backup is still running. Check Jobs for its status.");
  }

  async function create() {
    setBusy(true);
    try {
      const started = await api<{ id: string }>("/backups", "POST", {});
      notify("Backup started.");
      await waitForBackupJob(started.id);
      await invalidateResources(cache, ["backups", "jobs"]);
      notify("Backup created");
    } catch (error) {
      notify((error as Error).message, true);
    } finally {
      setBusy(false);
    }
  }

  async function restore(record: BackupRecord) {
    await api(`/backups/${record.id}/restore`, "POST", {});
    notify("Backup restored. Reloading Tally.");
    window.location.reload();
  }

  async function remove(record: BackupRecord) {
    await api(`/backups/${record.id}`, "DELETE", {});
    await invalidateResources(cache, ["backups"], 0);
    notify("Backup deleted");
  }

  function restoreMessage(record: BackupRecord) {
    const version = record.app_version || "an older version without version metadata";
    const versionNote = record.different_version
      ? ` This backup was created by Tally ${version}; the current version is ${boot.version}. Compatible database migrations will be applied before restore.`
      : "";
    return `Restore ${record.filename}? Current Tally data and saved settings will be replaced only after the backup is fully validated.${versionNote}`;
  }

  return (
    <section className="panel settings-card backup-archives">
      <div className="section-heading">
        <div><h3><Archive size={19} />Backup archives</h3><p className="muted">Download, restore, or remove verified snapshots of your data and saved settings.</p></div>
        <button className="button primary" disabled={busy} onClick={create}>{busy ? <Busy /> : <Plus size={16} />}Create manual backup</button>
      </div>
      {archives.error && <ErrorState error={archives.error} retry={() => archives.refetch()} />}
      {archives.isPending && <Busy />}
      {rows.map((record) => {
        const kind = record.kind === "auto" ? "Automatic" : "Manual";
        return (
          <div className="backup-row" key={record.id}>
            <Archive size={18} />
            <div className="backup-main">
              <strong>{record.filename}</strong>
              <div className="backup-meta">
                <span>{dateLabel(record.created_at || 0)}</span>
                {record.size ? <span>{bytes(record.size)}</span> : null}
                <span className={`backup-kind ${record.kind || "manual"}`}>{kind}</span>
                {record.app_version && <span>Tally {record.app_version}</span>}
                {record.legacy_version && <span>Version unavailable</span>}
                {record.schema && <span>Schema v{record.schema}</span>}
                {record.different_version && <span className="backup-version-note">Different from Tally {boot.version}</span>}
                {record.compatible === false && <span className="backup-version-note">Incompatible backup</span>}
                {record.archive_error && <span className="backup-version-note">{record.archive_error}</span>}
              </div>
            </div>
            <div className="backup-actions">
              <a className="button small" href={`/api/backups/${record.id}/download`} download><Download size={16} />Download</a>
              <button className="button small" disabled={record.compatible === false} onClick={() => setConfirm({ action: "restore", record })}><RotateCcw size={16} />Restore</button>
              <button className="button small danger" onClick={() => setConfirm({ action: "delete", record })}><Trash2 size={16} />Delete</button>
            </div>
          </div>
        );
      })}
      {archives.data && !rows.length && <p className="muted">No backups yet.</p>}
      <p className="small-text muted">Archives include saved connection credentials. Keep downloads private. Backup failures appear in Notifications and Logs. Restore validates and migrates an archive before replacing current data.</p>
      {confirm && (
        <Confirm
          title={confirm.action === "restore" ? "Restore backup" : "Delete backup"}
          message={confirm.action === "restore" ? restoreMessage(confirm.record) : `Delete ${confirm.record.filename}? This backup archive cannot be recovered after deletion.`}
          onClose={() => setConfirm(null)}
          onConfirm={() => confirm.action === "restore" ? restore(confirm.record) : remove(confirm.record)}
        />
      )}
    </section>
  );
}
