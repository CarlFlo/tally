import { episodeCode, type Episode } from "./lib";
import { i18n } from "./i18n";

export function groupReleases(episodes: Episode[]): Episode[][] {
  const groups = new Map<string, Episode[]>();
  for (const episode of episodes) {
    const group = groups.get(episode.show_id) || [];
    group.push(episode);
    groups.set(episode.show_id, group);
  }
  return [...groups.values()].map((group) =>
    group.sort((a, b) => a.season - b.season || a.number - b.number),
  );
}

export function releaseLabel(episodes: Episode[]): string {
  if (episodes.length === 1) return episodeCode(episodes[0]);
  const seasons = new Map<number, Episode[]>();
  for (const episode of episodes)
    seasons.set(episode.season, [
      ...(seasons.get(episode.season) || []),
      episode,
    ]);
  return [...seasons.entries()]
    .map(([season, entries]) => {
      const expected = entries[0].season_episode_count;
      if (
        season > 0 &&
        expected &&
        expected > 1 &&
        entries.length === expected &&
        entries.every(
          (ep) => ep.number > 0 && (!ep.type || ep.type === "regular"),
        )
      ) {
        return i18n.t("library.fullSeason", { season });
      }
      return entries.map(episodeCode).join(", ");
    })
    .join(" · ");
}
