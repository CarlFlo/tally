import { useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  Bell,
  Download,
  Globe,
  HardDrive,
  KeyRound,
  Laptop,
  Plus,
  ShieldCheck,
  Trash2,
} from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { NavLink } from "react-router-dom";
import { useLocalization } from "../i18n";
import {
  api,
  Avatar,
  Busy,
  Confirm,
  Dialog,
  resetSession,
  useApp,
  type Profile,
} from "../lib";
import { PageHeader } from "../PageHeader";
import {
  ProfileAvatarChoices,
  ProfileAvatarPreview,
  isValidHexColor,
} from "../ProfileCreateAvatar";
import { invalidateResources } from "../queryInvalidation";
import { queryKeys } from "../queryKeys";

type AuthMethod = "password" | "none";
type AuthProfile = Profile & { auth_method?: AuthMethod };

type SensitiveAction = {
  profile: AuthProfile;
  kind: "demote" | "delete";
};

function profileAuth(profile: AuthProfile): AuthMethod {
  return profile.auth_method || (profile.has_password ? "password" : "none");
}

function DeploymentSettingsNav() {
  const { t } = useTranslation();
  const tabs = [
    { path: "/admin/configuration/schedules", label: t("settings.schedules"), icon: <HardDrive size={17} /> },
    { path: "/admin/configuration/backups", label: t("settings.backups"), icon: <HardDrive size={17} /> },
    { path: "/admin/configuration/integrations/downloader", label: t("settings.torrentClient"), icon: <Download size={17} /> },
    { path: "/admin/configuration/integrations/search", label: t("settings.torrentSearch"), icon: <Globe size={17} /> },
    { path: "/admin/configuration/delivery", label: t("settings.notifications"), icon: <Bell size={17} /> },
    { path: "/admin/advanced/diagnostics", label: t("settings.debug"), icon: <Activity size={17} /> },
    { path: "/admin/access/profiles", label: t("settings.profiles"), icon: <Laptop size={17} /> },
  ];
  return (
    <nav className="settings-tabs" aria-label={t("settings.deploymentNav")}>
      {tabs.map((tab) => (
        <NavLink key={tab.path} to={tab.path} end>
          {tab.icon}
          {tab.label}
        </NavLink>
      ))}
    </nav>
  );
}

export function ProfilesSettingsPage() {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const profiles = boot.profiles as AuthProfile[];
  const current = boot.profile as AuthProfile;
  const currentProtected = profileAuth(current) === "password";
  const adminCount = profiles.filter((profile) => !!profile.is_admin).length;
  const [newProfile, setNewProfile] = useState(false);
  const [authProfile, setAuthProfile] = useState<AuthProfile | null>(null);
  const [deleting, setDeleting] = useState<AuthProfile | null>(null);
  const [adminConfirmation, setAdminConfirmation] = useState<{
    profile: AuthProfile;
    isAdmin: boolean;
  } | null>(null);
  const [sensitiveAction, setSensitiveAction] = useState<SensitiveAction | null>(null);

  async function refresh() {
    await invalidateResources(cache, ["bootstrap", "settings", "capabilities"]);
  }

  async function setAdmin(profile: AuthProfile, isAdmin: boolean, password = "") {
    await api("/profiles/" + profile.id + "/admin", "PATCH", {
      is_admin: isAdmin,
      password,
    });
    await refresh();
    notify(t(isAdmin ? "admin.granted" : "admin.removed"));
  }

  async function removeProfile(profile: AuthProfile, password = "") {
    await api(
      "/profiles/" + profile.id,
      "DELETE",
      password ? { password } : undefined,
    );
    if (profile.id === boot.profile?.id) {
      resetSession("/login");
      return;
    }
    cache.setQueryData(
      queryKeys.bootstrap(),
      (value: typeof boot | undefined) =>
        value
          ? { ...value, profiles: value.profiles.filter((item) => item.id !== profile.id) }
          : value,
    );
    notify(t("settings.profileDeleted"));
  }

  function authLabel(profile: AuthProfile) {
    return profileAuth(profile) === "password"
      ? t("profile.passwordProtected")
      : t("profile.authNone");
  }

  return (
    <div className="page settings-page">
      <PageHeader
        title={t("settings.title")}
        description={t("settings.sharedDescription")}
      />
      <DeploymentSettingsNav />
      <section className="panel settings-card">
        <div className="section-heading">
          <div>
            <h3>{t("settings.everyoneSpace")}</h3>
            <p className="muted">
              {t("settings.usedProfiles", { used: profiles.length, max: boot.max_profiles })}
            </p>
          </div>
          <button
            className="button primary"
            disabled={profiles.length >= boot.max_profiles}
            onClick={() => setNewProfile(true)}
          >
            <Plus size={17} />
            {t("settings.newProfile")}
          </button>
        </div>
        <div className="profile-settings-list">
          {profiles.map((profile) => {
            const isAdmin = !!profile.is_admin;
            const canDemote = !isAdmin || adminCount > 1;
            const canDelete = !isAdmin || profiles.length === 1 || adminCount > 1;
            return (
              <div key={profile.id}>
                <Avatar profile={profile} />
                <span>
                  <strong>{profile.display_name}</strong>
                  <small>
                    {t(isAdmin ? "settings.roleAdmin" : "settings.roleUser")}
                    {profile.id === boot.profile?.id ? t("settings.currentSuffix") : ""}
                    {" · "}{authLabel(profile)}
                  </small>
                </span>
                <span className="profile-role-actions">
                  <button className="button small" onClick={() => setAuthProfile(profile)}>
                    <KeyRound size={16} />
                    {t("profile.authentication")}
                  </button>
                  <button
                    className="button small"
                    disabled={isAdmin && !canDemote}
                    title={isAdmin && !canDemote ? t("admin.promoteFirst") : ""}
                    onClick={() => setAdminConfirmation({ profile, isAdmin: !isAdmin })}
                  >
                    <ShieldCheck size={16} />
                    {t(isAdmin ? "settings.removeAdmin" : "settings.makeAdmin")}
                  </button>
                  <button
                    className="button small danger"
                    aria-label={t("settings.deleteProfile", { name: profile.display_name })}
                    disabled={!canDelete}
                    title={!canDelete ? t("admin.promoteFirst") : ""}
                    onClick={() => {
                      if (isAdmin && currentProtected) {
                        setSensitiveAction({ profile, kind: "delete" });
                      } else {
                        setDeleting(profile);
                      }
                    }}
                  >
                    <Trash2 size={16} />
                    {t("common.delete")}
                  </button>
                </span>
              </div>
            );
          })}
        </div>
        <p className="muted small-text">{t("admin.signOutSwitch")}</p>
      </section>

      {newProfile && <CreateProfileDialog onClose={() => setNewProfile(false)} />}
      {authProfile && (
        <AuthenticationDialog
          profile={authProfile}
          requireActorPassword={currentProtected}
          onClose={() => setAuthProfile(null)}
          onSaved={async () => {
            setAuthProfile(null);
            await refresh();
            notify(t("profile.authenticationUpdated"));
          }}
        />
      )}
      {deleting && (
        <Confirm
          title={t("admin.deleteTitle", { name: deleting.display_name })}
          message={t("admin.deleteMessage")}
          onClose={() => setDeleting(null)}
          onConfirm={() => removeProfile(deleting)}
        />
      )}
      {adminConfirmation && (
        <Confirm
          title={adminConfirmation.isAdmin ? t("admin.grantTitle") : t("admin.removeTitle")}
          message={
            adminConfirmation.isAdmin
              ? t("admin.grantMessage", { name: adminConfirmation.profile.display_name })
              : t("admin.removeMessage", { name: adminConfirmation.profile.display_name })
          }
          onClose={() => setAdminConfirmation(null)}
          onConfirm={async () => {
            const pending = adminConfirmation;
            setAdminConfirmation(null);
            if (pending.isAdmin) {
              await setAdmin(pending.profile, true);
            } else if (currentProtected) {
              setSensitiveAction({ profile: pending.profile, kind: "demote" });
            } else {
              await setAdmin(pending.profile, false);
            }
          }}
        />
      )}
      {sensitiveAction && (
        <AdminReauthDialog
          title={sensitiveAction.kind === "demote" ? t("admin.confirmChange") : t("admin.confirmDelete")}
          onClose={() => setSensitiveAction(null)}
          onConfirm={async (password) => {
            if (sensitiveAction.kind === "demote")
              await setAdmin(sensitiveAction.profile, false, password);
            else await removeProfile(sensitiveAction.profile, password);
            setSensitiveAction(null);
          }}
        />
      )}
    </div>
  );
}

function AuthenticationDialog({
  profile,
  requireActorPassword,
  onClose,
  onSaved,
}: {
  profile: AuthProfile;
  requireActorPassword: boolean;
  onClose: () => void;
  onSaved: () => Promise<void>;
}) {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const existing = profileAuth(profile);
  const [method, setMethod] = useState<AuthMethod>(existing);
  const [password, setPassword] = useState("");
  const [actorPassword, setActorPassword] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <Dialog
      title={t("profile.authenticationFor", { name: profile.display_name })}
      onClose={onClose}
    >
      <form
        onSubmit={async (event) => {
          event.preventDefault();
          setBusy(true);
          try {
            await api(`/profiles/${profile.id}/authentication`, "PATCH", {
              method,
              password: method === "password" ? password : "",
              actor_password: actorPassword,
            });
            await onSaved();
          } catch (error) {
            notify((error as Error).message, true);
            setBusy(false);
          }
        }}
      >
        <label>
          {t("profile.authentication")}
          <select value={method} onChange={(event) => setMethod(event.target.value as AuthMethod)}>
            <option value="password">{t("profile.authPassword")}</option>
            <option value="none">{t("profile.authNone")}</option>
          </select>
        </label>
        {method === "password" ? (
          <label>
            {t("profile.newPassword")}
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
        ) : (
          <p className="callout auth-none-warning">{t("profile.authNoneWarning")}</p>
        )}
        {requireActorPassword && (
          <label>
            {t("admin.yourPassword")}
            <input
              type="password"
              autoComplete="current-password"
              required
              value={actorPassword}
              onChange={(event) => setActorPassword(event.target.value)}
            />
          </label>
        )}
        <div className="dialog-actions">
          <button type="button" className="button" onClick={onClose}>{t("common.cancel")}</button>
          <button className="button primary" disabled={busy}>
            {busy && <Busy />}{t("common.save")}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

function CreateProfileDialog({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const { locales, previewLocale } = useLocalization();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [name, setName] = useState("");
  const [avatar, setAvatar] = useState(
    () => ["mint", "amber", "rose", "blue", "peach"][boot.profiles.length % 5],
  );
  const [customColor, setCustomColor] = useState("");
  const [locale, setLocale] = useState("en");
  const [method, setMethod] = useState<AuthMethod>("password");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [busy, setBusy] = useState(false);
  const selectedAvatar = isValidHexColor(customColor) ? customColor : avatar;

  useEffect(() => {
    previewLocale("en");
    return () => previewLocale(null);
  }, [previewLocale]);

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (method === "password" && password !== confirm) {
      notify(t("profile.passwordMismatch"), true);
      return;
    }
    setBusy(true);
    try {
      await api("/profiles", "POST", {
        name,
        avatar: selectedAvatar,
        locale,
        auth_method: method,
        password: method === "password" ? password : "",
      });
      await invalidateResources(cache, ["bootstrap"]);
      notify(t("profile.created"));
      onClose();
    } catch (error) {
      notify((error as Error).message, true);
      setBusy(false);
    }
  }

  return (
    <Dialog title={t("profile.newSpace")} onClose={onClose}>
      <form className="login-form profile-create-form profile-create-dialog-form" onSubmit={submit}>
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
            onChange={(event) => setName(event.target.value)}
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
              <option key={item.locale} value={item.locale} disabled={!item.valid}>
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
          <select value={method} onChange={(event) => setMethod(event.target.value as AuthMethod)}>
            <option value="password">{t("profile.authPassword")}</option>
            <option value="none">{t("profile.authNone")}</option>
          </select>
        </label>
        {method === "password" ? (
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
                onChange={(event) => setPassword(event.target.value)}
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
                onChange={(event) => setConfirm(event.target.value)}
              />
            </label>
            <p className="small-text muted">
              {t("profile.passwordHint", {
                min: boot.password_min,
                max: boot.password_max,
              })}
            </p>
          </>
        ) : (
          <p className="callout auth-none-warning">{t("profile.authNoneWarning")}</p>
        )}
        <div className="dialog-actions">
          <button type="button" className="button" onClick={onClose}>{t("common.cancel")}</button>
          <button className="button primary" disabled={busy}>
            {busy && <Busy />}{t("profile.create")}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

function AdminReauthDialog({
  title,
  onClose,
  onConfirm,
}: {
  title: string;
  onClose: () => void;
  onConfirm: (password: string) => Promise<void>;
}) {
  const { t } = useTranslation();
  const { notify } = useApp();
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <Dialog title={title} onClose={onClose}>
      <form
        onSubmit={async (event) => {
          event.preventDefault();
          setBusy(true);
          try {
            await onConfirm(password);
          } catch (error) {
            notify((error as Error).message, true);
            setBusy(false);
          }
        }}
      >
        <p className="muted">{t("admin.reauth")}</p>
        <label>
          {t("admin.yourPassword")}
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
          <button type="button" className="button" onClick={onClose}>{t("common.cancel")}</button>
          <button className="button danger" disabled={busy}>
            {busy && <Busy />}{t("common.confirm")}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
