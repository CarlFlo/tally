import { useEffect, useRef, useState, type FormEvent } from "react";
import { useSearchParams } from "react-router-dom";
import {
  ArrowDownWideNarrow,
  Check,
  Clock3,
  Copy,
  Download,
  ExternalLink,
  Search,
  Send,
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

function torrentAge(value: string) {
  const published = new Date(value).getTime();
  if (!Number.isFinite(published)) return "Date unknown";
  const hours = Math.max(0, Math.floor((Date.now() - published) / 3_600_000));
  if (hours < 24) return `${hours}h old`;
  const days = Math.floor(hours / 24);
  return days < 30 ? `${days}d old` : dateLabel(value);
}

export function SearchPage() {
  const [params] = useSearchParams();
  const { notify } = useApp();
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
      await history.refetch();
    } catch (e) {
      if (!controller.signal.aborted) setError(e as Error);
    } finally {
      if (activeSearch.current === controller) {
        activeSearch.current = null;
        setBusy(false);
      }
    }
  }
  const filtered = results
    .filter(
      (r) =>
        r.seeders >= seeders &&
        r.size >= Number(minSize) * 1024 ** 3 &&
        (!maxSize || r.size <= Number(maxSize) * 1024 ** 3) &&
        [
          ...include.toLowerCase().split(/\s+/).filter(Boolean),
          ...quality.map((q) => q.toLowerCase()),
        ].every((word) => r.name.toLowerCase().includes(word)) &&
        !exclude
          .toLowerCase()
          .split(/\s+/)
          .filter(Boolean)
          .some((word) => r.name.toLowerCase().includes(word)),
    )
    .sort((a, b) =>
      sort === "size"
        ? a.size - b.size
        : sort === "name"
          ? a.name.localeCompare(b.name)
          : b.seeders - a.seeders,
    );
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
      notify("Torrent sent");
      await history.refetch();
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setSending(null);
    }
  }
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">YOU'RE IN THE DRIVER'S SEAT</span>
          <h1>
            Torrent search<span className="accent">.</span>
          </h1>
          <p>Find what you're looking for. Choose exactly what happens next.</p>
        </div>
      </div>
      <form onSubmit={search} className="torrent-search-form">
        <div className="search-input large-search">
          <Search size={22} />
          <input
            aria-label="Torrent search query"
            placeholder="A show, an episode, or anything else…"
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
          {busy ? <Busy /> : <Search size={18} />}Search torrents
        </button>
      </form>
      {settings.data && !settings.data.jackett_configured && (
        <div className="setup-notice">
          <span className="metric-icon purple">
            <ExternalLink size={19} />
          </span>
          <div>
            <strong>Connect Jackett to get started.</strong>
            <p>
              Add the Jackett base URL and API key under Settings, then test and
              save the connection.
            </p>
          </div>
        </div>
      )}
      <div className="search-layout">
        <aside className="filter-panel panel">
          <h3>
            <SlidersHorizontal size={17} />
            Refine your search
          </h3>
          <label>
            Minimum seeders
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
              Min size (GB)
              <input
                type="number"
                min="0"
                step="0.1"
                placeholder="Any"
                value={minSize}
                onChange={(e) => setMinSize(e.target.value)}
              />
            </label>
            <label>
              Max size (GB)
              <input
                type="number"
                min="0"
                step="0.1"
                placeholder="Any"
                value={maxSize}
                onChange={(e) => setMaxSize(e.target.value)}
              />
            </label>
          </div>
          <label>
            Include keywords
            <input
              placeholder="e.g. extended"
              value={include}
              onChange={(e) => setInclude(e.target.value)}
            />
          </label>
          <label>
            Exclude keywords
            <input
              placeholder="e.g. cam"
              value={exclude}
              onChange={(e) => setExclude(e.target.value)}
            />
          </label>
          <label className="field-title">QUALITY & FORMAT</label>
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
            Filters match title keywords and combine. They don't inspect media
            files.
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
            Reset filters
          </button>
        </aside>
        <section className="search-results">
          {error && <ErrorState error={error} />}
          <div className="results-toolbar">
            <h3>
              {searched ? `${filtered.length} results` : "Search results"}
            </h3>
            <label className="inline-select">
              <ArrowDownWideNarrow size={16} />
              <select
                aria-label="Sort torrent results"
                value={sort}
                onChange={(e) => setSort(e.target.value)}
              >
                <option value="seeders">Most seeders</option>
                <option value="size">Smallest size</option>
                <option value="name">Name A–Z</option>
              </select>
            </label>
          </div>
          <div className="panel">
            {!searched ? (
              <Empty
                icon={<Search size={31} />}
                title="A search starts with you."
              >
                Enter a query above. Results appear only when you search, and
                nothing is sent until you choose it.
              </Empty>
            ) : !filtered.length ? (
              <Empty title="No matching torrents">
                Try a broader query or adjust your filters.
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
                          ↑ {result.seeders} seeders
                        </span>
                        <span>↓ {result.leechers} leechers</span>
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
                        title="Copy magnet"
                        aria-label={"Copy magnet for " + result.name}
                        disabled={!result.magnet}
                        onClick={async () => {
                          try {
                            await navigator.clipboard.writeText(result.magnet);
                            notify("Magnet copied");
                          } catch {
                            notify(
                              "Clipboard access requires HTTPS or localhost.",
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
                          <Send size={16} />
                        )}
                        <span>
                          {sent.includes(result.id) ? "Sent" : "Send"}
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
            Sending a torrent never marks an episode as downloaded.
          </div>
          {history.data?.searches?.length > 0 && (
            <section className="recent-searches">
              <h3>
                <Clock3 size={17} />
                Recent searches
              </h3>
              <div>
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
              <h3>Recent submissions</h3>
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
