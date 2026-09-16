import { useQueryClient } from "@tanstack/react-query";
import { PageHeader } from "../PageHeader";
import {
  Activity,
  Bell,
  Check,
  Database,
  Download,
  Globe,
  HardDrive,
  KeyRound,
  Laptop,
  Moon,
  Plus,
  ShieldCheck,
  Sun,
  Trash2,
  UserRound,
  X,
} from "lucide-react";
import { useEffect, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { displayLocale } from "../dateFormatting";
import { NavLink } from "react-router-dom";
import {
  api,
  Avatar,
  Busy,
  Confirm,
  dateLabel,
  Dialog,
  Empty,
  ErrorState,
  resetSession,
  useApp,
  useLocal,
  type Profile,
} from "../lib";
import { SchedulingBackupsSettings } from "./BackupSettings";
import { BellNotificationSettings } from "./BellNotificationSettings";
import { DangerZone } from "./DangerZone";
import {
  DebugSettings,
} from "./DeploymentSettings";
import { JackettSettings } from "./JackettSettings";
import { DownloaderSettings } from "./DownloaderSettings";
import { NotificationSettings } from "./NotificationSettings";
import { invalidateResources } from "../queryInvalidation";
import { queryKeys } from "../queryKeys";
import { useLocalization } from "../i18n";

function normalizeHexColor(value: string) {
  if (!value || value.startsWith("#")) return value;
  return `#${value}`;
}

export { JobsPage } from "./Jobs";

export function StatisticsPage() {
  const { t, i18n } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const requestLimit = boot.preferences.request_limit || 20,
    scanLimit = boot.preferences.scan_limit || 20;
  const stats = useLocal<any>(
    "statistics",
    `/statistics?request_limit=${requestLimit}&scan_limit=${scanLimit}`,
    true,
  );
  function limitControl(
    key: "request_limit" | "scan_limit",
    label: string,
    value: number,
  ) {
    return (
      <label className="row-limit">
        {t("statistics.show")}{" "}
        <select
          aria-label={label}
          value={value}
          onChange={async (e) => {
            try {
              await api("/preferences", "PATCH", {
                [key]: Number(e.target.value),
              });
              await invalidateResources(cache, ["bootstrap"]);
            } catch (e) {
              notify((e as Error).message, true);
            }
          }}
        >
          {[20, 50, 100].map((n) => (
            <option value={n} key={n}>
              {n}
            </option>
          ))}
        </select>
      </label>
    );
  }

  const { data } = stats;
  const sum = (key: string) =>
    data?.summary.reduce((n: number, p: any) => n + p[key], 0) || 0;
  const days = Array.from({ length: 30 }, (_, i) => {
    const d = new Date();
    d.setUTCDate(d.getUTCDate() - 29 + i);
    const key = d.toISOString().slice(0, 10);
    return (
      data?.daily.find((p: any) => p.day === key) || {
        day: key,
        requests: 0,
        avoided: 0,
      }
    );
  });
  const max = Math.max(1, ...days.map((d) => d.requests + d.avoided));
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">{t("statistics.eyebrow")}</span>
          <h1>
            {t("statistics.title")}<span className="accent">.</span>
          </h1>
        </div>
      </div>
      {stats.error && <ErrorState error={stats.error} />}
      <div className="stats-grid">
        {[
          {
            label: t("statistics.providerRequests"),
            value: sum("requests"),
            icon: <Globe size={21} />,
            cls: "purple",
          },
          {
            label: t("statistics.callsAvoided"),
            value: sum("avoided"),
            icon: <ShieldCheck size={21} />,
            cls: "mint",
          },
          {
            label: t("statistics.cacheHits"),
            value: sum("cache_hits") + sum("conditional_hits"),
            icon: <Database size={21} />,
            cls: "amber",
          },
          {
            label: t("statistics.failedRequests"),
            value: sum("failures"),
            icon: <Activity size={21} />,
            cls: "rose",
          },
        ].map((m) => (
          <div className="panel stat-card" key={m.label}>
            <span className={"metric-icon " + m.cls}>{m.icon}</span>
            <strong>{m.value.toLocaleString(displayLocale(i18n.resolvedLanguage))}</strong>
            <span>{m.label}</span>
            <small>{t("statistics.last30")}</small>
          </div>
        ))}
      </div>
      <section className="panel chart-panel">
        <div className="section-heading">
          <h3>{t("statistics.requestActivity")}</h3>
          <div className="legend">
            <span>
              <i className="legend-dot purple" />
              {t("statistics.requests")}
            </span>
            <span>
              <i className="legend-dot mint" />
              {t("statistics.avoided")}
            </span>
          </div>
        </div>
        <div
          className="bar-chart"
          role="img"
          aria-label={t("statistics.chartSummary", { requests: sum("requests"), avoided: sum("avoided") })}
        >
          {days.map((d) => (
            <div
              key={d.day}
              title={t("statistics.chartPoint", { day: d.day, requests: d.requests, avoided: d.avoided })}
            >
              <i
                className="chart-avoided"
                style={{ height: `${(d.avoided / max) * 100}%` }}
              />
              <i
                className="chart-requests"
                style={{ height: `${(d.requests / max) * 100}%` }}
              />
            </div>
          ))}
        </div>
        <div className="chart-axis">
          <span>{days[0].day}</span>
          <span>{t("statistics.today")}</span>
        </div>
      </section>
      <div className="section-heading run-heading">
        <h2>{t("statistics.providers")}</h2>
        <span className="muted small-text">{t("statistics.shared")}</span>
      </div>
      <div className="panel table-scroll">
        {data?.summary.length ? (
          <table>
            <thead>
              <tr>
                <th>{t("statistics.provider")}</th>
                <th>{t("common.status")}</th>
                <th>{t("statistics.successFailed")}</th>
                <th>{t("statistics.retries429")}</th>
                <th>{t("statistics.notModified")}</th>
                <th>{t("statistics.avgLatency")}</th>
                <th>{t("statistics.backoff")}</th>
              </tr>
            </thead>
            <tbody>
              {data.summary.map((p: any) => {
                const state = data.states.find(
                  (s: any) => s.provider === p.provider,
                );
                return (
                  <tr key={p.provider}>
                    <td>
                      <strong>{p.provider}</strong>
                    </td>
                    <td>
                      <span className={"badge " + (state?.state || "healthy")}>
                        {state?.state || t("statistics.healthy")}
                      </span>
                    </td>
                    <td>
                      {p.successes} / {p.failures}
                    </td>
                    <td>
                      {p.retries} / {p.rate_limited}
                    </td>
                    <td>{p.conditional_hits}</td>
                    <td>{p.average_latency} ms</td>
                    <td>
                      {state?.blocked_until > 0
                        ? dateLabel(state.blocked_until)
                        : "—"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        ) : (
          <Empty title={t("statistics.quietTitle")}>
            {t("statistics.quietHelp")}
          </Empty>
        )}
      </div>
      <div className="section-heading run-heading">
        <h2>{t("statistics.recent")}</h2>
        {limitControl("request_limit", t("statistics.recentRows"), requestLimit)}
      </div>
      <div className="panel table-scroll">
        {data?.requests.length ? (
          <table>
            <thead>
              <tr>
                <th>{t("statistics.provider")}</th>
                <th>{t("statistics.trigger")}</th>
                <th>{t("statistics.entity")}</th>
                <th>{t("statistics.outcome")}</th>
                <th>{t("statistics.http")}</th>
                <th>{t("statistics.latency")}</th>
                <th>{t("common.time")}</th>
              </tr>
            </thead>
            <tbody>
              {data.requests.map((r: any) => (
                <tr key={r.id}>
                  <td>{r.provider}</td>
                  <td>{r.trigger.replaceAll("_", " ")}</td>
                  <td className="mono truncate">{r.entity || "—"}</td>
                  <td>{r.reason}</td>
                  <td>{r.status_code || "—"}</td>
                  <td>{r.duration_ms} ms</td>
                  <td>{dateLabel(r.created_at)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <Empty title={t("statistics.emptyRequests")}>
            {t("statistics.emptyRequestsHelp")}
          </Empty>
        )}
      </div>
      {data && (
        <section className="panel next-scans">
          <div className="section-heading">
            <h3>{t("statistics.nextChecks")}</h3>
            {limitControl("scan_limit", t("statistics.nextRows"), scanLimit)}
          </div>
          {data.next_scans.map((s: any) => (
            <div className="setting-row" key={s.id}>
              <span>{s.name}</span>
              <span className="muted">{dateLabel(s.next_check_at)}</span>
            </div>
          ))}
        </section>
      )}
    </div>
  );
}

export function SettingsPage({
  tab = "deployment",
}: {
  tab?:
    | "personal"
    | "profiles"
    | "deployment"
    | "security"
    | "danger"
    | "notifications"
    | "bell"
    | "search"
    | "torrent"
    | "debug";
}) {
  const { t, i18n } = useTranslation();
  const { locales } = useLocalization();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const personal = tab === "personal" || tab === "security" || tab === "danger";
  const settings = useLocal<any>(
    "settings",
    "/settings",
    !personal,
  );
  const sessions = useLocal<any[]>(
    "sessions",
    "/auth/sessions",
    tab === "security",
  );
  const [name, setName] = useState(boot.profile!.display_name);
  const [locale, setLocale] = useState(boot.profile!.locale || "en");
  const [avatar, setAvatar] = useState(boot.profile!.avatar);
  const [customColor, setCustomColor] = useState(
    /^#[0-9A-Fa-f]{6}$/.test(boot.profile!.avatar) ? boot.profile!.avatar : "",
  );
  const [newProfile, setNewProfile] = useState(false);
  const [deleting, setDeleting] = useState<Profile | null>(null);
  const [adminConfirmation, setAdminConfirmation] = useState<{
    profile: Profile;
    isAdmin: boolean;
  } | null>(null);
  const [sensitiveAction, setSensitiveAction] = useState<{
    profile: Profile;
    kind: "demote" | "delete";
  } | null>(null);
  const adminCount = boot.profiles.filter((profile) => !!profile.is_admin).length;
  const [busy, setBusy] = useState(false);
  const [password, setPassword] = useState("");
  const [current, setCurrent] = useState("");
  useEffect(() => {
    setLocale(boot.profile!.locale || "en");
  }, [boot.profile!.locale]);
  async function prefs(key: string, value: any) {
    try {
      await api("/preferences", "PATCH", { [key]: value });
      await invalidateResources(cache, ["bootstrap"]);
      notify(t("settings.preferenceSaved"));
    } catch (e) {
      notify((e as Error).message, true);
    }
  }
  async function saveProfile(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api("/profile", "PATCH", {
        name,
        avatar: customColor || (avatar.endsWith(".png") ? "" : avatar),
        locale,
      });
      await invalidateResources(cache, ["bootstrap"]);
      notify(t("settings.profileUpdated"));
    } catch (e) {
      setLocale(boot.profile!.locale || "en");
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  async function setAdmin(profile: Profile, isAdmin: boolean, password = "") {
    await api("/profiles/" + profile.id + "/admin", "PATCH", {
      is_admin: isAdmin,
      password,
    });
    await invalidateResources(cache, ["bootstrap", "settings", "capabilities"]);
    notify(t(isAdmin ? "admin.granted" : "admin.removed"));
  }
  async function removeProfile(profile: Profile, password = "") {
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
      (current: typeof boot | undefined) =>
        current
          ? {
              ...current,
              profiles: current.profiles.filter(
                (item) => item.id !== profile.id,
              ),
            }
          : current,
    );
    notify(t("settings.profileDeleted"));
  }
  return (
    <div className="page settings-page">
      <PageHeader
        title={personal ? t("settings.myProfile") : t("settings.title")}
        eyebrow={personal ? t("settings.personalEyebrow") : t("settings.sharedEyebrow")}
        description={
          personal
            ? t("settings.personalDescription")
            : t("settings.sharedDescription")
        }
      />
      <nav
        className="settings-tabs"
        aria-label={personal ? t("settings.personalNav") : t("settings.deploymentNav")}
      >
        {(personal
          ? [
              {
                path: "/profile",
                label: t("settings.profilePreferences"),
                icon: <UserRound size={17} />,
              },
              {
                path: "/profile/security",
                label: t("settings.security"),
                icon: <ShieldCheck size={17} />,
              },
              {
                path: "/profile/danger",
                label: t("settings.dangerZone"),
                icon: <Trash2 size={17} />,
              },
            ]
          : [
              {
                path: "/settings",
                label: t("settings.schedulingBackups"),
                icon: <HardDrive size={17} />,
              },
              {
                path: "/settings/torrent",
                label: t("settings.torrentClient"),
                icon: <Download size={17} />,
              },
              {
                path: "/settings/search",
                label: t("settings.torrentSearch"),
                icon: <Globe size={17} />,
              },
              {
                path: "/settings/notifications",
                label: t("settings.notifications"),
                icon: <Bell size={17} />,
              },
              {
                path: "/settings/bell",
                label: t("settings.bell"),
                icon: <Bell size={17} />,
              },
              {
                path: "/settings/debug",
                label: t("settings.debug"),
                icon: <Activity size={17} />,
              },
              {
                path: "/settings/profiles",
                label: t("settings.profiles"),
                icon: <Laptop size={17} />,
              },
            ]
        ).map((t) => (
          <NavLink key={t.path} to={t.path} end>
            {t.icon}
            {t.label}
          </NavLink>
        ))}
      </nav>
      {tab === "torrent" && (
        <>
          {settings.data?.operator ? (
            <DownloaderSettings />
          ) : (
            <p className="muted">
{t("settings.ownerConnection")}
            </p>
          )}
        </>
      )}
      {(tab === "search" ||
        tab === "notifications" ||
        tab === "bell") &&
        (settings.data?.operator ? (
          tab === "notifications" ? (
            <NotificationSettings />
          ) : tab === "bell" ? (
            <BellNotificationSettings />
          ) : (
            <JackettSettings />
          )
        ) : (
          <p className="muted">
{t("settings.ownerOnly")}
          </p>
        ))}
      {tab === "debug" && <DebugSettings />}
      {tab === "danger" && <DangerZone />}
      {tab === "personal" && (
        <div className="settings-columns">
          <section className="panel settings-card">
            <h3>
              {t("settings.yourProfile")}
              {boot.profile!.is_admin && <> · {t("settings.adminAccount")}</>}
            </h3>
            <p className="muted">{t("settings.profileHelp")}</p>
            <form onSubmit={saveProfile}>
              <div className="profile-editor">
                <Avatar
                  profile={{
                    ...boot.profile!,
                    avatar: customColor || avatar,
                    display_name: name || "?",
                  }}
                  large
                />
                <div className="avatar-options">
                  {["violet", "mint", "amber", "rose", "blue", "peach"].map(
                    (color) => (
                      <button
                        key={color}
                        className={
                          "avatar-option avatar-" +
                          color +
                          (!customColor && avatar === color ? " selected" : "")
                        }
                        type="button"
                        aria-label={t("accessibility.avatar", { color })}
                        onClick={() => {
                          setAvatar(color);
                          setCustomColor("");
                        }}
                      >
                        {!customColor && avatar === color && <Check size={17} />}
                      </button>
                    ),
                  )}
                  <label className="avatar-upload">
                    {t("settings.uploadImage")}
                    <input
                      type="file"
                      accept="image/png,image/jpeg,image/webp"
                      onChange={async (e) => {
                        if (!e.target.files?.[0]) return;
                        const form = new FormData();
                        form.set("avatar", e.target.files[0]);
                        try {
                          const response = await api(
                            "/profile/avatar",
                            "POST",
                            form,
                          );
                          setAvatar(response.avatar);
                          setCustomColor("");
                          await invalidateResources(cache, ["bootstrap"]);
                          notify(t("settings.avatarUpdated"));
                        } catch (e) {
                          notify((e as Error).message, true);
                        }
                      }}
                    />
                  </label>
                </div>
              </div>
              <label>
                {t("profile.customAvatarColor")}
                <input
                  value={customColor}
                  pattern="#[0-9A-Fa-f]{6}"
                  maxLength={7}
                  placeholder="#4F46E5"
                  spellCheck={false}
                  onChange={(e) => setCustomColor(normalizeHexColor(e.target.value))}
                />
                <small className="muted">{t("profile.customAvatarHelp")}</small>
              </label>
              <label>
                {t("profile.displayName")}
                <input
                  maxLength={80}
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              <label>
                {t("profile.language")}
                <select
                  aria-label={t("profile.language")}
                  value={locale}
                  onChange={(event) => setLocale(event.target.value)}
                >
                  {!locales.some((item) => item.locale === locale) && (
                    <option value={locale} disabled>
                      {locale} — {t("common.unavailable")}
                    </option>
                  )}
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
                {!locales.some((item) => item.locale === locale) && (
                  <small className="muted">
                    {locale}: {t("language.missingFile")}
                  </small>
                )}
              </label>
              <button className="button primary" disabled={busy}>
                {busy && <Busy />}{t("settings.saveProfile")}
              </button>
            </form>
          </section>
          <section className="panel settings-card">
            <h3>{t("settings.comfort")}</h3>
            <p className="muted">{t("settings.comfortHelp")}</p>
            <label>{t("appearance.title")}</label>
            <div className="theme-options">
              {[
                { id: "system", icon: <Laptop size={21} />, label: t("appearance.system") },
                { id: "light", icon: <Sun size={21} />, label: t("appearance.light") },
                { id: "dark", icon: <Moon size={21} />, label: t("appearance.dark") },
              ].map((t) => (
                <button
                  key={t.id}
                  className={boot.preferences.theme === t.id ? "selected" : ""}
                  onClick={() => prefs("theme", t.id)}
                >
                  {t.icon}
                  {t.label}
                </button>
              ))}
            </div>
            <label>
              {t("settings.timezone")}
              <input
                key={boot.preferences.timezone}
                defaultValue={boot.preferences.timezone}
                list="timezones"
                onBlur={(e) => {
                  if (e.target.value !== boot.preferences.timezone)
                    void prefs("timezone", e.target.value);
                }}
              />
              <datalist id="timezones">
                {Intl.supportedValuesOf("timeZone").map((tz) => (
                  <option key={tz} value={tz} />
                ))}
              </datalist>
            </label>
            <button
              className="text-button"
              onClick={() =>
                prefs(
                  "timezone",
                  Intl.DateTimeFormat().resolvedOptions().timeZone,
                )
              }
            >
{t("settings.useBrowserTimezone")}
            </button>
            <div className="two-fields">
              <label>
                {t("settings.weekStarts")}
                <select
                  value={boot.preferences.week_start}
                  onChange={(e) => prefs("week_start", Number(e.target.value))}
                >
                  <option value={1}>{t("settings.monday")}</option>
                  <option value={0}>{t("settings.sunday")}</option>
                </select>
              </label>
              <label>
                {t("settings.timeFormat")}
                <select
                  value={boot.preferences.time_format}
                  onChange={(e) => prefs("time_format", e.target.value)}
                >
                  <option value="24h">{t("settings.hour24")}</option>
                  <option value="12h">{t("settings.hour12")}</option>
                </select>
              </label>
            </div>
            <label>
              {t("settings.dateFormat")}
              <select
                value={boot.preferences.date_format}
                onChange={(e) => prefs("date_format", e.target.value)}
              >
                <option value="d MMM yyyy">
                  {new Intl.DateTimeFormat(displayLocale(i18n.resolvedLanguage), {
                    day: "numeric",
                    month: "short",
                    year: "numeric",
                    timeZone: "UTC",
                  }).format(new Date("2026-09-11T12:00:00Z"))}
                </option>
                <option value="yyyy-MM-dd">2026-09-11</option>
                <option value="MM/dd/yyyy">09/11/2026</option>
              </select>
            </label>
          </section>
        </div>
      )}
      {tab === "profiles" && (
        <section className="panel settings-card">
          <div className="section-heading">
            <div>
              <h3>{t("settings.everyoneSpace")}</h3>
              <p className="muted">
{t("settings.usedProfiles", { used: boot.profiles.length, max: boot.max_profiles })}
              </p>
            </div>
            {settings.data?.operator && boot.auth_mode !== "oidc" && (
              <button
                className="button primary"
                disabled={boot.profiles.length >= boot.max_profiles}
                onClick={() => setNewProfile(true)}
              >
                <Plus size={17} />
                {t("settings.newProfile")}
              </button>
            )}
          </div>
          <div className="profile-settings-list">
            {boot.profiles.map((p) => {
              const isAdmin = !!p.is_admin;
              const canDemote = !isAdmin || adminCount > 1;
              const canDelete =
                !isAdmin || boot.profiles.length === 1 || adminCount > 1;
              return (
                <div key={p.id}>
                  <Avatar profile={p} />
                  <span>
                    <strong>{p.display_name}</strong>
                    <small>
                      {t(isAdmin ? "settings.roleAdmin" : "settings.roleUser")}
                      {p.id === boot.profile?.id ? t("settings.currentSuffix") : ""}
                    </small>
                  </span>
                  <span className="profile-role-actions">
                    <button
                      className="button small"
                      disabled={isAdmin && !canDemote}
                      title={
                        isAdmin && !canDemote
                          ? t("admin.promoteFirst")
                          : ""
                      }
                      onClick={() =>
                        setAdminConfirmation({ profile: p, isAdmin: !isAdmin })
                      }
                    >
                      <ShieldCheck size={16} />
                      {t(isAdmin ? "settings.removeAdmin" : "settings.makeAdmin")}
                    </button>
                    <button
                      className="button small danger"
                      aria-label={t("settings.deleteProfile", { name: p.display_name })}
                      disabled={!canDelete}
                      title={!canDelete ? t("admin.promoteFirst") : ""}
                      onClick={() => {
                        if (isAdmin && boot.auth_mode === "local") {
                          setSensitiveAction({ profile: p, kind: "delete" });
                        } else {
                          setDeleting(p);
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
          <p className="muted small-text">
            {t("admin.signOutSwitch")}
          </p>
        </section>
      )}
      {tab === "deployment" && <SchedulingBackupsSettings />}
      {tab === "security" && (
        <div className="settings-columns">
          <section className="panel settings-card">
            <h3>
              <KeyRound size={19} />
              {t("settings.authentication")}
            </h3>
            {boot.auth_mode === "local" ? (
              <form
                onSubmit={async (e) => {
                  e.preventDefault();
                  setBusy(true);
                  try {
                    await api("/auth/password", "POST", { current, password });
                    setCurrent("");
                    setPassword("");
                    await invalidateResources(cache, ["bootstrap", "sessions"]);
                    notify(t("settings.passwordChanged"));
                  } catch (e) {
                    notify((e as Error).message, true);
                  } finally {
                    setBusy(false);
                  }
                }}
              >
                <p className="muted">
                  {t("settings.passwordChangeHelp")}
                </p>
                <label>
                  {t("settings.currentPassword")}
                  <input
                    type="password"
                    autoComplete="current-password"
                    required
                    value={current}
                    onChange={(e) => setCurrent(e.target.value)}
                  />
                </label>
                <label>
                  {t("settings.newPasswordOrPin")}
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
                <button className="button primary" disabled={busy}>
                  {busy && <Busy />}{t("settings.changePassword")}
                </button>
              </form>
            ) : (
              <p className="muted">
                {boot.auth_mode === "disabled"
                  ? t("settings.authDisabled")
                  : t("settings.identityManaged")}
              </p>
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
                    <strong>
                      {session.current ? t("settings.thisBrowser") : t("settings.browserSession")}
                    </strong>
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
                        else
                          await invalidateResources(cache, [
                            "bootstrap",
                            "sessions",
                          ]);
                        notify(t("settings.sessionRevoked"));
                      } catch (e) {
                        notify((e as Error).message, true);
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
      )}
      {newProfile && <CreateProfile onClose={() => setNewProfile(false)} />}{" "}
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
          title={
            adminConfirmation.isAdmin
              ? t("admin.grantTitle")
              : t("admin.removeTitle")
          }
          message={
            adminConfirmation.isAdmin
              ? t("admin.grantMessage", { name: adminConfirmation.profile.display_name })
              : t("admin.removeMessage", { name: adminConfirmation.profile.display_name })
          }
          onClose={() => setAdminConfirmation(null)}
          onConfirm={async () => {
            if (adminConfirmation.isAdmin) {
              await setAdmin(adminConfirmation.profile, true);
            } else if (boot.auth_mode === "local") {
              setSensitiveAction({
                profile: adminConfirmation.profile,
                kind: "demote",
              });
            } else {
              await setAdmin(adminConfirmation.profile, false);
            }
          }}
        />
      )}
      {sensitiveAction && (
        <AdminReauthDialog
          title={
            sensitiveAction.kind === "demote"
              ? t("admin.confirmChange")
              : t("admin.confirmDelete")
          }
          onClose={() => setSensitiveAction(null)}
          onConfirm={async (password) => {
            if (sensitiveAction.kind === "demote") {
              await setAdmin(sensitiveAction.profile, false, password);
            } else {
              await removeProfile(sensitiveAction.profile, password);
            }
            setSensitiveAction(null);
          }}
        />
      )}
    </div>
  );
}
function CreateProfile({ onClose }: { onClose: () => void }) {
  const { t } = useTranslation();
  const { locales, previewLocale } = useLocalization();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [name, setName] = useState("");
  const [avatar] = useState(
    () =>
      ["mint", "amber", "rose", "blue", "peach"][
        boot.profiles.length % 5
      ],
  );
  const [customColor, setCustomColor] = useState("");
  const [locale, setLocale] = useState("en");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  useEffect(() => {
    previewLocale("en");
    return () => previewLocale(null);
  }, [previewLocale]);
  return (
    <Dialog title={t("profile.newSpace")} onClose={onClose}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await api("/profiles", "POST", {
              name,
              avatar: customColor || avatar,
              password,
              locale,
            });
            await invalidateResources(cache, ["bootstrap"]);
            notify(t("profile.created"));
            onClose();
          } catch (e) {
            notify((e as Error).message, true);
          } finally {
            setBusy(false);
          }
        }}
      >
        <label>
          {t("profile.displayName")}
          <input
            autoFocus
            required
            maxLength={80}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        <label>
          {t("profile.customAvatarColor")}
          <input
            value={customColor}
            pattern="#[0-9A-Fa-f]{6}"
            maxLength={7}
            placeholder="#4F46E5"
            spellCheck={false}
            onChange={(e) => setCustomColor(normalizeHexColor(e.target.value))}
          />
          <small className="muted">{t("profile.customAvatarHelp")}</small>
        </label>
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
        {boot.auth_mode === "local" && (
          <label>
            {t("profile.passwordOptional")}
            <input
              type="password"
              autoComplete="new-password"
              minLength={boot.password_min}
              maxLength={boot.password_max}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>
        )}
        <div className="dialog-actions">
          <button type="button" className="button" onClick={onClose}>
            {t("common.cancel")}
          </button>
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
        <p className="muted">
          {t("admin.reauth")}
        </p>
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
          <button type="button" className="button" onClick={onClose}>
            {t("common.cancel")}
          </button>
          <button className="button danger" disabled={busy}>
            {busy && <Busy />}{t("common.confirm")}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
