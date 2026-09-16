import { useEffect, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";
import { api, Avatar, Busy, resetSession, useApp } from "./lib";
import { useLocalization } from "./i18n";

const htmlColor = /^#[0-9A-Fa-f]{6}$/;

export function RegisterProfile() {
  const { t } = useTranslation();
  const { locales, previewLocale } = useLocalization();
  const { boot, notify } = useApp();
  const [name, setName] = useState("");
  const [avatar, setAvatar] = useState("mint");
  const [customColor, setCustomColor] = useState("");
  const [locale, setLocale] = useState("en");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const selectedAvatar = htmlColor.test(customColor) ? customColor : avatar;
  useEffect(() => () => previewLocale(null), [previewLocale]);
  async function submit(event: FormEvent) {
    event.preventDefault();
    if (password !== confirm) {
      notify(t("profile.passwordMismatch"), true);
      return;
    }
    setBusy(true);
    try {
      await api("/auth/register", "POST", {
        name,
        avatar: selectedAvatar,
        password,
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
        ) : boot.auth_mode === "oidc" ? (
          <a className="button primary" href="/auth/oidc/start">
            {t("profile.continueSSO")}
          </a>
        ) : (
          <form className="login-form" onSubmit={submit}>
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
            <div className="avatar-choices" aria-label={t("accessibility.chooseAvatar")}>
              {["mint", "violet", "amber", "rose", "blue"].map((color) => (
                <button
                  className={!customColor && avatar === color ? "selected" : ""}
                  type="button"
                  key={color}
                  aria-label={t("accessibility.avatar", { color })}
                  aria-pressed={!customColor && avatar === color}
                  onClick={() => {
                    setAvatar(color);
                    setCustomColor("");
                  }}
                >
                  <Avatar
                    profile={{
                      id: "",
                      display_name: name || t("profile.you"),
                      avatar: color,
                      locale,
                    }}
                  />
                </button>
              ))}
            </div>
            <label>
              {t("profile.customAvatarColor")}
              <input
                value={customColor}
                pattern="#[0-9A-Fa-f]{6}"
                maxLength={7}
                placeholder="#4F46E5"
                spellCheck={false}
                onChange={(e) => setCustomColor(e.target.value)}
              />
              <small className="muted">{t("profile.customAvatarHelp")}</small>
            </label>
            {customColor && htmlColor.test(customColor) && (
              <div className="profile-editor">
                <Avatar
                  profile={{
                    id: "",
                    display_name: name || "You",
                    avatar: customColor,
                    locale,
                  }}
                  large
                />
              </div>
            )}
            <label>
              {t("profile.language")}
              <select
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
                    {item.name}: {item.error || t("profile.localeUnavailable")}
                  </small>
                ))}
            </label>
            {boot.auth_mode === "local" && (
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
