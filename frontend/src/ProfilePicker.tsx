import { ArrowRight, Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, NavLink, useNavigate, useParams } from "react-router-dom";
import { api, Avatar, resetSession, type Boot, type Profile } from "./lib";
import { LoginCredentials } from "./LoginCredentials";
import { Logo } from "./Logo";
import { useLocalization } from "./i18n";

export function ProfilePicker({
  boot,
  notify,
}: {
  boot: Boot;
  notify: (s: string, e?: boolean) => void;
}) {
  const { t } = useTranslation();
  const { previewLocale } = useLocalization();
  const { profileId } = useParams();
  const navigate = useNavigate();
  const selected =
    boot.auth_mode === "local"
      ? boot.profiles.find((profile) => profile.id === profileId)
      : undefined;
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    previewLocale(selected?.locale || null);
    return () => previewLocale(null);
  }, [previewLocale, selected?.locale]);
  if (profileId && !selected) return <Navigate to="/login" replace />;
  async function choose(profile: Profile) {
    if (boot.auth_mode === "local") {
      navigate("/login/" + profile.id);
      return;
    }
    if (boot.auth_mode === "oidc") {
      window.location.assign("/auth/oidc/start");
      return;
    }
    setBusy(true);
    try {
      await api("/profiles/select", "POST", { profile: profile.id });
      resetSession();
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="picker-screen">
      <Logo />
      <div className="picker-content">
        <span className="eyebrow">{t("login.eyebrow")}</span>
        <h1>
          {selected
            ? selected.has_password
              ? t("login.welcomeBack", { name: selected.display_name })
              : t("login.setupPassword", { name: selected.display_name })
            : "Who's keeping up?"}
        </h1>
        <p className="muted">
          {t("login.subtitle")}
        </p>
        {selected ? (
          <LoginCredentials key={selected.id} profile={selected} />
        ) : boot.auth_mode === "oidc" ? (
          <a className="button primary" href="/auth/oidc/start">
            {t("profile.continueSSO")}
            <ArrowRight size={18} />
          </a>
        ) : (
          <div className="profile-grid">
            {boot.profiles.map((p) => (
              <button
                className="picker-profile"
                key={p.id}
                disabled={busy}
                onClick={() => choose(p)}
              >
                <Avatar profile={p} large />
                <strong>{p.display_name}</strong>
              </button>
            ))}
            <NavLink
              to="/login/new"
              className="picker-profile add-profile-tile"
            >
              <span className="add-profile-icon">
                <Plus size={36} />
              </span>
              <strong>{t("login.addProfile")}</strong>
            </NavLink>
          </div>
        )}
      </div>
      <span className="picker-footer">
        {t("login.footer")}
      </span>
    </div>
  );
}
