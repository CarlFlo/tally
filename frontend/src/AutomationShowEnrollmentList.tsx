import { useQuery } from "@tanstack/react-query";
import { Bot, Search } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { api, Busy, ErrorState, dateOnly } from "./lib";
import { queryKeys } from "./queryKeys";

type AutomationShow = {
  id: string;
  name: string;
  status: string;
  next_episode?: string;
  active: boolean | number;
  automation_enabled: boolean | number;
};

type AutomationShowsResponse = { shows: AutomationShow[] };
export function AutomationShowEnrollmentList({
  enrollmentDraft,
  disabled,
  onEnrollmentChange,
}: {
  enrollmentDraft: Record<string, boolean>;
  disabled: boolean;
  onEnrollmentChange: (id: string, enabled: boolean, saved: boolean) => void;
}) {
  const { t } = useTranslation();
  const [filter, setFilter] = useState("");
  const query = useQuery<AutomationShowsResponse>({
    queryKey: queryKeys.torrentAutomationShows(),
    queryFn: ({ signal }) =>
      api("/torrents/automation/shows", "GET", undefined, signal),
  });

  const shows = query.data?.shows || [];
  const needle = filter.trim().toLowerCase();
  const visible = needle
    ? shows.filter((show) => show.name.toLowerCase().includes(needle))
    : shows.filter((show) => show.status.trim().toLowerCase() === "running");
  const enrolled = shows.filter(
    (show) => enrollmentDraft[show.id] ?? !!show.automation_enabled,
  ).length;

  return (
    <section className="panel settings-card automation-shows-card">
      <div className="automation-shows-heading">
        <div>
          <h3>
            <Bot size={19} />
            {t("torrentAutomation.showsTitle")}
          </h3>
          <p className="muted">{t("torrentAutomation.showsHelp")}</p>
        </div>
        <span className="badge">
          {t("torrentAutomation.enrolledCount", { count: enrolled })}
        </span>
      </div>

      <div className="search-input automation-show-search">
        <Search size={17} />
        <input
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === "Enter") event.preventDefault();
          }}
          aria-label={t("torrentAutomation.searchShows")}
          placeholder={t("torrentAutomation.searchShowsPlaceholder")}
        />
      </div>

      {query.error ? (
        <ErrorState error={query.error} retry={() => query.refetch()} />
      ) : query.isPending ? (
        <Busy />
      ) : visible.length ? (
        <div className="automation-show-list">
          {visible.map((show) => {
            const saved = !!show.automation_enabled;
            const enabled = enrollmentDraft[show.id] ?? saved;
            return (
              <div className="automation-show-row" key={show.id}>
                <div className="automation-show-copy">
                  <Link to={"/shows/" + show.id}>{show.name}</Link>
                  <span className="muted small-text">
                    {show.next_episode
                      ? t("torrentAutomation.nextEpisode", {
                          date: dateOnly(show.next_episode),
                        })
                      : t("torrentAutomation.noUpcomingEpisode")}
                    {show.status ? ` · ${show.status}` : ""}
                  </span>
                </div>
                <label className="toggle-setting compact-toggle automation-show-toggle">
                  <input
                    type="checkbox"
                    checked={enabled}
                    disabled={disabled}
                    onChange={(event) =>
                      onEnrollmentChange(show.id, event.target.checked, saved)
                    }
                  />
                  {enabled
                    ? t("torrentAutomation.enrolled")
                    : t("torrentAutomation.notEnrolled")}
                </label>
              </div>
            );
          })}
        </div>
      ) : (
        <p className="muted automation-shows-empty">
          {needle
            ? t("torrentAutomation.noShowMatches")
            : t("torrentAutomation.noActiveShows")}
        </p>
      )}
    </section>
  );
}
