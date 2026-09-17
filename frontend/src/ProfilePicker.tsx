import { Plus } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, NavLink, useNavigate, useParams } from "react-router-dom";
import { api, Avatar, resetSession, type Boot, type Profile } from "./lib";
import { LoginCredentials } from "./LoginCredentials";
import { Logo } from "./Logo";
import { useLocalization } from "./i18n";

type AuthProfile = Profile & {
  auth_method?: "password" | "none" | "oidc_unlinked";
};

function authMethod(profile: AuthProfile) {
  return profile.auth_method || (profile.has_password ? "password" : "none");
}

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
  const selected = boot.profiles.find((profile) => profile.id === profileId) as
    | AuthProfile
    | undefined;
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    previewLocale(selected?.locale || null);
    return () => previewLocale(null);
  }, [previewLocale, selected?.locale]);
  if (profileId && (!selected || authMethod(selected) !== "password"))
    return <Navigate to="/login" replace />;
  async function choose(profile: AuthProfile) {
    const method = authMethod(profile);
    if (method === "password") {
      navigate("/login/" + profile.id);
      return;
    }
    if (method === "oidc_unlinked") {
      notify(t("login.oidcUnlinked"), true);
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
            ? t("login.welcomeBack", { name: selected.display_name })
            : t("login.who")}
        </h1>
        <p className="muted">{t("login.subtitle")}</p>
        {selected ? (
          <LoginCredentials key={selected.id} profile={selected} />
        ) : (
          <div className="profile-grid">
            {(boot.profiles as AuthProfile[]).map((p) => (
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
      <span className="picker-footer">{t("login.footer")}</span>
    </div>
  );
}
