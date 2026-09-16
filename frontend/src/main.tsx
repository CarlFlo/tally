import { QueryClientProvider, useQuery } from "@tanstack/react-query";
import { AlertCircle, CalendarDays, Menu, Search, Tv, X } from "lucide-react";
import { lazy, Suspense, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { createRoot } from "react-dom/client";
import {
  BrowserRouter,
  Navigate,
  NavLink,
  Route,
  Routes,
  useLocation,
} from "react-router-dom";
import { AdminRoute } from "./AdminRoute";
import { HeaderProfileMenu } from "./HeaderProfileMenu";
import { InboxDropdown } from "./InboxDropdown";
import {
  api,
  AppContext,
  Busy,
  ErrorState,
  queryClient,
  type Boot,
} from "./lib";
import { LibraryActionsProvider } from "./LibraryActions";
import { LoginAppearance } from "./LoginAppearance";
import { Logo } from "./Logo";
import { LiveUpdates } from "./LiveUpdates";
import { Notice, type Toast } from "./Notice";
import { CalendarPage } from "./pages/Calendar";
import { SearchPage } from "./pages/Search";
import { AddShow, ShowPage, ShowsPage } from "./pages/Shows";
import { PasswordGate } from "./PasswordGate";
import { ProfilePicker } from "./ProfilePicker";
import { RegisterProfile } from "./RegisterProfile";
import { queryKeys } from "./queryKeys";
import { invalidateResources } from "./queryInvalidation";
import { LocalizationProvider, useLocalization } from "./i18n";

import "./style.css";
import "./activity.css";
import "./workspace.css";

const LogsPage = lazy(() =>
  import("./pages/Logs").then((module) => ({ default: module.LogsPage })),
);
const SettingsPage = lazy(() =>
  import("./pages/Operations").then((module) => ({
    default: module.SettingsPage,
  })),
);
const SystemPage = lazy(() =>
  import("./pages/System").then((module) => ({ default: module.SystemPage })),
);

function LocaleFallbackNotice() {
  const { t } = useTranslation();
  const { requestedLocale, activeLocale, localeStatus } = useLocalization();
  if (requestedLocale === activeLocale) return null;
  const status = localeStatus(requestedLocale);
  return (
    <div className="public-warning" role="status">
      <AlertCircle size={19} />
      <span>
        {t("language.fallbackNotice")}
        {status?.error ? ` ${status.error}` : ""}
      </span>
    </div>
  );
}

function RouteFallback() {
  return (
    <div className="page">
      <Busy />
    </div>
  );
}

function App() {
  const { t } = useTranslation();
  const bootstrap = useQuery<Boot>({
    queryKey: queryKeys.bootstrap(),
    queryFn: ({ signal }) => api("/bootstrap", "GET", undefined, signal),
  });
  const [add, setAdd] = useState(false);
  const [mobile, setMobile] = useState(false);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const toast = toasts[0] || null;
  function setToast(value: Toast | null) {
    setToasts((old) =>
      value
        ? value.error
          ? [...old.filter((t) => t.error), value]
          : old.some((t) => t.error)
            ? old
            : [value]
        : old.slice(1),
    );
  }
  const location = useLocation();
  const pageKey = (["calendar", "shows", "search", "system", "settings"] as const).find(
    (key) => location.pathname.startsWith("/" + key),
  );
  const pageLabel = pageKey ? t(`nav.${pageKey}`) : t("brand.tagline");
  const boot = bootstrap.data;
  const notify = (message: string, error = false, retry?: () => void) =>
    setToast({ message, error, retry });
  useEffect(() => {
    setMobile(false);
    setAdd(false);
  }, [location.key]);
  useEffect(() => {
    if (!mobile) return;
    const previousOverflow = document.body.style.overflow;
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key === "Escape") setMobile(false);
    };
    document.body.style.overflow = "hidden";
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      document.body.style.overflow = previousOverflow;
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, [mobile]);
  useEffect(() => {
    const revalidate = (event: PageTransitionEvent) => {
      if (event.persisted) window.location.reload();
    };
    const sessionChanged = (event: StorageEvent) => {
      if (event.key === "tally-session-change") {
        queryClient.clear();
        window.location.reload();
      }
    };
    window.addEventListener("pageshow", revalidate);
    window.addEventListener("storage", sessionChanged);
    return () => {
      window.removeEventListener("pageshow", revalidate);
      window.removeEventListener("storage", sessionChanged);
    };
  }, []);
  useEffect(() => {
    const theme =
      (boot?.profile ? boot.preferences?.theme : boot?.browser_theme) ||
      "system";
    document.documentElement.dataset.theme = theme;
  }, [boot?.preferences?.theme, boot?.browser_theme, boot?.profile?.id]);
  useEffect(() => {
    if (boot?.profile && !boot.restricted && !boot.preferences_initialized) {
      void api("/preferences", "PATCH", {
        timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
      })
        .then(() => invalidateResources(queryClient, ["bootstrap"]))
        .catch(() => {});
    }
  }, [boot?.profile?.id, boot?.preferences_initialized, boot?.restricted]);
  useEffect(() => {
    document.title = "Tally · " + pageLabel;
  }, [pageLabel]);
  if (bootstrap.isPending)
    return (
      <div className="startup">
        <Logo />
        <Busy />
      </div>
    );
  if (bootstrap.error)
    return (
      <div className="startup">
        <Logo />
        <ErrorState error={bootstrap.error} retry={() => bootstrap.refetch()} />
      </div>
    );
  if (!boot) return null;
  return (
    <LocalizationProvider profileLocale={boot.profile?.locale}>
    <AppContext.Provider value={{ boot, notify }}>
      <LibraryActionsProvider>
        <LiveUpdates enabled={!!boot.profile && !boot.restricted} />
        {!boot.profile ? (
          <>
            {" "}
            <LoginAppearance boot={boot} />
            <Routes>
              <Route path="/login/new" element={<RegisterProfile />} />
              <Route
                path="/login"
                element={<ProfilePicker boot={boot} notify={notify} />}
              />
              <Route
                path="/login/:profileId"
                element={<ProfilePicker boot={boot} notify={notify} />}
              />
              <Route path="*" element={<Navigate to="/login" replace />} />
            </Routes>
          </>
        ) : boot.restricted ? (
          <PasswordGate notify={notify} />
        ) : (
          <div className={"app-shell" + (mobile ? " mobile-nav-open" : "")}>
            {mobile && (
              <div className="mobile-scrim" onClick={() => setMobile(false)} />
            )}
            <button
              className="icon-button mobile-menu"
              aria-label={mobile ? t("accessibility.closeNavigation") : t("accessibility.openNavigation")}
              aria-expanded={mobile}
              aria-controls="main-navigation"
              onClick={() => setMobile((open) => !open)}
            >
              {mobile ? <X size={21} /> : <Menu size={21} />}
            </button>
            <aside
              id="main-navigation"
              className={"sidebar " + (mobile ? "open" : "")}
            >
              <NavLink to="/calendar" className="brand-link">
                <Logo />
              </NavLink>
              <div className="workspace-label">{t("brand.workspace")}</div>
              <nav aria-label={t("accessibility.mainNavigation")}>
                <div className="nav-group">
                  <NavLink to="/calendar">
                    <CalendarDays size={19} />
                    {t("nav.calendar")}
                  </NavLink>
                  <NavLink to="/shows">
                    <Tv size={19} />
                    {t("nav.shows")}
                  </NavLink>
                  <NavLink to="/search">
                    <Search size={19} />
                    {t("nav.search")}
                  </NavLink>
                </div>
              </nav>
              <div className="sidebar-bottom">
                <div className="local-status">
                  <span className="status-dot" />
                  Tally v{boot.version}
                </div>
              </div>
            </aside>
            <div className="main-shell">
              <header className="topbar">
                <div className="topbar-left">
                  <span className="topbar-breadcrumb">
                    {t("nav.yourSpace")} <span>/</span>{" "}
                    <strong>
                      {pageKey ? pageLabel : t("nav.calendar")}
                    </strong>
                  </span>
                </div>
                <div className="topbar-right">
                  <InboxDropdown />
                  <HeaderProfileMenu />
                </div>
              </header>
              {boot.warning && (
                <div className="public-warning" role="alert">
                  <AlertCircle size={19} />
                  {boot.warning}
                </div>
              )}
              <LocaleFallbackNotice />
              <main id="main">
                <Suspense fallback={<RouteFallback />}>
                  <Routes>
                    <Route
                      path="/calendar"
                      element={<CalendarPage onAdd={() => setAdd(true)} />}
                    />
                    <Route
                      path="/shows"
                      element={<ShowsPage onAdd={() => setAdd(true)} />}
                    />
                    <Route path="/shows/:id" element={<ShowPage />} />
                    <Route path="/search" element={<SearchPage />} />
                    <Route
                      path="/logs"
                      element={
                        !!boot.profile.is_admin ? (
                          <Navigate to="/system/logs" replace />
                        ) : (
                          <LogsPage personal />
                        )
                      }
                    />
                    <Route element={<AdminRoute />}>
                      <Route path="/system/*" element={<SystemPage />} />
                      <Route
                        path="/jobs"
                        element={<Navigate to="/system/jobs" replace />}
                      />
                      <Route
                        path="/statistics"
                        element={<Navigate to="/system/statistics" replace />}
                      />
                      <Route
                        path="/settings/backups"
                        element={<Navigate to="/settings" replace />}
                      />
                      <Route path="/settings" element={<SettingsPage />} />
                      {(
                        [
                          "torrent",
                          "search",
                          "notifications",
                          "bell",
                          "debug",
                        ] as const
                      ).map((tab) => (
                        <Route
                          key={tab}
                          path={"/settings/" + tab}
                          element={<SettingsPage tab={tab} />}
                        />
                      ))}
                      <Route
                        path="/settings/scheduling"
                        element={<Navigate to="/settings" replace />}
                      />
                      <Route
                        path="/settings/profiles"
                        element={<SettingsPage tab="profiles" />}
                      />
                    </Route>
                    <Route
                      path="/profile/danger"
                      element={<SettingsPage tab="danger" />}
                    />
                    <Route
                      path="/profile"
                      element={<SettingsPage tab="personal" />}
                    />
                    <Route
                      path="/profile/security"
                      element={<SettingsPage tab="security" />}
                    />
                    <Route
                      path="/login/*"
                      element={<Navigate to="/calendar" replace />}
                    />
                    <Route
                      path="/"
                      element={<Navigate to="/calendar" replace />}
                    />
                    <Route
                      path="*"
                      element={
                        <div className="page">
                          <h1>{t("errors.pageNotFound")}</h1>
                          <NavLink className="button" to="/calendar">
                            {t("errors.backToCalendar")}
                          </NavLink>
                        </div>
                      }
                    />
                  </Routes>
                </Suspense>
              </main>
              <footer className="footer">
                <span>
                  Tally <span className="footer-dot">·</span> {t("brand.tagline")}
                </span>
                <a
                  href="https://www.tvmaze.com"
                  target="_blank"
                  rel="noreferrer"
                >
                  {t("footer.metadata")}
                </a>
              </footer>
            </div>
          </div>
        )}
        {add && <AddShow onClose={() => setAdd(false)} />}
        {toast && <Notice toast={toast} dismiss={() => setToast(null)} />}
      </LibraryActionsProvider>
    </AppContext.Provider>
    </LocalizationProvider>
  );
}
createRoot(document.getElementById("root")!).render(
  <QueryClientProvider client={queryClient}>
    <BrowserRouter>
      <App />
    </BrowserRouter>
  </QueryClientProvider>,
);
