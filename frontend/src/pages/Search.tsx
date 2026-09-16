import { useEffect, useRef, useState, type FormEvent } from "react";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { useSearchParams } from "react-router-dom";
import {
  ArrowDownWideNarrow,
  Check,
  Clock3,
  Copy,
  Download,
  ExternalLink,
  Search,
  Trash2,
  SlidersHorizontal,
} from "lucide-react";
import {
  api,
  Busy,
  Empty,
  ErrorState,
  bytes,
  dateLabel,
  useApp,
  useLocal,
} from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { i18n } from "../i18n";

type Result = {
  id: string;
  name: string;
  size: number;
  seeders: number;
  leechers: number;
  provider: string;
  magnet: string;
  published: string;
  download_type: string;
  sendable: boolean;
};

const QUALITY_GROUPS = [
  ["720p", "1080p", "2160p"],
  ["WEB-DL", "WEBRip", "BluRay"],
  ["x264", "x265", "HEVC"],
  ["HDR", "DV"],
] as const;

function matchesQuality(name: string, selected: string[]) {
  const normalized = name.toLowerCase();
  return QUALITY_GROUPS.every((group) => {
    const chosen = group.filter((option) => selected.includes(option));
    return (
      chosen.length === 0 ||
      chosen.some((option) => normalized.includes(option.toLowerCase()))
    );
  });
}

function torrentAge(value: string) {
  const published = new Date(value).getTime();
  if (!Number.isFinite(published)) return i18n.t("search.dateUnknown");
  const hours = Math.max(0, Math.floor((Date.now() - published) / 3_600_000));
  if (hours < 24) return i18n.t("search.hoursOld", { count: hours });
  const days = Math.floor(hours / 24);
  return days < 30 ? i18n.t("search.daysOld", { count: days }) : dateLabel(value);
}

export function SearchPage() {
  const { t } = useTranslation();
  const [params] = useSearchParams();
  const { notify } = useApp();
  const cache = useQueryClient();
  const settings = useLocal<any>("capabilities", "/capabilities");
  const history = useLocal<any>("torrent-history", "/torrents/history");
  const [query, setQuery] = useState(params.get("q") || "");
  const [results, setResults] = useState<Result[]>([]);
  const [searched, setSearched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<Error | null>(null);
  const [sending, setSending] = useState<string | null>(null);
  const [sent, setSent] = useState<string[]>([]);
  const [keys] = useState(new Map<string, string>());
  const [seeders, setSeeders] = useState(0);
  const [minSize, setMinSize] = useState("");
  const [maxSize, setMaxSize] = useState("");
  const [include, setInclude] = useState("");
  const [exclude, setExclude] = useState("");
  const [quality, setQuality] = useState<string[]>([]);
  const [sort, setSort] = useState("seeders");
  const activeSearch = useRef<AbortController | null>(null);
  useEffect(
    () => () => {
      const controller = activeSearch.current;
      activeSearch.current = null;
      controller?.abort();
    },
    [],
  );
  async function search(e: FormEvent) {
    e.preventDefault();
    activeSearch.current?.abort();
    const controller = new AbortController();
    activeSearch.current = controller;
    setBusy(true);
    setError(null);
    try {
      const data = await api<{ results: Result[]; warnings: string[] }>(
        "/torrents/search",
        "POST",
        {
          query,
          min_seeders: 0,
          min_size: 0,
          max_size: 0,
          include: "",
          exclude: "",
        },
        controller.signal,
      );
      if (controller.signal.aborted) return;
      setResults(data.results);
      if (data.warnings.length) notify(data.warnings.join(" · "), true);
      setSearched(true);
      setSent([]);
      await invalidateResources(cache, ["torrent-history"]);
    } catch (e) {
      if (!controller.signal.aborted) setError(e as Error);
    } finally {
      if (activeSearch.current === controller) {
        activeSearch.current = null;
        setBusy(false);
      }
    }
  }
  const includeWords = include.toLowerCase().split(/\s+/).filter(Boolean);
  const excludeWords = exclude.toLowerCase().split(/\s+/).filter(Boolean);
  const filtered = results
    .filter((r) => {
      const name = r.name.toLowerCase();
      return (
        r.seeders >= seeders &&
        r.size >= Number(minSize) * 1024 ** 3 &&
        (!maxSize || r.size <= Number(maxSize) * 1024 ** 3) &&
        includeWords.every((word) => name.includes(word)) &&
        matchesQuality(r.name, quality) &&
        !excludeWords.some((word) => name.includes(word))
      );
    })
    .sort((a, b) =>
      sort === "size"
        ? a.size - b.size
        : sort === "name"
          ? a.name.localeCompare(b.name)
          : b.seeders - a.seeders,
    );
  async function clearHistory(kind: "searches" | "submissions") {
    try {
      await api("/torrents/history/" + kind, "DELETE");
      await invalidateResources(cache, ["torrent-history"]);
      notify(
        t(
          kind === "searches"
            ? "search.searchesCleared"
            : "search.submissionsCleared",
        ),
      );
    } catch (e) {
      notify((e as Error).message, true);
    }
  }

  async function send(result: Result) {
    setSending(result.id);
    try {
      if (!keys.has(result.id)) {
        keys.set(
          result.id,
          Array.from(crypto.getRandomValues(new Uint8Array(16)), (byte) =>
            byte.toString(16).padStart(2, "0"),
          ).join(""),
        );
      }
      await api("/torrents/send", "POST", {
        selection: result.id,
        idempotency_key: keys.get(result.id),
      });
      setSent([...sent, result.id]);
      notify(t("search.sent"));
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      await invalidateResources(cache, ["torrent-history"]);
      setSending(null);
    }
  }
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">{t("search.eyebrow")}</span>
          <h1>
            {t("search.title")}<span className="accent">.</span>
          </h1>
          <p>{t("search.description")}</p>
        </div>
      </div>
      <form onSubmit={search} className="torrent-search-form">
        <div className="search-input large-search">
          <Search size={22} />
          <input
            aria-label={t("search.query")}
            placeholder={t("search.placeholder")}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            required
            minLength={2}
            maxLength={200}
          />
        </div>
        <button
          className="button primary"
          disabled={
            busy || !settings.data?.jackett_configured
          }
        >
          {busy ? <Busy /> : <Search size={18} />}{t("search.searchButton")}
        </button>
      </form>
      {settings.data && !settings.data.jackett_configured && (
        <div className="setup-notice">
          <span className="metric-icon purple">
            <ExternalLink size={19} />
          </span>
          <div>
            <strong>{t("search.addJackett")}</strong>
            <p>
{t("search.jackettHelp")}
            </p>
          </div>
        </div>
      )}
      <div className="search-layout">
        <aside className="filter-panel panel">
          <h3>
            <SlidersHorizontal size={17} />
            {t("search.refine")}
          </h3>
          <label>
            {t("search.minSeeders")}
            <input
              type="number"
              min="0"
              max="1000000"
              value={seeders}
              onChange={(e) => setSeeders(Math.max(0, Number(e.target.value)))}
            />
          </label>
          <div className="two-fields">
            <label>
              {t("search.minSize")}
              <input
                type="number"
                min="0"
                step="0.1"
                placeholder={t("search.any")}
                value={minSize}
                onChange={(e) => setMinSize(e.target.value)}
              />
            </label>
            <label>
              {t("search.maxSize")}
              <input
                type="number"
                min="0"
                step="0.1"
                placeholder={t("search.any")}
                value={maxSize}
                onChange={(e) => setMaxSize(e.target.value)}
              />
            </label>
          </div>
          <label>
            {t("search.include")}
            <input
              placeholder={t("search.includeExample")}
              value={include}
              onChange={(e) => setInclude(e.target.value)}
            />
          </label>
          <label>
            {t("search.exclude")}
            <input
              placeholder={t("search.excludeExample")}
              value={exclude}
              onChange={(e) => setExclude(e.target.value)}
            />
          </label>
          <label className="field-title">{t("search.quality")}</label>
          <div className="quality-chips">
            {[
              "720p",
              "1080p",
              "2160p",
              "WEB-DL",
              "WEBRip",
              "BluRay",
              "x264",
              "x265",
              "HEVC",
              "HDR",
              "DV",
            ].map((q) => (
              <button
                type="button"
                className={quality.includes(q) ? "selected" : ""}
                key={q}
                aria-pressed={quality.includes(q)}
                onClick={() =>
                  setQuality(
                    quality.includes(q)
                      ? quality.filter((v) => v !== q)
                      : [...quality, q],
                  )
                }
              >
                {q}
              </button>
            ))}
          </div>
          <p className="small-text muted">
{t("search.filtersHelp")}
          </p>
          <button
            className="text-button"
            onClick={() => {
              setSeeders(0);
              setMinSize("");
              setMaxSize("");
              setInclude("");
              setExclude("");
              setQuality([]);
            }}
          >
            {t("search.resetFilters")}
          </button>
        </aside>
        <section className="search-results">
          {error && <ErrorState error={error} />}
          <div className="results-toolbar">
            <h3>
              {searched ? t("search.resultCount", { count: filtered.length }) : t("search.results")}
            </h3>
            <label className="inline-select">
              <ArrowDownWideNarrow size={16} />
              <select
                aria-label={t("search.sort")}
                value={sort}
                onChange={(e) => setSort(e.target.value)}
              >
                <option value="seeders">{t("search.mostSeeders")}</option>
                <option value="size">{t("search.smallest")}</option>
                <option value="name">{t("search.nameAZ")}</option>
              </select>
            </label>
          </div>
          <div className="panel">
            {!searched ? (
              <Empty
                icon={<Search size={31} />}
                title={t("search.emptyStart")}
              >
{t("search.startHelp")}
              </Empty>
            ) : !filtered.length ? (
              <Empty title={t("search.noMatches")}>
                {t("search.noMatchesHelp")}
              </Empty>
            ) : (
              <div className="torrent-results-list">
                {filtered.map((result) => (
                  <div className="torrent-result" key={result.id}>
                    <span className="torrent-file-icon">
                      <Download size={19} />
                    </span>
                    <div className="torrent-result-body">
                      <h4>{result.name}</h4>
                      <div className="torrent-meta">
                        <span>{bytes(result.size)}</span>
                        <span className="mint-text">
                          ↑ {t("search.seeders", { count: result.seeders })}
                        </span>
                        <span>↓ {t("search.leechers", { count: result.leechers })}</span>
                        <span>{result.provider}</span>
                        <span>{result.download_type}</span>
                        {result.published && (
                          <span title={dateLabel(result.published)}>
                            {torrentAge(result.published)}
                          </span>
                        )}
                      </div>
                    </div>
                    <div className="torrent-actions">
                      <button
                        className="icon-button"
                        title={t("search.copyMagnet")}
                        aria-label={t("search.copyMagnetFor", { name: result.name })}
                        disabled={!result.magnet}
                        onClick={async () => {
                          try {
                            await navigator.clipboard.writeText(result.magnet);
                            notify(t("search.copied"));
                          } catch {
                            notify(
                              t("search.clipboardError"),
                              true,
                            );
                          }
                        }}
                      >
                        <Copy size={17} />
                      </button>
                      <button
                        className={
                          "button small " +
                          (sent.includes(result.id) ? "active" : "")
                        }
                        disabled={
                          !settings.data?.downloader_configured ||
                          !result.sendable ||
                          sending !== null ||
                          sent.includes(result.id)
                        }
                        onClick={() => send(result)}
                      >
                        {sending === result.id ? (
                          <Busy />
                        ) : sent.includes(result.id) ? (
                          <Check size={16} />
                        ) : (
                          <Download size={16} />
                        )}
                        <span>
                          {sent.includes(result.id) ? t("search.sentShort") : t("search.send")}
                        </span>
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
          <div className="search-footnote">
            <Check size={14} />
            {t("search.sendingNote")}
          </div>
          {history.data?.searches?.length > 0 && (
            <section className="recent-searches">
              <div className="recent-history-title">
                <h3>
                  <Clock3 size={17} />
                  {t("search.recentSearches")}
                </h3>
                <button
                  type="button"
                  className="text-button"
                  onClick={() => clearHistory("searches")}
                >
                  <Trash2 size={14} />
                  {t("search.clearSearches")}
                </button>
              </div>
              <div className="recent-chip-list">
                {history.data.searches.slice(0, 6).map((h: any, i: number) => (
                  <button
                    key={i}
                    className="recent-chip"
                    onClick={() => setQuery(h.query)}
                  >
                    {h.query}
                    <span>{h.results}</span>
                  </button>
                ))}
              </div>
            </section>
          )}
          {history.data?.sends?.length > 0 && (
            <section className="recent-searches">
              <div className="recent-history-title">
                <h3>{t("search.recent")}</h3>
                <button
                  type="button"
                  className="text-button"
                  onClick={() => clearHistory("submissions")}
                >
                  <Trash2 size={14} />
                  {t("search.clearSubmissions")}
                </button>
              </div>
              {history.data.sends.slice(0, 5).map((h: any, i: number) => (
                <div className="history-send" key={i}>
                  <span>{h.name}</span>
                  <span className={"badge " + h.status}>{h.status}</span>
                </div>
              ))}
            </section>
          )}
        </section>
      </div>
    </div>
  );
}
