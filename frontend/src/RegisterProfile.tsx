import { useEffect, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";
import { api, Busy, resetSession, useApp } from "./lib";
import { useLocalization } from "./i18n";
import {
  ProfileAvatarChoices,
  ProfileAvatarPreview,
  isValidHexColor,
} from "./ProfileCreateAvatar";

type AuthMethod = "password" | "none";

export function RegisterProfile() {
  const { t } = useTranslation();
  const { locales, previewLocale } = useLocalization();
  const { boot, notify } = useApp();
  const [name, setName] = useState("");
  const [avatar, setAvatar] = useState("mint");
  const [customColor, setCustomColor] = useState("");
  const [locale, setLocale] = useState("en");
  const [authMethod, setAuthMethod] = useState<AuthMethod>("password");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const selectedAvatar = isValidHexColor(customColor) ? customColor : avatar;

  useEffect(() => {
    previewLocale(locale);
    return () => previewLocale(null);
    // The initial new-profile locale is intentionally previewed before save.
    // Subsequent select changes call previewLocale directly for immediate feedback.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [previewLocale]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (authMethod === "password" && password !== confirm) {
      notify(t("profile.passwordMismatch"), true);
      return;
    }
    setBusy(true);
    try {
      await api("/auth/register", "POST", {
        name,
        avatar: selectedAvatar,
        auth_method: authMethod,
        password: authMethod === "password" ? password : "",
        locale,
      });
      resetSession();
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="picker-screen">
      <div className="picker-content">
        <span className="eyebrow">{t("brand.workspace")}</span>
        <h1>
          {t("profile.create")}<span className="accent">.</span>
        </h1>
        {boot.profiles.length >= boot.max_profiles ? (
          <p className="muted">{t("profile.spacesFull")}</p>
        ) : (
          <form className="login-form profile-create-form" onSubmit={submit}>
            <ProfileAvatarPreview
              name={name}
              avatar={avatar}
              customColor={customColor}
              locale={locale}
            />
            <label>
              {t("profile.displayName")}
              <input
                autoFocus
                required
                maxLength={80}
                autoComplete="nickname"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </label>
            <ProfileAvatarChoices
              name={name}
              avatar={avatar}
              customColor={customColor}
              locale={locale}
              onAvatarChange={setAvatar}
              onCustomColorChange={setCustomColor}
            />
            <label>
              {t("profile.language")}
              <select
                aria-label={t("profile.language")}
                value={locale}
                onChange={(event) => {
                  const next = event.target.value;
                  setLocale(next);
                  previewLocale(next);
                }}
              >
                {locales.map((item) => (
                  <option
                    key={item.locale}
                    value={item.locale}
                    disabled={!item.valid}
                  >
                    {item.name}{item.valid ? "" : ` — ${t("common.unavailable")}`}
                  </option>
                ))}
              </select>
              <small className="muted">{t("profile.languageHelp")}</small>
              {locales
                .filter((item) => !item.valid)
                .map((item) => (
                  <small className="muted" key={item.locale}>
                    {item.name}: {item.error_code ? t(item.error_code, { defaultValue: item.error }) : item.error || t("profile.localeUnavailable")}
                  </small>
                ))}
            </label>
            <label>
              {t("profile.authentication")}
              <select
                value={authMethod}
                onChange={(event) => setAuthMethod(event.target.value as AuthMethod)}
              >
                <option value="password">{t("profile.authPassword")}</option>
                <option value="none">{t("profile.authNone")}</option>
              </select>
            </label>
            {authMethod === "none" ? (
              <p className="callout auth-none-warning">{t("profile.authNoneWarning")}</p>
            ) : (
              <>
                <label>
                  {t("profile.newPassword")}
                  <input
                    type="password"
                    autoComplete="new-password"
                    required
                    minLength={boot.password_min}
                    maxLength={boot.password_max}
                    value={password}
                    onChange={(e) => setPassword(e.target.value)}
                  />
                </label>
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
                <p className="small-text muted">
                  {t("profile.passwordHint", {
                    min: boot.password_min,
                    max: boot.password_max,
                  })}
                </p>
              </>
            )}
            <button className="button primary" disabled={busy}>
              {busy && <Busy />}{t("profile.create")}
            </button>
          </form>
        )}
        <NavLink className="text-button" to="/login">
          {t("profile.backToProfiles")}
        </NavLink>
      </div>
    </div>
  );
}
