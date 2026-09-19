import { useQueryClient } from "@tanstack/react-query";
import { KeyRound, Laptop, ShieldCheck, Trash2, UserRound, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";
import { api, Busy, dateLabel, resetSession, useApp, useLocal, type Profile } from "../lib";
import { PageHeader } from "../PageHeader";
import { invalidateResources } from "../queryInvalidation";

type AuthProfile = Profile & {
  auth_method?: "password" | "none";
};

function profileAuth(profile: AuthProfile) {
  return profile.auth_method || (profile.has_password ? "password" : "none");
}

export function ProfileSecurityPage() {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const sessions = useLocal<any[]>("sessions", "/auth/sessions", true);
  const profile = boot.profile as AuthProfile;
  const method = profileAuth(profile);
  const [busy, setBusy] = useState(false);
  const [password, setPassword] = useState("");
  const [current, setCurrent] = useState("");

  return (
    <div className="page settings-page">
      <PageHeader
        title={t("settings.myProfile")}
        description={t("settings.personalDescription")}
      />
      <nav className="settings-tabs" aria-label={t("settings.personalNav")}>
        <NavLink to="/account" end>
          <UserRound size={17} />
          {t("settings.profilePreferences")}
        </NavLink>
        <NavLink to="/account/security" end>
          <ShieldCheck size={17} />
          {t("settings.security")}
        </NavLink>
        <NavLink to="/account/danger" end>
          <Trash2 size={17} />
          {t("settings.dangerZone")}
        </NavLink>
      </nav>
      <div className="settings-columns">
        <section className="panel settings-card">
          <h3>
            <KeyRound size={19} />
            {t("settings.authentication")}
          </h3>
          {method === "password" ? (
            <form
              onSubmit={async (event) => {
                event.preventDefault();
                setBusy(true);
                try {
                  await api("/auth/password", "POST", { current, password });
                  setCurrent("");
                  setPassword("");
                  await invalidateResources(cache, ["bootstrap", "sessions"]);
                  notify(t("settings.passwordChanged"));
                } catch (error) {
                  notify((error as Error).message, true);
                } finally {
                  setBusy(false);
                }
              }}
            >
              <p className="muted">{t("settings.passwordChangeHelp")}</p>
              <label>
                {t("settings.currentPassword")}
                <input
                  type="password"
                  autoComplete="current-password"
                  required
                  value={current}
                  onChange={(event) => setCurrent(event.target.value)}
                />
              </label>
              <label>
                {t("settings.newPassword")}
                <input
                  type="password"
                  autoComplete="new-password"
                  required
                  minLength={boot.password_min}
                  maxLength={boot.password_max}
                  value={password}
                  onChange={(event) => setPassword(event.target.value)}
                />
              </label>
              <button className="button primary" disabled={busy}>
                {busy && <Busy />}{t("settings.changePassword")}
              </button>
            </form>
          ) : (
            <>
              <p><strong>{t("profile.authNone", { defaultValue: "No authentication" })}</strong></p>
              <p className="muted">
                {t("profile.authNoneStatus", {
                  defaultValue: "This profile does not require a password. An administrator can change its authentication method from Profiles.",
                })}
              </p>
              {!!profile.is_admin && (
                <NavLink className="button" to="/admin/access/profiles">
                  {t("profile.manageAuthentication", { defaultValue: "Manage authentication" })}
                </NavLink>
              )}
            </>
          )}
        </section>
        <section className="panel settings-card">
          <h3>
            <ShieldCheck size={19} />
            {t("settings.sessionsTitle")}
          </h3>
          {sessions.data?.length ? (
            sessions.data.map((session) => (
              <div className="session-row" key={session.id}>
                <Laptop size={21} />
                <div>
                  <strong>{session.current ? t("settings.thisBrowser") : t("settings.browserSession")}</strong>
                  <small>{session.user_agent.slice(0, 90)}</small>
                  <small>{t("settings.lastActive", { date: dateLabel(session.last_seen) })}</small>
                </div>
                <button
                  className="icon-button"
                  aria-label={t("settings.revokeSession")}
                  onClick={async () => {
                    try {
                      await api("/auth/sessions/" + session.id, "DELETE");
                      if (session.current) resetSession();
                      else await invalidateResources(cache, ["bootstrap", "sessions"]);
                      notify(t("settings.sessionRevoked"));
                    } catch (error) {
                      notify((error as Error).message, true);
                    }
                  }}
                >
                  <X size={17} />
                </button>
              </div>
            ))
          ) : (
            <p className="muted">{t("settings.noSessions")}</p>
          )}
        </section>
      </div>
    </div>
  );
}
