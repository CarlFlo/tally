import { useState } from "react";
import { Trash2 } from "lucide-react";
import { api, Confirm, resetSession, useApp } from "../lib";
export function DangerZone() {
  const { boot } = useApp(),
    [confirm, setConfirm] = useState(false);
  return (
    <section className="panel settings-card danger-zone">
      <h3>Delete my account</h3>
      {boot.profile?.id === "user0" ? (
        <p className="muted">
          This is the permanent administrator account. It cannot be deleted.
        </p>
      ) : (
        <>
          <p className="muted">
            Permanently delete your profile, preferences, follows, and episode
            progress. You will be signed out on every device. Other profiles
            keep their data.
          </p>
          <button className="button danger" onClick={() => setConfirm(true)}>
            <Trash2 size={17} />
            Delete my account
          </button>
          {confirm && (
            <Confirm
              title="Permanently delete your account?"
              message="This cannot be undone. Your profile and all personal progress will be removed."
              onClose={() => setConfirm(false)}
              onConfirm={async () => {
                await api(`/profiles/${boot.profile!.id}`, "DELETE");
                resetSession("/login");
              }}
            />
          )}
        </>
      )}
    </section>
  );
}
