import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { api, Avatar, Busy, resetSession, useApp, type Profile } from "./lib";

export function LoginCredentials({ profile }: { profile: Profile }) {
  const { t } = useTranslation();
  const { notify } = useApp();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(event: FormEvent) {
    event.preventDefault();
    setBusy(true);
    try {
      await api("/auth/login", "POST", {
        profile: profile.id,
        password,
      });
      resetSession();
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <form className="login-form" onSubmit={submit}>
      <Avatar profile={profile} large />
      <label>
        {t("login.password")}
        <input
          type="password"
          autoComplete="current-password"
          autoFocus
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          maxLength={1024}
        />
      </label>
      <button className="button primary" disabled={busy}>
        {busy ? <Busy /> : <ArrowRight size={18} />}
        {t("login.enterSpace")}
      </button>
      <button
        className="text-button"
        type="button"
        onClick={() => notify(t("login.passwordResetHelp"))}
      >
        {t("login.forgotPassword")}
      </button>
      <NavLink className="text-button" to="/login">
        {t("login.chooseAnother")}
      </NavLink>
    </form>
  );
}
