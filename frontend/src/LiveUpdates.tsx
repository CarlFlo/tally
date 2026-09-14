import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";
import {
  invalidateChanges,
  revalidateActiveServerData,
  type Change,
} from "./queryInvalidation";

type LiveEvent = {
  version: number;
  changes: Change[];
};

export function LiveUpdates({ enabled }: { enabled: boolean }) {
  const cache = useQueryClient();
  const hiddenChanges = useRef(new Map<string, Change>());

  useEffect(() => {
    if (!enabled) return;
    let disposed = false;

    const apply = (changes: Change[]) => {
      if (!changes.length) return;
      if (document.visibilityState !== "visible") {
        for (const change of changes)
          hiddenChanges.current.set(
            `${change.resource}:${change.id || ""}`,
            change,
          );
        return;
      }
      void invalidateChanges(cache, changes);
    };

    const stream = new EventSource("/api/events");
    stream.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as LiveEvent;
        if (payload.version !== 1 || !Array.isArray(payload.changes)) return;
        apply(payload.changes);
      } catch {
        // Ignore malformed live hints. Authoritative data is recovered on focus.
      }
    };

    const recover = () => {
      if (disposed || document.visibilityState !== "visible") return;
      const missed = [...hiddenChanges.current.values()];
      hiddenChanges.current.clear();
      if (missed.length) void invalidateChanges(cache, missed);
      else void revalidateActiveServerData(cache);
    };

    document.addEventListener("visibilitychange", recover);
    window.addEventListener("focus", recover);

    return () => {
      disposed = true;
      stream.close();
      hiddenChanges.current.clear();
      document.removeEventListener("visibilitychange", recover);
      window.removeEventListener("focus", recover);
    };
  }, [cache, enabled]);

  return null;
}
