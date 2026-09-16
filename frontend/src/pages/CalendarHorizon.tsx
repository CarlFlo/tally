import { Clock3, Star } from "lucide-react";
import type { CSSProperties } from "react";
import { useTranslation } from "react-i18next";
import {
  episodeCode,
  episodeDay,
  Poster,
  timeLabel,
  type Episode,
  type Prefs,
} from "../lib";
import { countdown, useNow } from "../releaseTime";

export function CalendarHorizon({
  episodes,
  prefs,
  today,
  select,
  maxHeight,
}: {
  episodes: Episode[];
  prefs: Prefs;
  today: string;
  select: (episode: Episode) => void;
  maxHeight?: number;
}) {
  const { t, i18n } = useTranslation();
  const now = useNow();
  const days = new Map<string, Episode[]>();
  for (const episode of episodes) {
    const key = episodeDay(episode, prefs.timezone);
    days.set(key, [...(days.get(key) || []), episode]);
  }
  const sortedDays = [...days.entries()].sort(([a], [b]) => a.localeCompare(b));
  return (
    <aside
      className="calendar-rail"
      style={
        maxHeight
          ? ({ "--calendar-rail-height": `${maxHeight}px` } as CSSProperties)
          : undefined
      }
    >
      <section className="rail-section horizon-section">
        <div className="section-heading">
          <h3>{t("calendar.onHorizon")}</h3>
        </div>
        {episodes.length ? (
          <div className="horizon-scroll">
            {sortedDays.map(([day, entries], index) => (
              <section
                className={`horizon-day${index === sortedDays.length - 1 ? " horizon-day-last" : ""}`}
                key={day}
              >
                <h4 className="tiny-label">
                  {day === today
                    ? t("calendar.today").toLocaleUpperCase(i18n.resolvedLanguage || "en")
                    : new Date(day + "T12:00:00").toLocaleDateString(i18n.resolvedLanguage || "en", {
                        weekday: "short",
                        month: "short",
                        day: "numeric",
                      })}
                </h4>
                {entries.map((episode) => (
                  <button
                    className="upcoming-card"
                    key={episode.id}
                    onClick={() => select(episode)}
                  >
                    <Poster
                      image={episode.show_image}
                      name={episode.show_name}
                    />
                    <div>
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
                      <small>
                        {episodeCode(episode)} · {timeLabel(episode, prefs)}
                        <span
                          className={`release-countdown ${countdown(episode, prefs.timezone, now) === t("calendar.available") ? "available" : ""}`}
                        >
                          {countdown(episode, prefs.timezone, now)}
                        </span>
                      </small>
                    </div>
                  </button>
                ))}
              </section>
            ))}
          </div>
        ) : (
          <div className="rail-empty">
            <div className="orbit-art">
              <span />
              <span />
              <Clock3 size={25} />
            </div>
            <h4>{t("calendar.horizonEmptyTitle")}</h4>
            <p>{t("calendar.horizonEmptyHelp")}</p>
          </div>
        )}
      </section>
    </aside>
  );
}
