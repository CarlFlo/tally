import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { ShieldCheck, Trash2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { api, Busy, Confirm, Dialog, resetSession, useApp } from "../lib";
import { invalidateResources } from "../queryInvalidation";

type SensitiveAction = "demote" | "delete";

export function DangerZone() {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const profile = boot.profile!;
  const adminCount = boot.profiles.filter((item) => !!item.is_admin).length;
  const isAdmin = !!profile.is_admin;
  const canDemote = isAdmin && adminCount > 1;
  const canDelete =
    !isAdmin || boot.profiles.length === 1 || adminCount > 1;
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [confirmDemote, setConfirmDemote] = useState(false);
  const [reauth, setReauth] = useState<SensitiveAction | null>(null);

  async function demote(password = "") {
    await api(`/profiles/${profile.id}/admin`, "PATCH", {
      is_admin: false,
      password,
    });
    await invalidateResources(cache, ["bootstrap", "settings", "capabilities"]);
    notify(t("danger.adminRemoved"));
  }

  async function remove(password = "") {
    await api(`/profiles/${profile.id}`, "DELETE", password ? { password } : undefined);
    resetSession("/login");
  }

  function requestDemotion() {
    if (!canDemote) return;
    if (boot.auth_mode === "local") setReauth("demote");
    else setConfirmDemote(true);
  }

  function requestDelete() {
    if (!canDelete) return;
    if (isAdmin && boot.auth_mode === "local") setReauth("delete");
    else setConfirmDelete(true);
  }

  return (
    <div className="settings-columns">
      {isAdmin && (
        <section className="panel settings-card danger-zone">
          <h3>
            <ShieldCheck size={19} />
            {t("danger.adminTitle")}
          </h3>
          <p className="muted">
{t("danger.adminHelp")}
          </p>
          <button
            className="button danger"
            disabled={!canDemote}
            title={!canDemote ? t("admin.promoteFirst") : ""}
            onClick={requestDemotion}
          >
            {t("danger.removeOwnAdmin")}
          </button>
        </section>
      )}
      <section className="panel settings-card danger-zone">
        <h3>{t("danger.deleteAccount")}</h3>
        <p className="muted">
{t("danger.deleteHelp")}
        </p>
        {!canDelete && (
          <p className="muted small-text">
{t("danger.promoteBeforeDelete")}
          </p>
        )}
        <button
          className="button danger"
          disabled={!canDelete}
          onClick={requestDelete}
        >
          <Trash2 size={17} />
          {t("danger.deleteAccount")}
        </button>
      </section>
      {confirmDemote && (
        <Confirm
          title={t("danger.removeAdminTitle")}
          message={t("danger.removeAdminMessage")}
          onClose={() => setConfirmDemote(false)}
          onConfirm={() => demote()}
        />
      )}
      {confirmDelete && (
        <Confirm
          title={t("danger.deleteTitle")}
          message={t("danger.deleteMessage")}
          onClose={() => setConfirmDelete(false)}
          onConfirm={() => remove()}
        />
      )}
      {reauth && (
        <Reauthenticate
          title={reauth === "demote" ? t("admin.confirmChange") : t("danger.confirmAccountDelete")}
          onClose={() => setReauth(null)}
          onConfirm={async (password) => {
            if (reauth === "demote") await demote(password);
            else await remove(password);
            setReauth(null);
          }}
        />
      )}
    </div>
  );
}

function Reauthenticate({
  title,
  onClose,
  onConfirm,
}: {
  title: string;
  onClose: () => void;
  onConfirm: (password: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const { notify } = useApp();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      await onConfirm(password);
    } catch (error) {
      notify((error as Error).message, true);
      setBusy(false);
    }
  }
  return (
    <Dialog title={title} onClose={onClose}>
      <form onSubmit={submit}>
        <p className="muted">
{t("admin.reauth")}
        </p>
        <label>
          {t("admin.yourPassword")}
          <input
            data-autofocus
            required
            type="password"
            autoComplete="current-password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </label>
        <div className="dialog-actions">
          <button type="button" className="button" onClick={onClose}>
            {t("common.cancel")}
          </button>
          <button className="button danger" disabled={busy}>
            {busy && <Busy />}{t("common.confirm")}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
