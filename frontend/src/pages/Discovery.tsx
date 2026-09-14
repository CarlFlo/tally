import { useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Check, Plus, Search, Star, X } from "lucide-react";
import { api, Busy, Dialog, Empty, ErrorState, Poster, useApp } from "../lib";
import { useLibraryActions } from "../LibraryActions";
import { queryKeys } from "../queryKeys";

export function AddShow({ onClose }: { onClose: () => void }) {
  const [query, setQuery] = useState("");
  const [debounced, setDebounced] = useState("");
  const [selected, setSelected] = useState<any>(null);
  const { boot, notify } = useApp();
  const library = useLibraryActions();
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 350);
    return () => clearTimeout(timer);
  }, [query]);
  const suggestions = useQuery<any[]>({
    queryKey: queryKeys.showSuggestions(),
    queryFn: ({ signal }) =>
      api("/shows/suggestions", "GET", undefined, signal),
    enabled: query.trim() === "",
    staleTime: 12 * 60 * 60_000,
    retry: false,
  });
  const search = useQuery<any[]>({
    queryKey: queryKeys.showSearch(debounced.toLowerCase()),
    queryFn: ({ signal }) =>
      api(
        "/shows/search?q=" + encodeURIComponent(debounced),
        "GET",
        undefined,
        signal,
      ),
    enabled: debounced.length >= 2 && !!query.trim(),
    staleTime: 10 * 60_000,
    retry: false,
  });
  const browsing = !query.trim();
  const results = browsing ? suggestions : search;
  const button = (result: any) => (
    <button
      className={
        "button small " + (library.followed(result) ? "added" : "primary")
      }
      aria-pressed={library.followed(result)}
      onClick={() => library.toggle(result)}
    >
      {library.followed(result) ? <Check size={17} /> : <Plus size={17} />}
      <span>{library.followed(result) ? "Added" : "Add"}</span>
    </button>
  );
  return (
    <>
      <Dialog
        title="Find your next favorite"
        onClose={onClose}
        className="discovery-dialog"
      >
        <p className="muted dialog-subtitle">
          {browsing
            ? "Highly rated shows airing recently. Find something worth following."
            : "Search TVmaze. Your next story is out there."}
        </p>
        <div className="search-input large-search">
          <Search size={21} />
          <input
            data-autofocus
            aria-label="Search for a TV show"
            placeholder="Search shows"
            value={query}
            maxLength={200}
            onChange={(e) => setQuery(e.target.value)}
          />
          {results.isFetching ? (
            <Busy />
          ) : (
            query && (
              <button
                className="icon-button"
                aria-label="Clear search"
                onClick={() => setQuery("")}
              >
                <X size={17} />
              </button>
            )
          )}
        </div>
        <div className="discovery-label">
          <span className="eyebrow">
            {browsing ? "RECENTLY ON AIR · RATED 7+" : "SEARCH RESULTS"}
          </span>
          {library.pending > 0 && (
            <span className="muted small-text">
              {library.pending} queued · keep exploring
            </span>
          )}
        </div>
        {results.error && (
          <ErrorState error={results.error} retry={() => results.refetch()} />
        )}
        {!browsing && query.trim().length < 2 ? (
          <Empty title="Keep typing" icon={<Search size={24} />}>
            Use at least two characters to search.
          </Empty>
        ) : results.isPending ? (
          <div className="discovery-loading">
            <Busy /> Finding shows…
          </div>
        ) : !results.data?.length ? (
          <Empty title={browsing ? "Find your next story" : "No shows found"}>
            {browsing
              ? "Search for a title to start your watchlist."
              : "Try another title or check the spelling."}
          </Empty>
        ) : (
          <div className="discovery-grid">
            {results.data.map((result) => (
              <article className="search-show-card" key={result.show.id}>
                <div
                  className="discovery-card-main"
                  role="button"
                  tabIndex={0}
                  onClick={() => setSelected(result)}
                  aria-label={`About ${result.show.name}`}
                  onKeyDown={(event) => {
                    if (event.key === "Enter" || event.key === " ") {
                      event.preventDefault();
                      setSelected(result);
                    }
                  }}
                >
                  <div className="discovery-poster-wrap">
                    <Poster
                      image={result.show.image?.medium}
                      name={result.show.name}
                    />
                    <div
                      className="discovery-add"
                      onClick={(event) => event.stopPropagation()}
                    >
                      {button(result)}
                    </div>
                  </div>
                  <h3>
                    {result.show.name}{" "}
                    <span>{result.show.premiered?.slice(0, 4)}</span>
                  </h3>
                  <p>
                    {result.show.network?.name ||
                      result.show.webChannel?.name ||
                      result.show.status}
                  </p>
                  {result.show.rating?.average > 0 && (
                    <span className="discovery-rating">
                      <Star size={12} />
                      {result.show.rating.average}
                    </span>
                  )}
                </div>
              </article>
            ))}
          </div>
        )}
        <div className="dialog-footnote">
          TVmaze ratings · Recent US broadcast & worldwide streaming · Click
          Added to undo
        </div>
        {boot.preferences.debug_mode && (
          <button
            className="text-button debug-button"
            onClick={() =>
              notify(
                "Preview: the show could not be added. Your library has not changed.",
                true,
                () => notify("Preview retry complete."),
              )
            }
          >
            Preview add failure
          </button>
        )}
      </Dialog>
      {selected && (
        <Dialog
          title={selected.show.name}
          onClose={() => setSelected(null)}
          className="discovery-details"
        >
          <div className="discovery-detail-content">
            <Poster
              image={selected.show.image?.medium}
              name={selected.show.name}
            />
            <div>
              <div className="detail-meta">
                <span>{selected.show.status}</span>
                <span>{selected.show.premiered?.slice(0, 4)}</span>
                {selected.show.rating?.average > 0 && (
                  <span>
                    <Star size={14} />
                    {selected.show.rating.average}
                  </span>
                )}
              </div>
              <p className="description">
                {selected.show.summary || "No summary available."}
              </p>
              <div className="genre-list">
                {selected.show.genres?.map((g: string) => (
                  <span key={g}>{g}</span>
                ))}
              </div>
              {button(selected)}
            </div>
          </div>
        </Dialog>
      )}
    </>
  );
}
