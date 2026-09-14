import { useEffect, useRef } from "react";
import { useQueryClient } from "@tanstack/react-query";

export function LiveUpdates({ enabled }: { enabled: boolean }) {
  const cache = useQueryClient();
  const timer = useRef<number | undefined>(undefined);
  useEffect(() => {
    if (!enabled) return;
    let disposed = false;
    let refreshing = false;
    let changedWhileRefreshing = false;
    const schedule = () => {
      if (document.visibilityState !== "visible") {
        changedWhileRefreshing = true;
        return;
      }
      if (timer.current !== undefined || refreshing) return;
      timer.current = window.setTimeout(() => {
        timer.current = undefined;
        refreshing = true;
        changedWhileRefreshing = false;
        void cache
          .invalidateQueries(
            { type: "active", refetchType: "active" },
            { cancelRefetch: false },
          )
          .finally(() => {
            refreshing = false;
            if (!disposed && changedWhileRefreshing) schedule();
          });
      }, 150);
    };
    const refresh = () => {
      if (refreshing) changedWhileRefreshing = true;
      schedule();
    };
    const stream = new EventSource("/api/events");
    stream.onmessage = refresh;
    const visible = () => {
      if (document.visibilityState === "visible") refresh();
    };
    document.addEventListener("visibilitychange", visible);
    window.addEventListener("focus", refresh);
    return () => {
      disposed = true;
      stream.close();
      window.clearTimeout(timer.current);
      timer.current = undefined;
      document.removeEventListener("visibilitychange", visible);
      window.removeEventListener("focus", refresh);
    };
  }, [cache, enabled]);
  return null;
}
