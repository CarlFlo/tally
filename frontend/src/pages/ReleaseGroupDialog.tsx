import { Dialog, type Episode } from "../lib";
import { EpisodeRow } from "../EpisodeControls";
import { releaseLabel } from "../calendarReleases";

export function ReleaseGroupDialog({
  episodes,
  select,
  close,
}: {
  episodes: Episode[];
  select: (episode: Episode) => void;
  close: () => void;
}) {
  if (!episodes.length) return null;
  return (
    <Dialog
      title={episodes[0].show_name}
      onClose={close}
      className="release-group-dialog"
    >
      <p className="muted">{releaseLabel(episodes)}</p>
      <div className="season-episodes">
        {episodes.map((episode) => (
          <EpisodeRow
            key={episode.id}
            episode={episode}
            onOpen={() => select(episode)}
          />
        ))}
      </div>
    </Dialog>
  );
}
