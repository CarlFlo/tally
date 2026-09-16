import { Sparkles } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api, Busy, resetSession, SignOutButton } from "./lib";
import { Logo } from "./Logo";

export function PasswordGate({
  notify,
}: {
  notify: (s: string, e?: boolean) => void;
}) {
  const { t } = useTranslation();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <div className="picker-screen">
      <Logo />
      <div className="password-gate">
        <Sparkles size={30} />
        <h1>{t("passwordGate.title")}</h1>
        <p className="muted">{t("passwordGate.temporary")}</p>
        <form
          onSubmit={async (e) => {
            e.preventDefault();
            setBusy(true);
            try {
              await api("/auth/password", "POST", { password });
              resetSession();
            } catch (e) {
              notify((e as Error).message, true);
            } finally {
              setBusy(false);
            }
          }}
        >
          <label>
            {t("passwordGate.newPasswordOrPin")}
            <input
              type="password"
              autoComplete="new-password"
              required
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>
          <button className="button primary" disabled={busy}>
            {busy && <Busy />}{t("passwordGate.setNewPassword")}
          </button>
        </form>
        <SignOutButton className="text-button" />
      </div>
    </div>
  );
}
