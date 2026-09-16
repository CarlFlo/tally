import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Archive,
  Download,
  Plus,
  RotateCcw,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import {
  api,
  Busy,
  bytes,
  Confirm,
  dateLabel,
  ErrorState,
  useApp,
} from "../lib";
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
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [confirm, setConfirm] = useState<{
    action: "restore" | "delete";
    record: BackupRecord;
  } | null>(null);
  const archives = useQuery<{ records: BackupRecord[] }>({
    queryKey: queryKeys.backups(boot.profile!.id),
    queryFn: ({ signal }) => api("/backups", "GET", undefined, signal),
    staleTime: 0,
    refetchOnMount: "always",
    refetchOnWindowFocus: true,
  });
  const rows = archives.data?.records || [];

  async function create() {
    setBusy(true);
    try {
      await api<{ id: string }>("/backups", "POST", {});
      notify(t("backups.started"));
    } catch (error) {
      notify((error as Error).message, true);
    } finally {
      setBusy(false);
    }
  }

  async function restore(record: BackupRecord) {
    await api(`/backups/${record.id}/restore`, "POST", {});
    notify(t("backups.restored"));
    window.location.reload();
  }

  async function remove(record: BackupRecord) {
    await api(`/backups/${record.id}`, "DELETE", {});
    await invalidateResources(cache, ["backups"], 0);
    notify(t("backups.deleted"));
  }

  function restoreMessage(record: BackupRecord) {
    const version =
      record.app_version || t("backups.oldNoMetadata");
    const versionNote = record.different_version
      ? t("backups.restoreVersionNote", { backupVersion: version, currentVersion: boot.version })
      : "";
    return t("backups.restoreMessage", { filename: record.filename }) + versionNote;
  }

  return (
    <section className="panel settings-card backup-archives">
      <div className="section-heading">
        <div>
          <h3>
            <Archive size={19} />
            {t("backups.archives")}
          </h3>
          <p className="muted">
{t("backups.archivesHelp")}
          </p>
        </div>
        <div className="backup-actions">
          <button className="button primary" disabled={busy} onClick={create}>
            {busy ? <Busy /> : <Plus size={16} />}
            {t("backups.manualCreate")}
          </button>
        </div>
      </div>
      {archives.error && (
        <ErrorState error={archives.error} retry={() => archives.refetch()} />
      )}
      {archives.isPending && <Busy />}
      {rows.map((record) => {
        const kind =
          record.kind === "auto"
            ? t("backups.automatic")
            : record.kind === "imported"
              ? t("backups.imported")
              : t("backups.manual");
        return (
          <div className="backup-row" key={record.id}>
            <Archive size={18} />
            <div className="backup-main">
              <strong>{record.filename}</strong>
              <div className="backup-meta">
                <span>{dateLabel(record.created_at || 0)}</span>
                {record.size ? <span>{bytes(record.size)}</span> : null}
                <span className={`backup-kind ${record.kind || "manual"}`}>
                  {kind}
                </span>
                {record.app_version && <span>Tally {record.app_version}</span>}
                {record.legacy_version && <span>{t("backups.versionUnavailable")}</span>}
                {record.schema && <span>{t("backups.schema", { schema: record.schema })}</span>}
                {record.different_version && (
                  <span className="backup-version-note">
                    {t("backups.different", { version: boot.version })}
                  </span>
                )}
                {record.compatible === false && (
                  <span className="backup-version-note">
                    {t("backups.incompatible")}
                  </span>
                )}
                {record.archive_error && (
                  <span className="backup-version-note">
                    {record.archive_error}
                  </span>
                )}
              </div>
            </div>
            <div className="backup-actions">
              <a
                className="button small"
                href={`/api/backups/${record.id}/download`}
                download
              >
                <Download size={16} />
                {t("backups.download")}
              </a>
              <button
                className="button small"
                disabled={record.compatible === false}
                onClick={() => setConfirm({ action: "restore", record })}
              >
                <RotateCcw size={16} />
                {t("backups.restore")}
              </button>
              <button
                className="button small danger"
                onClick={() => setConfirm({ action: "delete", record })}
              >
                <Trash2 size={16} />
                {t("common.delete")}
              </button>
            </div>
          </div>
        );
      })}
      {archives.data && !rows.length && (
        <p className="muted">{t("backups.empty")}</p>
      )}
      <p className="small-text muted">
{t("backups.privateWarning")}
      </p>
      {confirm && (
        <Confirm
          title={
            confirm.action === "restore" ? t("backups.restoreTitle") : t("backups.deleteTitle")
          }
          message={
            confirm.action === "restore"
              ? restoreMessage(confirm.record)
              : t("backups.deleteMessage", { filename: confirm.record.filename })
          }
          onClose={() => setConfirm(null)}
          onConfirm={() =>
            confirm.action === "restore"
              ? restore(confirm.record)
              : remove(confirm.record)
          }
        />
      )}
    </section>
  );
}
