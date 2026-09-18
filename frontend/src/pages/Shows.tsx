import { ShowActionsMenu } from "../ShowActionsMenu";
import { ShowAutomationEnrollmentButton } from "../ShowAutomationEnrollment";
import { released } from "../releaseTime";
import { EpisodeRow, FavoriteButton } from "../EpisodeControls";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  ArrowLeft,
  ArrowUpRight,
  Check,
  ChevronDown,
  Download,
  Plus,
  RefreshCw,
  Search,
  Star,
  Trash2,
  Tv,
  X,
} from "lucide-react";
import {
  api,
  Busy,
  Confirm,
  Dialog,
  Empty,
  EpisodeDrawer,
  ErrorState,
  Poster,
  dateLabel,
  dateOnly,
  episodeCode,
  useApp,
  useLocal,
  type Episode,
  type Show,
} from "../lib";
import { invalidateResources } from "../queryInvalidation";

export { AddShow } from "./Discovery";
export function ShowsPage({ onAdd }: { onAdd: () => void }) {
  const { t } = useTranslation();
  const shows = useLocal<Show[]>("shows", "/shows");
  const [filter, setFilter] = useState("");
  const [status, setStatus] = useState("all");
  const list = (shows.data || []).filter(
    (s) =>
      s.name.toLowerCase().includes(filter.toLowerCase()) &&
      (status === "all" || s.status === status),
  );
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">{t("library.eyebrow")}</span>
          <h1>
            {t("library.title")}<span className="accent">.</span>
          </h1>
          <p>{t("library.description")}</p>
        </div>
        <button className="button primary" onClick={onAdd}>
          <Plus size={18} />
          {t("library.addShow")}
        </button>
      </div>
      <div className="library-toolbar">
        <div className="search-input">
          <Search size={18} />
          <input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            aria-label={t("library.filter")}
            placeholder={t("library.filterPlaceholder")}
          />
        </div>
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          aria-label={t("library.status")}
        >
          <option value="all">{t("library.all")}</option>
          <option value="Running">{t("library.running")}</option>
          <option value="Ended">{t("library.ended")}</option>
          <option value="To Be Determined">{t("library.tbd")}</option>
        </select>
        <span className="muted small-text">{t("library.showCount", { count: list.length })}</span>
      </div>
      {shows.error && <ErrorState error={shows.error} />}{" "}
      {shows.isPending ? (
        <Busy />
      ) : !list.length ? (
        <div className="panel library-empty">
          <Empty
            icon={<Tv size={32} />}
            title={
              filter || status !== "all"
                ? t("library.noMatches")
                : t("library.emptyTitle")
            }
            action={
              <button className="button primary" onClick={onAdd}>
                <Plus size={17} />
                {t("library.explore")}
              </button>
            }
          >
            {filter
              ? t("library.tryFilter")
              : t("library.emptyHelp")}
          </Empty>
        </div>
      ) : (
        <div className="library-sections">
          {[
            { title: t("library.favorites"), favorite: true, items: list.filter((s) => s.favorite) },
            { title: t("library.all"), favorite: false, items: list.filter((s) => !s.favorite) },
          ]
            .filter((group) => group.items.length)
            .map((group) => (
              <section key={group.title} aria-label={group.title}>
                <h2 className="library-section-title">
                  {group.favorite && <Star size={19} fill="currentColor" />}{" "}
                  {group.title}
                </h2>
                <div className="show-grid">
                  {group.items.map((show) => (
                    <div className="show-card-shell" key={show.id}>
                      <FavoriteButton show={show} />
                      <ShowActionsMenu show={show} />
                      <Link
                        className="show-card"
                        to={"/shows/" + show.id}
                        key={show.id}
                      >
                        <div className="show-poster-wrap">
                          <Poster image={show.image} name={show.name} />
                          <span className="poster-badge">{show.status}</span>
                          <span className="poster-arrow">
                            <ArrowUpRight size={21} />
                          </span>
                        </div>
                        <h3>{show.name}</h3>
                        <p>
                          {show.premiered?.slice(0, 4)}
                          {show.network && ` · ${show.network}`}
                        </p>
                        <div className="progress-track">
                          <i
                            style={{
                              width: `${show.episode_count ? (show.watched_count / show.episode_count) * 100 : 0}%`,
                            }}
                          />
                        </div>
                        <div className="show-progress">
                          <span>
                            {t("library.watchedProgress", { watched: show.watched_count, total: show.episode_count })}
                          </span>
                          {show.episode_count > 0 &&
                          show.watched_count === show.episode_count ? (
                            <span className="completion-badge">
                              <Check size={16} />
                              {t("library.completed")}
                            </span>
                          ) : show.aired_count > 0 &&
                            show.aired_unwatched === 0 ? (
                            <span className="completion-badge">
                              <Check size={16} />
                              {t("library.caughtUp")}
                            </span>
                          ) : null}
                        </div>
                      </Link>
                    </div>
                  ))}
                </div>
              </section>
            ))}
        </div>
      )}
    </div>
  );
}
export function ShowPage() {
  const { t } = useTranslation();
  const { id } = useParams();
  const query = useLocal<{
    show: Show;
    episodes: Episode[];
    seasons: any[];
    external_ids: any[];
  }>("show", "/shows/" + id);
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const navigate = useNavigate();
  const [season, setSeason] = useState<number | null>(null);
  const [episode, setEpisode] = useState<Episode | null>(null);
  const [busy, setBusy] = useState(false);
  if (query.error)
    return (
      <div className="page">
        <ErrorState error={query.error} />
        <Link to="/shows" className="button">
          {t("library.back")}
        </Link>
      </div>
    );
  if (!query.data)
    return (
      <div className="page">
        <Busy />
      </div>
    );
  const { show, episodes, external_ids } = query.data;
  const seasons = [...new Set(episodes.map((e) => e.season))].sort(
    (a, b) => b - a,
  );
  const active = season ?? seasons[0];
  const filtered = episodes.filter((e) => e.season === active);
  const releasedInSeason = filtered.filter((e) =>
    released(e, boot.preferences.timezone),
  );
  const releasedSeasonWatched =
    releasedInSeason.length > 0 && releasedInSeason.every((e) => !!e.watched);
  const releasedSeasonDownloaded =
    releasedInSeason.length > 0 && releasedInSeason.every((e) => !!e.downloaded);
  const watched = episodes.filter((e) => !!e.watched).length;
  const aired = episodes.filter((e) => released(e, boot.preferences.timezone));
  const complete = episodes.length > 0 && watched === episodes.length;
  const caughtUp = aired.length > 0 && aired.every((e) => e.watched);
  async function bulk(body: any) {
    setBusy(true);
    try {
      await api("/shows/" + id + "/bulk", "POST", body);
      notify(t("library.statesUpdated"));
      await invalidateResources(cache, ["show", "shows", "calendar"]);
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="page">
      <Link className="back-link" to="/shows">
        <ArrowLeft size={16} />
        {t("library.title")}
      </Link>
      <div className="show-detail-hero">
        <Poster image={show.image} name={show.name} />
        <div>
          <span className="eyebrow">{show.network || t("library.libraryEyebrow")}</span>
          <h1>
            {show.name}
            <span className="accent">.</span>
          </h1>
          <div className="detail-meta">
            <span className="badge">{show.status}</span>
            <span>{show.premiered?.slice(0, 4)}</span>
            {show.rating > 0 && (
              <span>
                <Star size={15} />
                {show.rating}
              </span>
            )}
            {show.runtime > 0 && <span>{show.runtime} {t("common.minuteShort")}</span>}
          </div>
          <p className="description">
            {show.summary || t("library.noSummary")}
          </p>
          <div className="genre-list">
            {(JSON.parse(show.genres) || []).map((g: string) => (
              <span key={g}>{g}</span>
            ))}
          </div>
          <div className="show-detail-actions">
            <FavoriteButton show={show} />
            <ShowAutomationEnrollmentButton show={show} />
            <button
              className="button"
              disabled={busy}
              onClick={() => bulk({ aired_only: true, watched: true })}
            >
              <Check size={17} />
              {t("library.markAired")}
            </button>
            <ShowActionsMenu
              show={show}
              detail
              onRemoved={() => navigate("/shows")}
            />
          </div>
        </div>
      </div>
      <div className="detail-progress panel">
        <div>
          <span className="metric-icon mint">
            <Check size={20} />
          </span>
          <strong>
            {watched} <span className="muted">/ {episodes.length}</span>
          </strong>
          <span className="muted">{t("library.episodesWatched")}</span>
          {(complete || caughtUp) && (
            <span className="completion-badge">
              <Check size={17} />
              {complete ? t("library.completed") : t("library.caughtUp")}
            </span>
          )}
        </div>
        <div className="progress-track">
          <i
            style={{
              width: `${episodes.length ? (watched / episodes.length) * 100 : 0}%`,
            }}
          />
        </div>
        <span className="muted small-text">
          {t("library.lastSynced", { date: dateLabel(show.last_checked_at) })}
        </span>
      </div>
      <div className="episode-toolbar">
        <h2>{t("library.episodes")}</h2>
        <span className="episode-legend">
          <i className="watched" />
          {t("library.watched")} <i className="available" />
          {t("library.available")} <i />
          {t("library.upcoming")}
        </span>
        <label className="season-select">
          <select
            value={active ?? ""}
            onChange={(e) => setSeason(Number(e.target.value))}
            aria-label={t("library.season")}
          >
            {seasons.map((n) => (
              <option key={n} value={n}>
                {n === 0 ? t("library.specials") : t("library.season", { number: n })}
              </option>
            ))}
          </select>
          <ChevronDown size={16} />
        </label>
        <div className="bulk-actions">
          <button
            className="button small"
            disabled={busy || !releasedInSeason.length}
            onClick={() =>
              bulk({
                season: active,
                aired_only: true,
                watched: !releasedSeasonWatched,
              })
            }
          >
            <Check size={16} />
            {releasedSeasonWatched
              ? t("library.unwatchSeason")
              : t("library.watchSeason")}
          </button>
          <button
            className="button small"
            disabled={busy || !releasedInSeason.length}
            onClick={() =>
              bulk({
                season: active,
                aired_only: true,
                downloaded: !releasedSeasonDownloaded,
              })
            }
          >
            <Download size={16} />
            {releasedSeasonDownloaded
              ? t("library.clearDownloadedSeason")
              : t("library.markDownloadedSeason")}
          </button>
        </div>
      </div>
      <div className="panel episode-list">
        {filtered.length ? (
          filtered.map((ep) => (
            <EpisodeRow
              key={ep.id}
              episode={ep}
              onOpen={() => setEpisode(ep)}
            />
          ))
        ) : (
          <Empty title={t("library.shaping")}>
            {t("library.noEpisodes")}
          </Empty>
        )}
      </div>
      <div className="source-links">
        {external_ids
          .filter((e) => e.provider === "tvmaze")
          .map((e) => (
            <a
              key={e.provider}
              href={"https://www.tvmaze.com/shows/" + e.external_id}
              target="_blank"
              rel="noreferrer"
            >
              {t("library.viewTVmaze")}
              <ArrowUpRight size={14} />
            </a>
          ))}
      </div>
      {episode && (
        <EpisodeDrawer episode={episode} onClose={() => setEpisode(null)} />
      )}{" "}
    </div>
  );
}
