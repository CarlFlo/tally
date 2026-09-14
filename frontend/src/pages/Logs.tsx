import { useDeferredValue, useState } from "react";
import { Search, ScrollText, ChevronLeft, ChevronRight } from "lucide-react";
import { Busy, dateLabel, Empty, ErrorState, useLocal } from "../lib";
export const actionLabel = (action: string) =>
  action.replaceAll("_", " ").replace(/^./, (c) => c.toUpperCase());
export function LogsPage({ personal = false }: { personal?: boolean }) {
  const [search, setSearch] = useState(""),
    [action, setAction] = useState(""),
    [offset, setOffset] = useState(0);
  const q = useDeferredValue(search);
  const query = useLocal<{
    entries: any[];
    actions: { action: string }[];
    total: number;
  }>(
    "logs",
    `/logs?${new URLSearchParams({ q, action, offset: String(offset) })}`,
    true,
  );
  return (
    <div className="page logs-page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">
            {personal ? "YOUR ACTIVITY" : "BEHIND THE SCENES"}
          </span>
          <h1>
            Logs<span className="accent">.</span>
          </h1>
          {personal && (
            <p>Library changes and important activity for your profile.</p>
          )}
        </div>
      </div>
      <div className="library-toolbar">
        <div className="search-input">
          <Search size={18} />
          <input
            aria-label="Search logs"
            placeholder="Search shows, profiles, or activity"
            maxLength={200}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setOffset(0);
            }}
          />
        </div>
        <select
          aria-label="Filter by action"
          value={action}
          onChange={(e) => {
            setAction(e.target.value);
            setOffset(0);
          }}
        >
          <option value="">All actions</option>
          {query.data?.actions.map((a) => (
            <option key={a.action} value={a.action}>
              {actionLabel(a.action)}
            </option>
          ))}
        </select>
      </div>
      {query.error && (
        <ErrorState error={query.error} retry={() => query.refetch()} />
      )}
      {query.isPending ? (
        <Busy />
      ) : (
        <section className="panel activity-list" aria-label="Activity log">
          {!query.data?.entries.length ? (
            <Empty icon={<ScrollText size={30} />} title="No activity found">
              New actions will appear here as you use Tally.
            </Empty>
          ) : (
            query.data.entries.map((entry) => (
              <article className="activity-entry" key={entry.id}>
                <span className="badge">{actionLabel(entry.action)}</span>
                <div>
                  <strong>{entry.message}</strong>
                  <small>{entry.profile_name || "System"}</small>
                </div>
                <time
                  dateTime={new Date(entry.created_at * 1000).toISOString()}
                >
                  {dateLabel(entry.created_at)}
                </time>
              </article>
            ))
          )}
        </section>
      )}
      {!!query.data?.total && (
        <div className="log-pagination">
          <span className="muted">
            {offset + 1}–{Math.min(offset + 50, query.data.total)} of{" "}
            {query.data.total}
          </span>
          <button
            className="button small"
            disabled={!offset}
            onClick={() => setOffset(Math.max(0, offset - 50))}
          >
            <ChevronLeft size={16} />
            Previous
          </button>
          <button
            className="button small"
            disabled={offset + 50 >= query.data.total}
            onClick={() => setOffset(offset + 50)}
          >
            Next
            <ChevronRight size={16} />
          </button>
        </div>
      )}
    </div>
  );
}
