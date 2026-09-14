export const queryKeys = {
  bootstrap: () => ["bootstrap"] as const,
  calendar: (path?: string) =>
    path ? (["calendar", path] as const) : (["calendar"] as const),
  shows: () => ["shows"] as const,
  show: (id?: string) => (id ? (["show", id] as const) : (["show"] as const)),
  showActions: (profileId?: string) =>
    profileId
      ? (["show-actions", profileId] as const)
      : (["show-actions"] as const),
  jobs: () => ["jobs"] as const,
  schedules: () => ["schedules"] as const,
  statistics: () => ["statistics"] as const,
  logs: () => ["logs"] as const,
  settings: () => ["settings"] as const,
  editableSettings: () => ["editable-settings"] as const,
  downloader: () => ["downloader"] as const,
  backups: () => ["backups"] as const,
  inbox: (profileId?: string) =>
    profileId ? (["inbox", profileId] as const) : (["inbox"] as const),
  torrentHistory: () => ["torrent-history"] as const,
  capabilities: () => ["capabilities"] as const,
  sessions: () => ["sessions"] as const,
  local: (key: string, path: string) => [key, path] as const,
} as const;
