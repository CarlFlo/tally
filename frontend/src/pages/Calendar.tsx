import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  CalendarDays,
  Check,
  Download,
  ChevronLeft,
  ChevronRight,
  Plus,
  Star,
  Tv,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import {
  api,
  Empty,
  episodeCode,
  episodeDay,
  EpisodeDrawer,
  ErrorState,
  localDay,
  Poster,
  timeLabel,
  useApp,
  useLocal,
  type Episode,
  type Show,
} from "../lib";
import { useNow } from "../releaseTime";
import { dateTimeFormatter } from "../dateFormatting";
import { CalendarGrid } from "./CalendarGrid";
import { CalendarFilter } from "./CalendarFilter";
import { CalendarHorizon } from "./CalendarHorizon";
import { ReleaseGroupDialog } from "./ReleaseGroupDialog";
import { queryKeys } from "../queryKeys";
import { invalidateResources } from "../queryInvalidation";

export function CalendarPage({ onAdd }: { onAdd: () => void }) {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const prefs = boot.preferences;
  const now = useNow(60_000);
  const todayKey = dateTimeFormatter("en-CA", {
    timeZone: prefs.timezone,
  }).format(now);
  const today = new Date(todayKey + "T12:00:00");
  const [date, setDate] = useState(today);
  const [view, setView] = useState(prefs.calendar_view || "month");
  const [group, setGroup] = useState<string[]>([]);
  const [selected, setSelected] = useState<Episode | null>(null);
  const [filter, setFilter] = useState("all");
  const [expanded, setExpanded] = useState<string[]>([]);
  const calendarSection = useRef<HTMLElement | null>(null);
  const [calendarHeight, setCalendarHeight] = useState<number>();
  const shows = useLocal<Show[]>("shows", "/shows");
  const { start, end, days } = useMemo(() => {
    const start = new Date(
      date.getFullYear(),
      date.getMonth(),
      view === "month" ? 1 : date.getDate(),
      12,
    );
    if (view !== "agenda")
      start.setDate(
        start.getDate() - ((start.getDay() - prefs.week_start + 7) % 7),
      );
    const count = view === "month" ? 42 : view === "week" ? 7 : 30;
    const end = new Date(start);
    end.setDate(end.getDate() + count);
    return {
      start,
      end,
      days: Array.from({ length: count }, (_, i) => {
        const d = new Date(start);
        d.setDate(d.getDate() + i);
        return d;
      }),
    };
  }, [date, view, prefs.week_start]);
  const path = `/calendar?from=${localDay(start)}&to=${localDay(end)}`;
  const activePath = useRef(path);
  activePath.current = path;
  const episodes = useQuery<Episode[]>({
    queryKey: queryKeys.calendar(path),
    queryFn: ({ signal }) => api(path, "GET", undefined, signal),
    placeholderData: (prev) => prev,
  });
  useEffect(() => {
    const node = calendarSection.current;
    if (!node) return;
    const updateHeight = () => setCalendarHeight(node.getBoundingClientRect().height);
    updateHeight();
    const observer = new ResizeObserver(updateHeight);
    observer.observe(node);
    return () => observer.disconnect();
  }, [view, episodes.data?.length, shows.data?.length]);
  useEffect(() => {
    const from = new Date(end);
    const to = new Date(end);
    to.setDate(to.getDate() + 42);
    const prefetchPath = `/calendar?from=${localDay(from)}&to=${localDay(to)}`;
    const queryKey = queryKeys.calendar(prefetchPath);
    void cache.prefetchQuery({
      queryKey,
      queryFn: ({ signal }) => api(prefetchPath, "GET", undefined, signal),
    });
    return () => {
      if (activePath.current !== prefetchPath)
        void cache.cancelQueries({ queryKey, exact: true, type: "inactive" });
    };
  }, [localDay(end), cache]);
  const all = episodes.data || [];
  const filtered = all.filter(
    (e) =>
      filter === "all" || (filter === "unwatched" ? !e.watched : !!e.watched),
  );
  const byDay = new Map<string, Episode[]>();
  for (const ep of filtered) {
    const key = episodeDay(ep, prefs.timezone);
    const group = byDay.get(key);
    if (group) group.push(ep);
    else byDay.set(key, [ep]);
  }
  const inRange = all.filter((e) => {
    const key = episodeDay(e, prefs.timezone);
    return key >= localDay(start) && key < localDay(end);
  });
  const upcoming = inRange.filter(
    (e) => episodeDay(e, prefs.timezone) >= todayKey,
  );
  function move(direction: number) {
    const next = new Date(date);
    if (view === "month") {
      next.setDate(1);
      next.setMonth(next.getMonth() + direction);
    } else
      next.setDate(next.getDate() + (view === "week" ? 7 : 30) * direction);
    setDate(next);
  }
  async function changeView(next: string) {
    setView(next);
    try {
      await api("/preferences", "PATCH", { calendar_view: next });
      await invalidateResources(cache, ["bootstrap"]);
    } catch (e) {
      notify((e as Error).message, true);
    }
  }
  const heading =
    view === "week"
      ? `${start.toLocaleDateString("en", { month: "short", day: "numeric" })} – ${days[6].toLocaleDateString("en", { month: "short", day: "numeric" })}`
      : date.toLocaleDateString("en", { month: "long", year: "numeric" });
  return (
    <div className="page calendar-page">
      <div className="page-heading">
        <div>
          <div className="eyebrow">GOOD SHOWS. GREAT TIMING.</div>
          <h1>
            Your calendar<span className="accent">.</span>
          </h1>
        </div>
        <button className="button primary small" onClick={onAdd}>
          <Plus size={16} />
          Add show
        </button>
        <div className="heading-note calendar-date-note">
          <CalendarDays size={17} />
          {today.toLocaleDateString("en", {
            weekday: "short",
            month: "short",
            day: "numeric",
          })}
        </div>
      </div>
      <div className="calendar-layout">
        <section className="calendar-section" ref={calendarSection}>
          <div className="calendar-toolbar">
            <div className="month-controls">
              <div className="arrows">
                <button
                  className="icon-button"
                  aria-label="Previous period"
                  onClick={() => move(-1)}
                >
                  <ChevronLeft size={18} />
                </button>
                <button
                  className="icon-button"
                  aria-label="Next period"
                  onClick={() => move(1)}
                >
                  <ChevronRight size={18} />
                </button>
              </div>
              <button className="button small" onClick={() => setDate(today)}>
                Today
              </button>
              <h2>{heading}</h2>
            </div>
            <div className="calendar-toolbar-actions">
              <div className="segmented" aria-label="Calendar view">
                {["month", "week", "agenda"].map((v) => (
                  <button
                    key={v}
                    className={view === v ? "selected" : ""}
                    aria-pressed={view === v}
                    onClick={() => changeView(v)}
                  >
                    {v[0].toUpperCase() + v.slice(1)}
                  </button>
                ))}
              </div>
              <CalendarFilter value={filter} onChange={setFilter} />
            </div>
          </div>
          {episodes.error && (
            <ErrorState
              error={episodes.error}
              retry={() => episodes.refetch()}
            />
          )}
          <div className={"calendar-board view-" + view}>
            {view !== "agenda" && (
              <div className="weekdays">
                {Array.from({ length: 7 }, (_, i) => (
                  <span key={i}>
                    {
                      ["SUN", "MON", "TUE", "WED", "THU", "FRI", "SAT"][
                        (i + prefs.week_start) % 7
                      ]
                    }
                  </span>
                ))}
              </div>
            )}
            {view === "agenda" ? (
              <div className="agenda">
                {days
                  .filter((d) => byDay.has(localDay(d)))
                  .map((d) => (
                    <div className="agenda-day" key={localDay(d)}>
                      <div
                        className={
                          "agenda-date " +
                          (localDay(d) === todayKey ? "is-today" : "")
                        }
                      >
                        <strong>{d.getDate()}</strong>
                        <span>
                          {d.toLocaleDateString("en", {
                            month: "short",
                            weekday: "short",
                          })}
                        </span>
                      </div>
                      <div>
                        {byDay.get(localDay(d))?.map((ep) => (
                          <button
                            className={`agenda-episode ${ep.watched || ep.downloaded ? "completed" : ep.favorite ? "favorite" : ""}`}
                            key={ep.id}
                            onClick={() => setSelected(ep)}
                          >
                            <Poster image={ep.show_image} name={ep.show_name} />
                            <span>
                              <strong>
                                {!!ep.favorite && (
                                  <Star
                                    className="calendar-favorite"
                                    size={11}
                                    fill="currentColor"
                                    aria-label="Favorite show"
                                  />
                                )}
                                {ep.show_name}
                              </strong>
                              <small>
                                {episodeCode(ep)} · {ep.name}
                              </small>
                            </span>
                            <span className="agenda-time">
                              {timeLabel(ep, prefs)}
                            </span>
                            <span className="agenda-state-icons">
                              {!!ep.downloaded && (
                                <Download className="mint-text" size={17} aria-label="Downloaded" />
                              )}
                              {!!ep.watched && (
                                <Check className="mint-text" size={18} aria-label="Watched" />
                              )}
                            </span>
                          </button>
                        ))}
                      </div>
                    </div>
                  ))}
                {!filtered.length && (
                  <Empty
                    icon={<CalendarDays size={28} />}
                    title="A little breathing room"
                  >
                    No episodes in this date range. Your followed shows will
                    appear here when they air.
                  </Empty>
                )}
              </div>
            ) : (
              <CalendarGrid
                days={days}
                date={date}
                view={view}
                todayKey={todayKey}
                prefs={prefs}
                byDay={byDay}
                expanded={expanded}
                expand={(key) => setExpanded((old) => [...old, key])}
                select={(entries) =>
                  entries.length === 1
                    ? setSelected(entries[0])
                    : setGroup(entries.map((ep) => ep.id))
                }
              />
            )}
          </div>
          <div className="calendar-legend">
            <div className="legend">
              <span>
                <i className="legend-dot purple" />
                Upcoming
              </span>
              <span>
                <i className="legend-dot amber" />
                Favorite
              </span>
              <span>
                <i className="legend-dot mint" />
                Watched / Downloaded
              </span>
            </div>
          </div>
          {view !== "agenda" && shows.data?.length === 0 && (
            <div className="calendar-onboarding">
              <span className="mini-orbit">
                <Tv size={21} />
              </span>
              <div>
                <strong>Your next favorite belongs here.</strong>
                <p>Add a show and its episodes will find their place.</p>
              </div>
              <button className="button primary small" onClick={onAdd}>
                <Plus size={16} />
                Find a show
              </button>
            </div>
          )}
        </section>
        <CalendarHorizon
          episodes={upcoming}
          prefs={prefs}
          today={todayKey}
          select={setSelected}
          maxHeight={calendarHeight}
        />
      </div>
      {!!group.length && (
        <ReleaseGroupDialog
          episodes={all.filter((ep) => group.includes(ep.id))}
          select={setSelected}
          close={() => setGroup([])}
        />
      )}
      {selected && (
        <EpisodeDrawer episode={selected} onClose={() => setSelected(null)} />
      )}
    </div>
  );
}
