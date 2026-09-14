import { useLayoutEffect, useRef } from "react";

// For imperative reads/tests. Durable mutations keep their own completion
// lifecycle; cancelling a browser request cannot undo a committed server write.
export function useLatestRequest() {
  const active = useRef<AbortController | null>(null);
  useLayoutEffect(() => () => {
    active.current?.abort();
    active.current = null;
  }, []);
  return () => {
    active.current?.abort();
    const controller = new AbortController();
    active.current = controller;
    return controller.signal;
  };
}
