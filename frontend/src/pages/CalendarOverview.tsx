import { useTranslation } from "react-i18next";
export function CalendarOverview({
  shows,
  episodes,
  watched,
}: {
  shows: number;
  episodes: number;
  watched: number;
}) {
  const { t } = useTranslation();
  return (
    <section className="calendar-overview" aria-label={t("calendar.overviewLabel")}>
      <div>
        <strong>{shows}</strong>
        <small>{t("calendar.showsInOrbit", { count: shows })}</small>
      </div>
      <div>
        <strong>{episodes}</strong>
        <small>{t("calendar.episodesThisView", { count: episodes })}</small>
      </div>
      <div>
        <strong>{watched}</strong>
        <small>{t("calendar.caughtUp", { count: watched })}</small>
      </div>
    </section>
  );
}
