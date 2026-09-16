import { Check, Download, Star } from "lucide-react";
import { localDay, timeLabel, type Episode, type Prefs } from "../lib";
import { groupReleases, releaseLabel } from "../calendarReleases";
import { useTranslation } from "react-i18next";

type Props = {
  days: Date[];
  date: Date;
  view: string;
  todayKey: string;
  prefs: Prefs;
  byDay: Map<string, Episode[]>;
  expanded: string[];
  expand: (day: string) => void;
  select: (episodes: Episode[]) => void;
};

export function CalendarGrid({
  days,
  date,
  view,
  todayKey,
  prefs,
  byDay,
  expanded,
  expand,
  select,
}: Props) {
  const { t } = useTranslation();
  return (
    <div className="month-grid">
      {days.map((day) => {
        const key = localDay(day),
          groups = groupReleases(byDay.get(key) || []),
          today = key === todayKey;
        const visible =
          expanded.includes(key) || view === "week"
            ? groups
            : groups.slice(0, 3);
        return (
          <div
            key={key}
            className={`day-cell ${day.getMonth() !== date.getMonth() && view === "month" ? "adjacent" : ""} ${key < todayKey ? "past" : ""} ${today ? "today-cell" : ""}`}
          >
            <div className="date-line">
              <span className={today ? "today-number" : ""}>
                {day.getDate()}
              </span>
              {today && <small>{t("calendar.today").toLocaleUpperCase()}</small>}
            </div>
            <div className="day-entries">
              {visible.map((episodes) => {
                const episode = episodes[0],
                  watched = episodes.every((ep) => ep.watched),
                  downloaded = episodes.every((ep) => ep.downloaded),
                  favorite = episodes.some((ep) => ep.favorite),
                  completed = watched || downloaded;
                return (
                  <button
                    className={`calendar-episode ${completed ? "completed" : favorite ? "favorite" : ""}`}
                    key={episode.show_id}
                    title={`${episode.show_name}: ${releaseLabel(episodes)}`}
                    onClick={() => select(episodes)}
                  >
                    <strong>
                      {!!episode.favorite && (
                        <Star
                          className="calendar-favorite"
                          size={11}
                          fill="currentColor"
                          aria-label={t("calendar.favorite")}
                        />
                      )}
                      {episode.show_name}
                    </strong>
                    <span className="release-codes">
                      {releaseLabel(episodes)}
                      <span className="calendar-state-icons">
                        {downloaded && <Download size={11} aria-label={t("calendar.downloaded")} />}
                        {watched && <Check size={11} aria-label={t("calendar.watched")} />}
                      </span>
                    </span>
                    {view === "week" && (
                      <small>{timeLabel(episode, prefs)}</small>
                    )}
                  </button>
                );
              })}
              {groups.length > 3 &&
                !expanded.includes(key) &&
                view !== "week" && (
                  <button className="more-episodes" onClick={() => expand(key)}>
                    {t("calendar.more", { count: groups.length - 3 })}
                  </button>
                )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
