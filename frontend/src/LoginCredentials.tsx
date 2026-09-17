import { useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { api, Avatar, Busy, resetSession, useApp, type Profile } from "./lib";

export function LoginCredentials({ profile }: { profile: Profile }) {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const setup = !profile.has_password;
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (setup && password !== confirm) {
      notify(t("profile.passwordMismatch"), true);
      return;
    }
    setBusy(true);
    try {
      await api(setup ? "/auth/setup" : "/auth/login", "POST", {
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
      {setup && (
        <p className="muted">
          {t("login.choosePassword")}
        </p>
      )}
      <label>
        {setup ? t("profile.newPassword") : t("login.password")}
        <input
          type="password"
          autoComplete={setup ? "new-password" : "current-password"}
          autoFocus
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          minLength={setup ? boot.password_min : undefined}
          maxLength={setup ? boot.password_max : 1024}
        />
      </label>
      {setup && (
        <label>
          {t("profile.confirmPassword")}
          <input
            type="password"
            autoComplete="new-password"
            required
            maxLength={boot.password_max}
            value={confirm}
            onChange={(e) => setConfirm(e.target.value)}
          />
        </label>
      )}
      <button className="button primary" disabled={busy}>
        {busy ? <Busy /> : <ArrowRight size={18} />}
        {setup ? t("login.setPasswordContinue") : t("login.enterSpace")}
      </button>
      {!setup && (
        <button
          className="text-button"
          type="button"
          onClick={async () => {
            try {
              const result = await api("/auth/recover", "POST", {
                profile: profile.id,
              });
              notify(result.message);
            } catch (e) {
              notify((e as Error).message, true);
            }
          }}
        >
          {t("login.forgotPassword")}
        </button>
      )}
      <NavLink className="text-button" to="/login">
        {t("login.chooseAnother")}
      </NavLink>
    </form>
  );
}
