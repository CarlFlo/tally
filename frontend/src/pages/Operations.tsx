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

export { JobsPage } from "./Jobs";

export function StatisticsPage() {
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
        Show{" "}
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
          <span className="eyebrow">LESS GUESSWORK. MORE VISIBILITY.</span>
          <h1>
            Statistics<span className="accent">.</span>
          </h1>
        </div>
      </div>
      {stats.error && <ErrorState error={stats.error} />}
      <div className="stats-grid">
        {[
          {
            label: "Provider requests",
            value: sum("requests"),
            icon: <Globe size={21} />,
            cls: "purple",
          },
          {
            label: "Calls avoided",
            value: sum("avoided"),
            icon: <ShieldCheck size={21} />,
            cls: "mint",
          },
          {
            label: "Cache hits",
            value: sum("cache_hits") + sum("conditional_hits"),
            icon: <Database size={21} />,
            cls: "amber",
          },
          {
            label: "Failed requests",
            value: sum("failures"),
            icon: <Activity size={21} />,
            cls: "rose",
          },
        ].map((m) => (
          <div className="panel stat-card" key={m.label}>
            <span className={"metric-icon " + m.cls}>{m.icon}</span>
            <strong>{m.value.toLocaleString()}</strong>
            <span>{m.label}</span>
            <small>Last 30 days</small>
          </div>
        ))}
      </div>
      <section className="panel chart-panel">
        <div className="section-heading">
          <h3>Request activity</h3>
          <div className="legend">
            <span>
              <i className="legend-dot purple" />
              Requests
            </span>
            <span>
              <i className="legend-dot mint" />
              Avoided
            </span>
          </div>
        </div>
        <div
          className="bar-chart"
          role="img"
          aria-label={`Last 30 days: ${sum("requests")} requests and ${sum("avoided")} calls avoided`}
        >
          {days.map((d) => (
            <div
              key={d.day}
              title={`${d.day}: ${d.requests} requests, ${d.avoided} avoided`}
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
          <span>Today</span>
        </div>
      </section>
      <div className="section-heading run-heading">
        <h2>Providers</h2>
        <span className="muted small-text">Shared across all profiles</span>
      </div>
      <div className="panel table-scroll">
        {data?.summary.length ? (
          <table>
            <thead>
              <tr>
                <th>Provider</th>
                <th>Status</th>
                <th>Success / failed</th>
                <th>Retries / 429s</th>
                <th>304s</th>
                <th>Avg. latency</th>
                <th>Backoff until</th>
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
                        {state?.state || "healthy"}
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
          <Empty title="A quiet start.">
            Provider statistics appear after your first search or sync.
          </Empty>
        )}
      </div>
      <div className="section-heading run-heading">
        <h2>Recent requests</h2>
        {limitControl("request_limit", "Recent requests rows", requestLimit)}
      </div>
      <div className="panel table-scroll">
        {data?.requests.length ? (
          <table>
            <thead>
              <tr>
                <th>Provider</th>
                <th>Trigger</th>
                <th>Entity</th>
                <th>Outcome</th>
                <th>HTTP</th>
                <th>Latency</th>
                <th>Time</th>
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
          <Empty title="Nothing to report yet.">
            Local calendar browsing doesn't make provider requests.
          </Empty>
        )}
      </div>
      {data && (
        <section className="panel next-scans">
          <div className="section-heading">
            <h3>Next metadata checks</h3>
            {limitControl("scan_limit", "Next metadata checks rows", scanLimit)}
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
  const [avatar, setAvatar] = useState(boot.profile!.avatar);
  const [newProfile, setNewProfile] = useState(false);
  const [deleting, setDeleting] = useState<Profile | null>(null);
  const [busy, setBusy] = useState(false);
  const [password, setPassword] = useState("");
  const [current, setCurrent] = useState("");
  async function prefs(key: string, value: any) {
    try {
      await api("/preferences", "PATCH", { [key]: value });
      await invalidateResources(cache, ["bootstrap"]);
      notify("Preference saved");
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
        avatar: avatar.endsWith(".png") ? "" : avatar,
      });
      await invalidateResources(cache, ["bootstrap"]);
      notify("Profile updated");
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="page settings-page">
      <PageHeader
        title={personal ? "My profile" : "Settings"}
        eyebrow={personal ? "JUST THE WAY YOU LIKE IT" : "YOUR SHARED SPACE"}
        description={
          personal
            ? "Your profile, preferences, and account."
            : "Manage profiles and your Tally deployment."
        }
      />
      <nav
        className="settings-tabs"
        aria-label={personal ? "Personal settings" : "Deployment settings"}
      >
        {(personal
          ? [
              {
                path: "/profile",
                label: "Profile & preferences",
                icon: <UserRound size={17} />,
              },
              {
                path: "/profile/security",
                label: "Security",
                icon: <ShieldCheck size={17} />,
              },
              {
                path: "/profile/danger",
                label: "Danger zone",
                icon: <Trash2 size={17} />,
              },
            ]
          : [
              {
                path: "/settings",
                label: "Scheduling & backups",
                icon: <HardDrive size={17} />,
              },
              {
                path: "/settings/torrent",
                label: "Torrent client",
                icon: <Download size={17} />,
              },
              {
                path: "/settings/search",
                label: "Torrent search",
                icon: <Globe size={17} />,
              },
              {
                path: "/settings/notifications",
                label: "Notifications",
                icon: <Bell size={17} />,
              },
              {
                path: "/settings/bell",
                label: "Bell Notifications",
                icon: <Bell size={17} />,
              },
              {
                path: "/settings/debug",
                label: "Debug",
                icon: <Activity size={17} />,
              },
              {
                path: "/settings/profiles",
                label: "Profiles",
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
              The deployment owner manages this connection.
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
            Only the deployment owner can change these settings.
          </p>
        ))}
      {tab === "debug" && <DebugSettings />}
      {tab === "danger" && <DangerZone />}
      {tab === "personal" && (
        <div className="settings-columns">
          <section className="panel settings-card">
            <h3>Your profile</h3>
            <p className="muted">A familiar face in your own little space.</p>
            <form onSubmit={saveProfile}>
              <div className="profile-editor">
                <Avatar
                  profile={{
                    ...boot.profile!,
                    avatar,
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
                          (avatar === color ? " selected" : "")
                        }
                        type="button"
                        aria-label={color + " avatar"}
                        onClick={() => setAvatar(color)}
                      >
                        {avatar === color && <Check size={17} />}
                      </button>
                    ),
                  )}
                  <label className="avatar-upload">
                    Upload image
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
                          await invalidateResources(cache, ["bootstrap"]);
                          notify("Avatar updated");
                        } catch (e) {
                          notify((e as Error).message, true);
                        }
                      }}
                    />
                  </label>
                </div>
              </div>
              <label>
                Display name
                <input
                  maxLength={80}
                  required
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                />
              </label>
              <p className="small-text muted">
                {boot.profile!.id === "user0"
                  ? "Administrator account"
                  : "Your personal account"}
              </p>
              <button className="button primary" disabled={busy}>
                {busy && <Busy />}Save profile
              </button>
            </form>
          </section>
          <section className="panel settings-card">
            <h3>Make yourself comfortable</h3>
            <p className="muted">These choices follow your profile.</p>
            <label>Appearance</label>
            <div className="theme-options">
              {[
                { id: "system", icon: <Laptop size={21} />, label: "System" },
                { id: "light", icon: <Sun size={21} />, label: "Light" },
                { id: "dark", icon: <Moon size={21} />, label: "Dark" },
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
              Timezone
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
              Use this browser's timezone
            </button>
            <div className="two-fields">
              <label>
                Week starts
                <select
                  value={boot.preferences.week_start}
                  onChange={(e) => prefs("week_start", Number(e.target.value))}
                >
                  <option value={1}>Monday</option>
                  <option value={0}>Sunday</option>
                </select>
              </label>
              <label>
                Time format
                <select
                  value={boot.preferences.time_format}
                  onChange={(e) => prefs("time_format", e.target.value)}
                >
                  <option value="24h">24-hour</option>
                  <option value="12h">12-hour</option>
                </select>
              </label>
            </div>
            <label>
              Date format
              <select
                value={boot.preferences.date_format}
                onChange={(e) => prefs("date_format", e.target.value)}
              >
                <option value="d MMM yyyy">11 Sep 2026</option>
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
              <h3>Everyone gets their own space</h3>
              <p className="muted">
                {boot.profiles.length} of {boot.max_profiles} profiles used.
              </p>
            </div>
            {settings.data?.operator && boot.auth_mode !== "oidc" && (
              <button
                className="button primary"
                disabled={boot.profiles.length >= boot.max_profiles}
                onClick={() => setNewProfile(true)}
              >
                <Plus size={17} />
                New profile
              </button>
            )}
          </div>
          <div className="profile-settings-list">
            {boot.profiles.map((p) => (
              <div key={p.id}>
                <Avatar profile={p} />
                <span>
                  <strong>{p.display_name}</strong>
                  <small>
                    {p.id === "user0" ? "admin" : ""}
                    {p.id === boot.profile?.id
                      ? p.id === "user0"
                        ? " · Current profile"
                        : "Current profile"
                      : ""}
                  </small>
                </span>
                {p.id === "user0" ? (
                  <span className="badge">Permanent</span>
                ) : (
                  (boot.auth_mode === "disabled" ||
                    p.id === boot.profile!.id) && (
                    <button
                      className="icon-button"
                      aria-label={"Delete " + p.display_name}
                      onClick={() => setDeleting(p)}
                    >
                      <Trash2 size={17} />
                    </button>
                  )
                )}
              </div>
            ))}
          </div>
          <p className="muted small-text">
            To use another profile, sign out first.
          </p>
        </section>
      )}
      {tab === "deployment" && <SchedulingBackupsSettings />}
      {tab === "security" && (
        <div className="settings-columns">
          <section className="panel settings-card">
            <h3>
              <KeyRound size={19} />
              Authentication
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
                    notify("Password changed; other sessions revoked");
                  } catch (e) {
                    notify((e as Error).message, true);
                  } finally {
                    setBusy(false);
                  }
                }}
              >
                <p className="muted">
                  Changing your password signs out your other sessions.
                </p>
                <label>
                  Current password
                  <input
                    type="password"
                    autoComplete="current-password"
                    required
                    value={current}
                    onChange={(e) => setCurrent(e.target.value)}
                  />
                </label>
                <label>
                  New password or PIN
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
                  {busy && <Busy />}Change password
                </button>
              </form>
            ) : (
              <p className="muted">
                {boot.auth_mode === "disabled"
                  ? "Authentication is disabled. Profiles are convenient personal spaces for a trusted network. Enable local or OIDC authentication through your deployment environment."
                  : "Your identity provider manages sign-in and credentials."}
              </p>
            )}
          </section>
          <section className="panel settings-card">
            <h3>
              <ShieldCheck size={19} />
              Your sessions
            </h3>
            {sessions.data?.length ? (
              sessions.data.map((session) => (
                <div className="session-row" key={session.id}>
                  <Laptop size={21} />
                  <div>
                    <strong>
                      {session.current ? "This browser" : "Browser session"}
                    </strong>
                    <small>{session.user_agent.slice(0, 90)}</small>
                    <small>Last active {dateLabel(session.last_seen)}</small>
                  </div>
                  <button
                    className="icon-button"
                    aria-label="Revoke session"
                    onClick={async () => {
                      try {
                        await api("/auth/sessions/" + session.id, "DELETE");
                        if (session.current) resetSession();
                        else
                          await invalidateResources(cache, [
                            "bootstrap",
                            "sessions",
                          ]);
                        notify("Session revoked");
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
              <p className="muted">No authenticated sessions.</p>
            )}
          </section>
        </div>
      )}
      {newProfile && <CreateProfile onClose={() => setNewProfile(false)} />}{" "}
      {deleting && (
        <Confirm
          title={"Delete " + deleting.display_name + "?"}
          message="This permanently removes this profile, preferences, follows, and episode progress. Other profiles keep their data."
          onClose={() => setDeleting(null)}
          onConfirm={async () => {
            await api("/profiles/" + deleting.id, "DELETE");
            resetSession();
            notify("Profile deleted");
          }}
        />
      )}
    </div>
  );
}
function CreateProfile({ onClose }: { onClose: () => void }) {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [name, setName] = useState("");
  const [password, setPassword] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <Dialog title="A new personal space" onClose={onClose}>
      <form
        onSubmit={async (e) => {
          e.preventDefault();
          setBusy(true);
          try {
            await api("/profiles", "POST", {
              name,
              avatar: ["mint", "amber", "rose", "blue", "peach"][
                boot.profiles.length % 5
              ],
              password,
            });
            await invalidateResources(cache, ["bootstrap"]);
            notify("Profile created");
            onClose();
          } catch (e) {
            notify((e as Error).message, true);
          } finally {
            setBusy(false);
          }
        }}
      >
        <label>
          Display name
          <input
            autoFocus
            required
            maxLength={80}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </label>
        {boot.auth_mode === "local" && (
          <label>
            Password (optional)
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
            Cancel
          </button>
          <button className="button primary" disabled={busy}>
            {busy && <Busy />}Create profile
          </button>
        </div>
      </form>
    </Dialog>
  );
}
