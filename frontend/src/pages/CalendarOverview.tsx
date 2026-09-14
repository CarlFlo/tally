export function CalendarOverview({
  shows,
  episodes,
  watched,
}: {
  shows: number;
  episodes: number;
  watched: number;
}) {
  return (
    <section className="calendar-overview" aria-label="Calendar overview">
      <div>
        <strong>{shows}</strong>
        <small>shows in your orbit</small>
      </div>
      <div>
        <strong>{episodes}</strong>
        <small>episodes this view</small>
      </div>
      <div>
        <strong>{watched}</strong>
        <small>already caught up</small>
      </div>
    </section>
  );
}
