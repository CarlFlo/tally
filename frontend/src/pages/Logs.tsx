import { useDeferredValue, useState } from "react";
import { Search, ScrollText, ChevronLeft, ChevronRight } from "lucide-react";
import { Busy, dateLabel, Empty, ErrorState, useLocal } from "../lib";
import { useTranslation } from "react-i18next";
export const actionLabel = (action: string) =>
  action.replaceAll("_", " ").replace(/^./, (c) => c.toUpperCase());
export function LogsPage({ personal = false }: { personal?: boolean }) {
  const { t } = useTranslation();
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
            {personal ? t("logs.personalEyebrow") : t("logs.systemEyebrow")}
          </span>
          <h1>
            {t("nav.logs")}<span className="accent">.</span>
          </h1>
          {personal && (
            <p>{t("logs.personalDescription")}</p>
          )}
        </div>
      </div>
      <div className="library-toolbar">
        <div className="search-input">
          <Search size={18} />
          <input
            aria-label={t("logs.search")}
            placeholder={t("logs.searchPlaceholder")}
            maxLength={200}
            value={search}
            onChange={(e) => {
              setSearch(e.target.value);
              setOffset(0);
            }}
          />
        </div>
        <select
          aria-label={t("logs.filter")}
          value={action}
          onChange={(e) => {
            setAction(e.target.value);
            setOffset(0);
          }}
        >
          <option value="">{t("logs.allActions")}</option>
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
        <section className="panel activity-list" aria-label={t("logs.title")}>
          {!query.data?.entries.length ? (
            <Empty icon={<ScrollText size={30} />} title={t("logs.empty")}>
              {t("logs.newActions")}
            </Empty>
          ) : (
            query.data.entries.map((entry) => (
              <article className="activity-entry" key={entry.id}>
                <span className="badge">{actionLabel(entry.action)}</span>
                <div>
                  <strong>{entry.message}</strong>
                  <small>{entry.profile_name || t("logs.system")}</small>
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
{t("logs.pageRange", {
              from: offset + 1,
              to: Math.min(offset + 50, query.data.total),
              total: query.data.total,
            })}
          </span>
          <button
            className="button small"
            disabled={!offset}
            onClick={() => setOffset(Math.max(0, offset - 50))}
          >
            <ChevronLeft size={16} />
            {t("logs.previous")}
          </button>
          <button
            className="button small"
            disabled={offset + 50 >= query.data.total}
            onClick={() => setOffset(offset + 50)}
          >
            {t("logs.next")}
            <ChevronRight size={16} />
          </button>
        </div>
      )}
    </div>
  );
}
