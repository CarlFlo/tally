import type { QueryClient, QueryKey } from "@tanstack/react-query";
import { queryKeys } from "./queryKeys";

export type Resource =
  | "bootstrap"
  | "calendar"
  | "shows"
  | "show"
  | "show-actions"
  | "jobs"
  | "schedules"
  | "statistics"
  | "logs"
  | "settings"
  | "editable-settings"
  | "downloader"
  | "backups"
  | "inbox"
  | "torrent-history"
  | "downloads"
  | "capabilities"
  | "sessions"
  | "locales";

export type Change = {
  resource: Resource;
  id?: string;
};

const prefixes: Record<Resource, () => QueryKey> = {
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
  downloads: queryKeys.downloads,
  capabilities: queryKeys.capabilities,
  sessions: queryKeys.sessions,
  locales: queryKeys.locales,
};

type Waiter = {
  resolve: () => void;
  reject: (error: unknown) => void;
};

type Pending = {
  timer?: number;
  flushing: boolean;
  delay: number;
  changes: Map<string, Change>;
  waiters: Waiter[];
};

const pending = new WeakMap<QueryClient, Pending>();

function key(change: Change) {
  return `${change.resource}:${change.id || ""}`;
}

function schedule(client: QueryClient, state: Pending) {
  if (state.flushing || state.timer !== undefined || !state.changes.size) return;
  const delay = state.delay;
  state.delay = 75;
  state.timer = window.setTimeout(() => void flush(client, state), delay);
}

async function flush(client: QueryClient, state: Pending) {
  if (state.flushing) return;
  state.timer = undefined;
  if (!state.changes.size) return;

  state.flushing = true;
  const changes = [...state.changes.values()];
  const waiters = state.waiters.splice(0);
  state.changes.clear();

  try {
    await Promise.all(
      changes.map((change) => {
        const prefix = prefixes[change.resource];
        const queryKey =
          change.resource === "show" && change.id
            ? queryKeys.show(change.id)
            : prefix();
        return client.invalidateQueries(
          { queryKey, refetchType: "active" },
          { cancelRefetch: true },
        );
      }),
    );
    waiters.forEach(({ resolve }) => resolve());
  } catch (error) {
    waiters.forEach(({ reject }) => reject(error));
  } finally {
    state.flushing = false;
    schedule(client, state);
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
    state = {
      flushing: false,
      delay,
      changes: new Map(),
      waiters: [],
    };
    pending.set(client, state);
  } else {
    state.delay = Math.min(state.delay, delay);
  }

  for (const change of changes) state.changes.set(key(change), change);

  const result = new Promise<void>((resolve, reject) =>
    state!.waiters.push({ resolve, reject }),
  );
  schedule(client, state);
  return result;
}

export function invalidateResources(
  client: QueryClient,
  resources: Resource[],
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
