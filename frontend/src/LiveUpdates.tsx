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

const SUSPENSION_RECOVERY_MS = 30_000;

export function LiveUpdates({ enabled }: { enabled: boolean }) {
  const cache = useQueryClient();
  const hiddenChanges = useRef(new Map<string, Change>());

  useEffect(() => {
    if (!enabled) return;
    let disposed = false;
    let needsRecovery = false;
    let hiddenAt = 0;

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
    stream.onerror = () => {
      needsRecovery = true;
    };
    stream.onmessage = (event) => {
      try {
        const payload = JSON.parse(event.data) as LiveEvent;
        if (payload.version !== 1 || !Array.isArray(payload.changes)) return;
        apply(payload.changes);
      } catch {
        // Malformed hints are ignored. A later reconnect or suspension
        // recovery revalidates authoritative active data.
      }
    };

    const recover = () => {
      if (disposed || document.visibilityState !== "visible") return;
      const missed = [...hiddenChanges.current.values()];
      hiddenChanges.current.clear();
      if (missed.length) void invalidateChanges(cache, missed);
      if (needsRecovery) void revalidateActiveServerData(cache);
      needsRecovery = false;
    };
    stream.onopen = recover;

    const visibility = () => {
      if (document.visibilityState !== "visible") {
        hiddenAt = Date.now();
        return;
      }
      if (hiddenAt && Date.now() - hiddenAt >= SUSPENSION_RECOVERY_MS)
        needsRecovery = true;
      hiddenAt = 0;
      recover();
    };

    document.addEventListener("visibilitychange", visibility);
    window.addEventListener("focus", recover);

    return () => {
      disposed = true;
      stream.close();
      hiddenChanges.current.clear();
      document.removeEventListener("visibilitychange", visibility);
      window.removeEventListener("focus", recover);
    };
  }, [cache, enabled]);

  return null;
}
