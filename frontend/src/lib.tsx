import { QueryClient, useQuery, useQueryClient } from "@tanstack/react-query";
import { dateTimeFormatter, displayLocale } from "./dateFormatting";
import { requestPool, RequestPoolOverloadError } from "./requestPool";
import { queryKeys } from "./queryKeys";
import { invalidateResources } from "./queryInvalidation";
import { i18n } from "./i18n";
import {
  ArrowUpRight,
  Check,
  Download,
  LoaderCircle,
  LogOut,
  Search,
  Tv,
  X,
} from "lucide-react";
import {
  createContext,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useNavigate } from "react-router-dom";

export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1, refetchOnWindowFocus: false },
    mutations: { retry: false },
  },
});
export async function api<T = any>(
  path: string,
  method = "GET",
  body?: unknown,
  signal?: AbortSignal,
): Promise<T> {
  const deadline = new AbortController();
  const timer = window.setTimeout(
    () => deadline.abort(new DOMException(i18n.t("errors.requestTimedOut"), "TimeoutError")),
    90_000,
  );
  const combined = signal
    ? AbortSignal.any([signal, deadline.signal])
    : deadline.signal;
  try {
    return await requestPool.run(combined, async () => {
      const form = body instanceof FormData;
      const response = await fetch("/api" + path, {
        method,
        credentials: "same-origin",
        signal: combined,
        headers: {
          ...(method !== "GET" ? { "X-Tally-CSRF": "1" } : {}),
          ...(!form && body !== undefined
            ? { "Content-Type": "application/json" }
            : {}),
        },
        body: body === undefined ? undefined : form ? body : JSON.stringify(body),
      });
      const data = await response.json();
      combined.throwIfAborted();
      if (!response.ok) {
        if (response.status === 401 || data.code === "password_change_required")
          void invalidateResources(queryClient, ["bootstrap"]);
        const fallback = data.error || i18n.t("errors.requestFailed", { status: response.status });
        const message = data.code
          ? i18n.t("errors." + data.code, { defaultValue: fallback })
          : fallback;
        throw new Error(message);
      }
      return data;
    });
  } catch (error) {
    if (error instanceof RequestPoolOverloadError)
      throw new Error(
        i18n.t("errors.tooManyRequests", {
          defaultValue: "Too many pending requests. Please try again.",
        }),
      );
    throw error;
  } finally {
    window.clearTimeout(timer);
  }
}
export type Profile = {
  id: string;
  display_name: string;
  avatar: string;
  locale: string;
  has_password?: boolean | number;
  is_admin?: boolean | number;
};
export type Prefs = {
  theme: string;
  timezone: string;
  calendar_view: string;
  week_start: number;
  time_format: string;
  date_format: string;
  debug_mode: boolean;
  debug_job_state: string;
  request_limit: number;
  scan_limit: number;
  job_type_filter: string;
  job_status_filter: string;
  bell_categories: string[];
};
export type Boot = {
  version: string;
  browser_theme: string;
  profiles: Profile[];
  profile: Profile | null;
  preferences: Prefs;
  preferences_initialized: boolean;
  auth_mode: string;
  restricted: boolean;
  warning: string;
  max_profiles: number;
  password_min: number;
  password_max: number;
};
export function resetSession(destination = "/calendar") {
  queryClient.clear();
  try {
    // Other tabs share cookies; discard their previous profile's cached UI too.
    localStorage.setItem(
      "tally-session-change",
      `${Date.now()}-${Math.random()}`,
    );
  } catch {
    // Sign-out must still work when browser storage is unavailable.
  }
  window.location.replace(destination);
}
export type Show = {
  favorite: number;
  id: string;
  name: string;
  summary: string;
  image: string;
  status: string;
  premiered: string;
  network: string;
  genres: string;
  rating: number;
  runtime: number;
  episode_count: number;
  watched_count: number;
  aired_count: number;
  aired_unwatched: number;
  next_episode: string;
  last_checked_at: number;
  next_check_at: number;
};
export type Episode = {
  favorite: number;
  id: string;
  show_id: string;
  show_name: string;
  show_image: string;
  name: string;
  summary: string;
  season: number;
  season_episode_count?: number;
  number: number;
  airdate: string;
  airstamp: string;
  runtime: number;
  watched: number | boolean;
  downloaded: number | boolean;
  network: string;
  type: string;
};
export const AppContext = createContext<{
  boot: Boot;
  notify: (message: string, error?: boolean, retry?: () => void) => void;
}>({} as any);
export const useApp = () => useContext(AppContext);
export function SignOutButton({
  className = "button",
}: {
  className?: string;
}) {
  const { notify } = useApp();
  const [busy, setBusy] = useState(false);
  return (
    <button
      type="button"
      className={className}
      disabled={busy}
      onClick={async () => {
        setBusy(true);
        try {
          await api("/auth/logout", "POST", {});
          resetSession("/login");
        } catch (e) {
          notify((e as Error).message, true);
          setBusy(false);
        }
      }}
    >
      {busy ? <Busy /> : <LogOut size={17} />}
      {busy ? i18n.t("profile.signingOut") : i18n.t("profile.signOut")}
    </button>
  );
}
export const imageURL = (url: string) =>
  url ? "/api/images?url=" + encodeURIComponent(url) : "";
function avatarTextColor(color: string) {
  const value = color.slice(1);
  const red = Number.parseInt(value.slice(0, 2), 16);
  const green = Number.parseInt(value.slice(2, 4), 16);
  const blue = Number.parseInt(value.slice(4, 6), 16);
  return (red * 299 + green * 587 + blue * 114) / 1000 > 150
    ? "#17141f"
    : "#ffffff";
}
export function Avatar({
  profile,
  large = false,
}: {
  profile: Profile;
  large?: boolean;
}) {
  const customColor = /^#[0-9A-Fa-f]{6}$/.test(profile.avatar);
  return (
    <span
      className={`avatar ${customColor ? "avatar-custom" : "avatar-" + profile.avatar} ${large ? "large" : ""}`}
      style={
        customColor
          ? { backgroundColor: profile.avatar, color: avatarTextColor(profile.avatar) }
          : undefined
      }
    >
      {profile.avatar.endsWith(".png") ? (
        <img src={"/api/avatars/" + profile.avatar} alt="" />
      ) : (
        Array.from(profile.display_name.trim()).slice(0, 2).join("").toUpperCase()
      )}
    </span>
  );
}
export function Poster({
  image,
  name,
  className = "",
}: {
  image?: string;
  name: string;
  className?: string;
}) {
  const [failed, setFailed] = useState(false);
  return (
    <div className={"poster " + className}>
      {image && !failed ? (
        <img
          src={imageURL(image)}
          alt={i18n.t("common.poster", { name })}
          loading="lazy"
          onError={() => setFailed(true)}
        />
      ) : (
        <>
          <Tv size={30} />
          <span>{name}</span>
        </>
      )}
    </div>
  );
}
export function Busy() {
  return <LoaderCircle size={17} className="spin" aria-label={i18n.t("common.loading")} />;
}
export function Empty({
  icon = <Tv size={30} />,
  title,
  children,
  action,
}: {
  icon?: ReactNode;
  title: string;
  children: ReactNode;
  action?: ReactNode;
}) {
  return (
    <div className="empty">
      <span className="empty-icon">{icon}</span>
      <h3>{title}</h3>
      <p>{children}</p>
      {action}
    </div>
  );
}
export function ErrorState({
  error,
  retry,
}: {
  error: Error;
  retry?: () => void;
}) {
  return (
    <div className="error-box" role="alert">
      <span>{error.message}</span>
      {retry && (
        <button className="button small" onClick={retry}>
          {i18n.t("common.retry")}
        </button>
      )}
    </div>
  );
}
export function Dialog({
  title,
  children,
  onClose,
  drawer = false,
  className = "",
}: {
  title: string;
  children: ReactNode;
  onClose: () => void;
  drawer?: boolean;
  className?: string;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useLayoutEffect(() => {
    const dialog = ref.current;
    dialog?.showModal();
    dialog?.querySelector<HTMLInputElement>("[data-autofocus]")?.focus();
    // React clears refs before passive unmount cleanup. Retain the actual
    // element so its modal state is always closed, including nested dialogs.
    return () => dialog?.close();
  }, []);
  return (
    <dialog
      ref={ref}
      className={(drawer ? "dialog drawer" : "dialog") + " " + className}
      onCancel={onClose}
      onClick={(e) => {
        if (e.target !== e.currentTarget) return;
        const box = e.currentTarget.getBoundingClientRect();
        if (
          e.clientX < box.left ||
          e.clientX > box.right ||
          e.clientY < box.top ||
          e.clientY > box.bottom
        )
          onClose();
      }}
      aria-label={title}
    >
      <div className="dialog-head">
        <h2>{title}</h2>
        <button
          className="icon-button"
          aria-label={i18n.t("common.closeDialog")}
          onClick={onClose}
        >
          <X size={20} />
        </button>
      </div>
      {children}
    </dialog>
  );
}
export function Confirm({
  title,
  message,
  onConfirm,
  onClose,
}: {
  title: string;
  message: string;
  onConfirm: () => Promise<void>;
  onClose: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const { notify } = useApp();
  return (
    <Dialog title={title} onClose={onClose}>
      <p className="muted">{message}</p>
      <div className="dialog-actions">
        <button className="button" onClick={onClose}>
          {i18n.t("common.cancel")}
        </button>
        <button
          className="button danger"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            try {
              await onConfirm();
              onClose();
            } catch (e) {
              notify((e as Error).message, true);
            } finally {
              setBusy(false);
            }
          }}
        >
          {busy && <Busy />}{i18n.t("common.confirm")}
        </button>
      </div>
    </Dialog>
  );
}
export function useLocal<T>(
  key: string,
  path: string,
  enabled = true,
) {
  return useQuery<T>({
    enabled,
    queryKey: queryKeys.local(key, path),
    queryFn: ({ signal }) => api<T>(path, "GET", undefined, signal),
  });
}
export function episodeCode(e: Episode) {
  return e.number
    ? `S${String(e.season).padStart(2, "0")}E${String(e.number).padStart(2, "0")}`
    : `S${String(e.season).padStart(2, "0")} · ${i18n.t("common.special")}`;
}
export function localDay(d: Date) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
export function episodeDay(e: Episode, timezone: string) {
  if (!e.airstamp || !Number.isFinite(new Date(e.airstamp).getTime()))
    return e.airdate;
  return dateTimeFormatter("en-CA", {
    timeZone: timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(new Date(e.airstamp));
}
export function timeLabel(e: Episode, prefs: Prefs) {
  return e.airstamp && Number.isFinite(new Date(e.airstamp).getTime())
    ? dateTimeFormatter(displayLocale(i18n.resolvedLanguage), {
        timeZone: prefs.timezone,
        hour: "numeric",
        minute: "2-digit",
        hour12: prefs.time_format === "12h",
      }).format(new Date(e.airstamp))
    : i18n.t("calendar.timeTBA");
}
export function dateLabel(value: number | string | null) {
  if (!value) return i18n.t("calendar.notYet");
  const prefs = queryClient.getQueryData<Boot>(queryKeys.bootstrap())?.preferences;
  const date = new Date(typeof value === "number" ? value * 1000 : value);
  if (!Number.isFinite(date.getTime())) return i18n.t("calendar.dateTBA");
  const day = dateTimeFormatter("en-CA", {
    timeZone: prefs?.timezone,
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
  }).format(date);
  return (
    dateOnly(day) +
    " · " +
    dateTimeFormatter(displayLocale(i18n.resolvedLanguage), {
      timeZone: prefs?.timezone,
      hour: "2-digit",
      minute: "2-digit",
      hour12: prefs?.time_format === "12h",
    }).format(date)
  );
}
export function dateOnly(day: string) {
  if (!day) return i18n.t("calendar.dateTBA");
  const date = new Date(day + "T12:00:00Z");
  if (!Number.isFinite(date.getTime())) return i18n.t("calendar.dateTBA");
  const format = queryClient.getQueryData<Boot>(queryKeys.bootstrap())?.preferences
    ?.date_format;
  const [year, month, dateNumber] = day.split("-");
  if (format === "yyyy-MM-dd") return day;
  if (format === "MM/dd/yyyy") return `${month}/${dateNumber}/${year}`;
  return dateTimeFormatter(displayLocale(i18n.resolvedLanguage), {
    timeZone: "UTC",
    day: "numeric",
    month: "short",
    year: "numeric",
  }).format(date);
}
export function bytes(value: number) {
  if (!value) return "—";
  const n = Math.floor(Math.log(value) / Math.log(1024));
  return `${(value / 1024 ** n).toFixed(n > 0 ? 1 : 0)} ${["B", "KB", "MB", "GB", "TB"][n]}`;
}
export function EpisodeDrawer({
  episode,
  onClose,
}: {
  episode: Episode;
  onClose: () => void;
}) {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const navigate = useNavigate();
  const [ep, setEp] = useState(episode);
  const [busy, setBusy] = useState(false);
  async function toggle(field: "watched" | "downloaded") {
    const old = ep;
    const value = !ep[field];
    setEp({ ...ep, [field]: value });
    setBusy(true);
    try {
      await api("/episodes/" + ep.id, "PATCH", { [field]: value });
      notify(
        field === "watched"
          ? value
            ? i18n.t("calendar.markedWatched")
            : i18n.t("calendar.markedUnwatched")
          : value
            ? i18n.t("calendar.markedDownloaded")
            : i18n.t("calendar.markedNotDownloaded"),
      );
      await invalidateResources(cache, ["calendar", "show", "shows"]);
    } catch (e) {
      setEp(old);
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <Dialog title={i18n.t("calendar.episodeDetails")} onClose={onClose} drawer>
      <div className="episode-hero">
        <Poster image={ep.show_image} name={ep.show_name} />
        <div>
          <span className="eyebrow">{episodeCode(ep)}</span>
          <h2>{ep.show_name}</h2>
          <p>{ep.name}</p>
          <span className="muted small-text">
            {dateOnly(episodeDay(ep, boot.preferences.timezone))} ·{" "}
            {timeLabel(ep, boot.preferences)}
            {ep.runtime ? ` · ${ep.runtime} ${i18n.t("common.minuteShort")}` : ""}
          </span>
        </div>
      </div>
      <p className="description">
        {ep.summary || i18n.t("calendar.noEpisodeSummary")}
      </p>
      <div className="drawer-actions">
        <button
          className={"button " + (ep.watched ? "active" : "")}
          disabled={busy}
          onClick={() => toggle("watched")}
        >
          <Check size={18} />
          {ep.watched ? i18n.t("calendar.watched") : i18n.t("calendar.markWatched")}
        </button>
        <button
          className={"button " + (ep.downloaded ? "active" : "")}
          disabled={busy}
          onClick={() => toggle("downloaded")}
        >
          <Download size={18} />
          {ep.downloaded ? i18n.t("calendar.downloaded") : i18n.t("calendar.markDownloaded")}
        </button>
        <button
          className="button primary"
          onClick={() => {
            onClose();
            navigate(
              "/search?q=" +
                encodeURIComponent(
                  ep.show_name +
                    " " +
                    episodeCode(ep).replace(` · ${i18n.t("common.special")}`, ""),
                ),
            );
          }}
        >
          <Search size={18} />
          {i18n.t("shows.searchTorrents")}
        </button>
        <button
          className="button ghost"
          onClick={() => {
            onClose();
            navigate("/shows/" + ep.show_id);
          }}
        >
          {i18n.t("shows.viewShow")}
          <ArrowUpRight size={17} />
        </button>
      </div>
    </Dialog>
  );
}
