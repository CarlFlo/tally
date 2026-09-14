import { Clock3, Star } from "lucide-react";
import type { ReactNode } from "react";
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
  overview,
}: {
  episodes: Episode[];
  prefs: Prefs;
  today: string;
  select: (episode: Episode) => void;
  overview: ReactNode;
}) {
  const now = useNow();
  const days = new Map<string, Episode[]>();
  for (const episode of episodes) {
    const key = episodeDay(episode, prefs.timezone);
    days.set(key, [...(days.get(key) || []), episode]);
  }
  const sortedDays = [...days.entries()].sort(([a], [b]) => a.localeCompare(b));
  return (
    <aside className="calendar-rail">
      {overview}
      <section className="rail-section">
        <div className="section-heading">
          <h3>On the horizon</h3>
          <span className="tiny-label">NEXT UP</span>
        </div>
        {episodes.length ? (
          sortedDays.map(([day, entries], index) => (
              <section className={`horizon-day${index === sortedDays.length - 1 ? " horizon-day-last" : ""}`} key={day}>
                <h4 className="tiny-label">
                  {day === today
                    ? "TODAY"
                    : new Date(day + "T12:00:00").toLocaleDateString("en", {
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
                            aria-label="Favorite show"
                          />
                        )}
                        {episode.show_name}
                      </strong>
                      <small>
                        {episodeCode(episode)} · {timeLabel(episode, prefs)}
                        <span
                          className={`release-countdown ${countdown(episode, prefs.timezone, now) === "Available" ? "available" : ""}`}
                        >
                          {countdown(episode, prefs.timezone, now)}
                        </span>
                      </small>
                    </div>
                  </button>
                ))}
              </section>
            ))
        ) : (
          <div className="rail-empty">
            <div className="orbit-art">
              <span />
              <span />
              <Clock3 size={25} />
            </div>
            <h4>Something to look forward to.</h4>
            <p>Upcoming episodes from your shows will land right here.</p>
          </div>
        )}
      </section>
    </aside>
  );
}
