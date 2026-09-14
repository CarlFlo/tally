import {
  createContext,
  useContext,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { api, useApp } from "./lib";

type Action = {
  external_id: number;
  name: string;
  desired: number;
  followed: number;
  status: string;
  revision: number;
  error: string;
};
type Candidate = { show: { id: number; name: string }; followed: boolean };
const Context = createContext<{
  followed: (r: Candidate) => boolean;
  toggle: (r: Candidate) => void;
  pending: number;
}>({} as any);
export const useLibraryActions = () => useContext(Context);

export function LibraryActionsProvider({ children }: { children: ReactNode }) {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [optimistic, setOptimistic] = useState<Record<number, boolean>>({});
  const desired = useRef<Record<number, boolean>>({});
  const chains = useRef(new Map<number, Promise<void>>());
  const versions = useRef(new Map<number, number>());
  const seen = useRef(new Set<string>());
  const previous = useRef("");
  const actions = useQuery<Action[]>({
    queryKey: ["show-actions", boot.profile?.id],
    queryFn: ({ signal }) =>
      api("/show-actions", "GET", undefined, signal),
    enabled: !!boot.profile && !boot.restricted,
  });
  function submit(r: Candidate, follow: boolean) {
    const id = r.show.id;
    const version = (versions.current.get(id) || 0) + 1;
    versions.current.set(id, version);
    desired.current[id] = follow;
    setOptimistic((old) => ({ ...old, [id]: follow }));
    const next = (chains.current.get(id) || Promise.resolve()).then(
      async () => {
        try {
          await api("/show-actions", "POST", {
            tvmaze_id: id,
            name: r.show.name,
            follow,
          });
          await cache.invalidateQueries({ queryKey: ["show-actions"] });
        } catch (e) {
          notify((e as Error).message, true, () => submit(r, follow));
        } finally {
          if (versions.current.get(id) === version) {
            delete desired.current[id];
            setOptimistic((old) => {
              const copy = { ...old };
              delete copy[id];
              return copy;
            });
            chains.current.delete(id);
          }
        }
      },
    );
    chains.current.set(id, next);
  }
  useEffect(() => {
    const signature = JSON.stringify(actions.data);
    if (signature === previous.current) return;
    previous.current = signature;
    for (const action of actions.data || []) {
      const token = `${action.external_id}:${action.revision}`;
      if (action.status === "failed" && !seen.current.has(token)) {
        seen.current.add(token);
        notify(action.error || `Could not update ${action.name}.`, true, () =>
          submit(
            {
              show: { id: action.external_id, name: action.name },
              followed: !!action.followed,
            },
            !!action.desired,
          ),
        );
      }
    }
    for (const key of [
      "shows",
      "show",
      "calendar",
      "show-search",
      "show-suggestions",
    ])
      void cache.invalidateQueries({ queryKey: [key] });
  }, [actions.data]);
  function followed(r: Candidate) {
    if (r.show.id in optimistic) return optimistic[r.show.id];
    const action = actions.data?.find((a) => a.external_id === r.show.id);
    return action
      ? ["queued", "running"].includes(action.status)
        ? !!action.desired
        : !!action.followed
      : !!r.followed;
  }
  return (
    <Context.Provider
      value={{
        followed,
        toggle: (r) => submit(r, !(desired.current[r.show.id] ?? followed(r))),
        pending: (actions.data || []).filter((a) =>
          ["queued", "running"].includes(a.status),
        ).length,
      }}
    >
      {children}
    </Context.Provider>
  );
}
