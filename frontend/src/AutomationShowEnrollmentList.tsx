import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Bot, Search } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Link } from "react-router-dom";
import { api, Busy, ErrorState, dateOnly, useApp } from "./lib";
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
type Enrollment = { policy: string; enabled: boolean };

export function AutomationShowEnrollmentList() {
  const { t } = useTranslation();
  const { notify } = useApp();
  const cache = useQueryClient();
  const [filter, setFilter] = useState("");
  const [busy, setBusy] = useState<string | null>(null);
  const query = useQuery<AutomationShowsResponse>({
    queryKey: queryKeys.torrentAutomationShows(),
    queryFn: ({ signal }) =>
      api("/torrents/automation/shows", "GET", undefined, signal),
  });

  const shows = query.data?.shows || [];
  const needle = filter.trim().toLowerCase();
  const visible = needle
    ? shows.filter((show) => show.name.toLowerCase().includes(needle))
    : shows.filter((show) => !!show.active);
  const enrolled = shows.filter((show) => !!show.automation_enabled).length;

  async function setEnrollment(show: AutomationShow, enabled: boolean) {
    setBusy(show.id);
    try {
      const result = await api<Enrollment>(
        `/torrents/automation/shows/${show.id}`,
        "PUT",
        { enabled },
      );
      cache.setQueryData<AutomationShowsResponse>(
        queryKeys.torrentAutomationShows(),
        (current) =>
          current
            ? {
                shows: current.shows.map((item) =>
                  item.id === show.id
                    ? { ...item, automation_enabled: result.enabled }
                    : item,
                ),
              }
            : current,
      );
      cache.setQueryData(queryKeys.torrentAutomationShow(show.id), result);
      notify(
        enabled
          ? t("torrentAutomation.showEnabled", { name: show.name })
          : t("torrentAutomation.showDisabled", { name: show.name }),
      );
    } catch (error) {
      notify((error as Error).message, true);
    } finally {
      setBusy(null);
    }
  }

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
            const enabled = !!show.automation_enabled;
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
                    disabled={busy === show.id}
                    onChange={(event) =>
                      void setEnrollment(show, event.target.checked)
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
