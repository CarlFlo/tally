import { useQueryClient } from "@tanstack/react-query";
import { ArrowUpRight, Check, Download, Star } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import {
  api,
  dateOnly,
  episodeCode,
  useApp,
  type Episode,
  type Show,
} from "./lib";
import { released } from "./releaseTime";
import { invalidateResources } from "./queryInvalidation";

export function EpisodeRow({
  episode,
  onOpen,
}: {
  episode: Episode;
  onOpen: () => void;
}) {
  const { t } = useTranslation();
  const [ep, setEp] = useState(episode);
  const [pending, setPending] = useState(false);
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  useEffect(() => {
    if (!pending) setEp(episode);
  }, [episode, pending]);
  async function toggle(field: "watched" | "downloaded") {
    const value = !ep[field];
    setEp((old) => ({ ...old, [field]: value }));
    setPending(true);
    try {
      await api("/episodes/" + ep.id, "PATCH", { [field]: value });
      await invalidateResources(cache, ["show", "shows", "calendar"]);
    } catch (e) {
      setEp(episode);
      notify((e as Error).message, true);
    } finally {
      setPending(false);
    }
  }
  return (
    <div
      className={`episode-row ${ep.watched ? "episode-watched" : released(ep, boot.preferences.timezone) ? "episode-available" : "episode-upcoming"}`}
      onClick={onOpen}
    >
      <span className="episode-number">
        {ep.number ? String(ep.number).padStart(2, "0") : "SP"}
      </span>
      <button
        className="episode-row-title"
        onClick={(e) => {
          e.stopPropagation();
          onOpen();
        }}
      >
        <strong>{ep.name}</strong>
        <small>
          {episodeCode(ep)} · {ep.runtime ? `${ep.runtime} ${t("common.minuteShort")}` : t("calendar.runtimeTBA")}
        </small>
      </button>
      <span className="episode-airdate">
        {dateOnly(ep.airdate)}
        <small className="episode-status">
          {ep.watched
            ? t("calendar.watched")
            : released(ep, boot.preferences.timezone)
              ? t("calendar.available")
              : t("calendar.upcoming")}
        </small>
      </span>
      <button
        className={
          "icon-button state-icon " + (ep.downloaded ? "amber-text" : "")
        }
        aria-label={ep.downloaded ? t("calendar.markNotDownloaded") : t("calendar.markDownloaded")}
        aria-pressed={!!ep.downloaded}
        disabled={pending}
        onClick={(e) => {
          e.stopPropagation();
          void toggle("downloaded");
        }}
      >
        <Download size={17} />
      </button>
      <button
        className={"icon-button state-icon " + (ep.watched ? "mint-text" : "")}
        aria-label={ep.watched ? t("calendar.markUnwatched") : t("calendar.markWatched")}
        aria-pressed={!!ep.watched}
        disabled={pending}
        onClick={(e) => {
          e.stopPropagation();
          void toggle("watched");
        }}
      >
        <Check size={18} />
      </button>
      <button
        className="icon-button"
        aria-label={t("calendar.detailsFor", { name: ep.name })}
        onClick={(e) => {
          e.stopPropagation();
          onOpen();
        }}
      >
        <ArrowUpRight size={17} />
      </button>
    </div>
  );
}
export function FavoriteButton({ show }: { show: Show }) {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [busy, setBusy] = useState(false);
  const [stamping, setStamping] = useState(false);
  useEffect(() => {
    if (!stamping) return;
    const timer = setTimeout(() => setStamping(false), 350);
    return () => clearTimeout(timer);
  }, [stamping]);
  return (
    <button
      className={
        "favorite-button " +
        (show.favorite ? "is-favorite" : "") +
        (stamping ? " is-stamping" : "")
      }
      aria-label={t(show.favorite ? "library.unfavorite" : "library.favorite", { name: show.name })}
      aria-pressed={!!show.favorite}
      disabled={busy}
      onClick={async (e) => {
        e.preventDefault();
        e.stopPropagation();
        setBusy(true);
        if (!show.favorite) setStamping(true);
        try {
          await Promise.all([
            api(`/shows/${show.id}/favorite`, "PATCH", {
              favorite: !show.favorite,
            }),
            !show.favorite
              ? new Promise((resolve) => setTimeout(resolve, 320))
              : Promise.resolve(),
          ]);
          await invalidateResources(cache, ["shows", "show", "calendar"]);
        } catch (e) {
          notify((e as Error).message, true);
        } finally {
          setBusy(false);
        }
      }}
    >
      <Star
        size={16}
        fill={show.favorite || stamping ? "currentColor" : "none"}
      />
    </button>
  );
}
