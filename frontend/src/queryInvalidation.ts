import type { QueryClient, QueryKey } from "@tanstack/react-query";
import { queryKeys } from "./queryKeys";

export type Change = {
  resource: string;
  id?: string;
};

const prefixes: Record<string, () => QueryKey> = {
  bootstrap: queryKeys.bootstrap,
  calendar: () => queryKeys.calendar(),
  shows: queryKeys.shows,
  show: () => queryKeys.show(),
  "show-actions": () => queryKeys.showActions(),
  jobs: queryKeys.jobs,
  schedules: queryKeys.schedules,
  statistics: queryKeys.statistics,
  logs: queryKeys.logs,
  settings: queryKeys.settings,
  "editable-settings": queryKeys.editableSettings,
  downloader: queryKeys.downloader,
  backups: queryKeys.backups,
  inbox: () => queryKeys.inbox(),
  "torrent-history": queryKeys.torrentHistory,
  capabilities: queryKeys.capabilities,
  sessions: queryKeys.sessions,
};

type Pending = {
  timer?: number;
  changes: Map<string, Change>;
  waiters: Array<{ resolve: () => void; reject: (error: unknown) => void }>;
};

const pending = new WeakMap<QueryClient, Pending>();

function key(change: Change) {
  return `${change.resource}:${change.id || ""}`;
}

async function flush(client: QueryClient, state: Pending) {
  state.timer = undefined;
  const changes = [...state.changes.values()];
  const waiters = state.waiters.splice(0);
  state.changes.clear();
  try {
    await Promise.all(
      changes.map((change) => {
        const prefix = prefixes[change.resource];
        if (!prefix) return Promise.resolve();
        const queryKey =
          change.resource === "show" && change.id
            ? queryKeys.show(change.id)
            : prefix();
        return client.invalidateQueries(
          { queryKey, refetchType: "active" },
          { cancelRefetch: false },
        );
      }),
    );
    waiters.forEach(({ resolve }) => resolve());
  } catch (error) {
    waiters.forEach(({ reject }) => reject(error));
  }
}

export function invalidateChanges(
  client: QueryClient,
  changes: Change[],
  delay = 75,
): Promise<void> {
  if (!changes.length) return Promise.resolve();
  let state = pending.get(client);
  if (!state) {
    state = { changes: new Map(), waiters: [] };
    pending.set(client, state);
  }
  for (const change of changes) state.changes.set(key(change), change);
  if (state.timer === undefined)
    state.timer = window.setTimeout(() => void flush(client, state!), delay);
  return new Promise<void>((resolve, reject) =>
    state!.waiters.push({ resolve, reject }),
  );
}

export function invalidateResources(
  client: QueryClient,
  resources: string[],
  delay = 75,
) {
  return invalidateChanges(
    client,
    resources.map((resource) => ({ resource })),
    delay,
  );
}

export async function revalidateActiveServerData(client: QueryClient) {
  await client.invalidateQueries(
    {
      type: "active",
      refetchType: "active",
      predicate: (query) =>
        !["schedule-preview", "show-search", "show-suggestions"].includes(
          String(query.queryKey[0]),
        ),
    },
    { cancelRefetch: false },
  );
}
