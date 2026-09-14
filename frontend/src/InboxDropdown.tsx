import { useEffect, useRef } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { Bell, CheckCircle2, Clock3, X, XCircle } from "lucide-react";
import { api, Busy, dateLabel, ErrorState, useApp } from "./lib";
import { usePopover } from "./usePopover";
import { queryKeys } from "./queryKeys";
import { invalidateResources } from "./queryInvalidation";

type Entry = {
  id: number;
  action: string;
  message: string;
  created_at: number;
  status: string;
};
type Inbox = { entries: Entry[]; unread: number; latest_id: number };

export function InboxDropdown() {
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
        aria-label={`Notifications${query.data?.unread ? `, ${query.data.unread} unread` : ""}`}
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
          aria-label="Notifications"
        >
          <div className="inbox-heading">
            <h3>Bell notifications</h3>
            <button
              className="text-button"
              disabled={!query.data?.entries.length}
              onClick={() =>
                void update("/inbox/clear", "POST", { through: latest })
              }
            >
              Clear all
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
                      ? "Failed"
                      : entry.status === "started"
                        ? "Started"
                        : "Success"}
                  </span>
                </div>
                <button
                  className="icon-button"
                  aria-label={`Dismiss ${entry.message}`}
                  onClick={() => void update(`/inbox/${entry.id}`, "DELETE")}
                >
                  <X size={15} />
                </button>
              </article>
            ))}
            {query.data && !query.data.entries.length && (
              <p className="inbox-empty muted">You're all caught up.</p>
            )}
          </div>
          <Link className="inbox-all" to="/logs" onClick={() => setOpen(false)}>
            View all logs
          </Link>
        </section>
      )}
    </div>
  );
}
