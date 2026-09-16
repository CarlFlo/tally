import { useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Bell, CheckCircle2, Clock3, X, XCircle } from "lucide-react";
import { api, Busy, dateLabel, ErrorState, useApp } from "./lib";
import { usePopover } from "./usePopover";
import { queryKeys } from "./queryKeys";
import { invalidateResources } from "./queryInvalidation";
import { useTranslation } from "react-i18next";

type Entry = {
  id: number;
  action: string;
  message: string;
  created_at: number;
  status: string;
};
type Inbox = { entries: Entry[]; unread: number; latest_id: number };

export function InboxDropdown() {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const { open, setOpen, root, trigger } = usePopover();
  const key = queryKeys.inbox(boot.profile!.id);
  const query = useQuery<Inbox>({
    queryKey: key,
    queryFn: ({ signal }) => api("/inbox", "GET", undefined, signal),
  });
  const seen = useRef(0);
  const latest = query.data?.latest_id || 0;
  useEffect(() => {
    if (!open || latest <= seen.current) return;
    seen.current = latest;
    void api("/inbox/seen", "POST", { through: latest })
      .then(() => invalidateResources(cache, ["inbox"]))
      .catch(() => {
        seen.current = 0;
      });
  }, [open, latest, cache]);
  async function update(path: string, method: string, body?: unknown) {
    try {
      await api(path, method, body);
      await invalidateResources(cache, ["inbox"]);
    } catch (e) {
      notify((e as Error).message, true);
    }
  }
  return (
    <div className="header-popover" ref={root}>
      <button
        ref={trigger}
        className={
          "icon-button inbox-bell " + (query.data?.unread ? "has-unread" : "")
        }
        aria-label={`${t("inbox.title")}${query.data?.unread ? `, ${t("inbox.unread", { count: query.data.unread })}` : ""}`}
        aria-expanded={open}
        aria-controls="notification-inbox"
        onClick={() => setOpen(!open)}
      >
        <Bell size={21} />
        {!!query.data?.unread && (
          <span className="unread-dot" aria-hidden="true" />
        )}
      </button>
      {open && (
        <section
          className="header-dropdown inbox-dropdown"
          id="notification-inbox"
          aria-label={t("inbox.title")}
        >
          <div className="inbox-heading">
            <h3>{t("inbox.label")}</h3>
            <button
              className="text-button"
              disabled={!query.data?.entries.length}
              onClick={() =>
                void update("/inbox/clear", "POST", { through: latest })
              }
            >
              {t("inbox.clearAll")}
            </button>
          </div>
          {query.isPending && <Busy />}
          {query.error && (
            <ErrorState error={query.error} retry={() => query.refetch()} />
          )}
          <div className="inbox-entries">
            {query.data?.entries.map((entry) => (
              <article className="inbox-entry" key={entry.id}>
                {entry.status === "failed" ? (
                  <XCircle className="inbox-failed" size={20} />
                ) : entry.status === "started" ? (
                  <Clock3 className="muted" size={20} />
                ) : (
                  <CheckCircle2 className="inbox-success" size={20} />
                )}
                <div>
                  <p>{entry.message}</p>
                  <small>{dateLabel(entry.created_at)}</small>
                  <span className={"inbox-status " + entry.status}>
                    {entry.status === "failed"
                      ? t("inbox.failed")
                      : entry.status === "started"
                        ? t("inbox.started")
                        : t("inbox.success")}
                  </span>
                </div>
                <button
                  className="icon-button"
                  aria-label={t("inbox.dismissEntry", { message: entry.message })}
                  onClick={() => void update(`/inbox/${entry.id}`, "DELETE")}
                >
                  <X size={15} />
                </button>
              </article>
            ))}
            {query.data && !query.data.entries.length && (
              <p className="inbox-empty muted">{t("inbox.empty")}</p>
            )}
          </div>
          <Link className="inbox-all" to="/logs" onClick={() => setOpen(false)}>
            {t("inbox.viewLogs")}
          </Link>
        </section>
      )}
    </div>
  );
}
