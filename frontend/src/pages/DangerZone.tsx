import { useState, type FormEvent } from "react";
import { ShieldCheck, Trash2 } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { api, Busy, Confirm, Dialog, resetSession, useApp } from "../lib";
import { invalidateResources } from "../queryInvalidation";

type SensitiveAction = "demote" | "delete";

export function DangerZone() {
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
    notify("Administrator access removed");
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
            Administrator access
          </h3>
          <p className="muted">
            Remove your own administrator permission. Another administrator
            must remain so the deployment cannot be locked out.
          </p>
          <button
            className="button danger"
            disabled={!canDemote}
            title={!canDemote ? "Promote another administrator first." : ""}
            onClick={requestDemotion}
          >
            Remove my administrator access
          </button>
        </section>
      )}
      <section className="panel settings-card danger-zone">
        <h3>Delete my account</h3>
        <p className="muted">
          Permanently delete your profile, preferences, follows, and episode
          progress. You will be signed out on every device. Other profiles keep
          their data.
        </p>
        {!canDelete && (
          <p className="muted small-text">
            Promote another profile to administrator before deleting this
            account.
          </p>
        )}
        <button
          className="button danger"
          disabled={!canDelete}
          onClick={requestDelete}
        >
          <Trash2 size={17} />
          Delete my account
        </button>
      </section>
      {confirmDemote && (
        <Confirm
          title="Remove your administrator access?"
          message="You will immediately lose access to deployment settings and administrator tools."
          onClose={() => setConfirmDemote(false)}
          onConfirm={() => demote()}
        />
      )}
      {confirmDelete && (
        <Confirm
          title="Permanently delete your account?"
          message="This cannot be undone. Your profile and all personal progress will be removed."
          onClose={() => setConfirmDelete(false)}
          onConfirm={() => remove()}
        />
      )}
      {reauth && (
        <Reauthenticate
          title={reauth === "demote" ? "Confirm administrator change" : "Confirm account deletion"}
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
          Re-enter your password to confirm this administrator action.
        </p>
        <label>
          Your password
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
            Cancel
          </button>
          <button className="button danger" disabled={busy}>
            {busy && <Busy />}Confirm
          </button>
        </div>
      </form>
    </Dialog>
  );
}
