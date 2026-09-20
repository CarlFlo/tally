export const queryKeys = {
  bootstrap: () => ["bootstrap"] as const,
  calendar: (path?: string) =>
    path ? (["calendar", path] as const) : (["calendar"] as const),
  shows: () => ["shows"] as const,
  show: (id?: string) =>
    id ? (["show", `/shows/${id}`] as const) : (["show"] as const),
  showActions: (profileId?: string) =>
    profileId
      ? (["show-actions", profileId] as const)
      : (["show-actions"] as const),
  jobs: (kind?: string, status?: string, statusNot = false) =>
    kind && status
      ? (["jobs", kind, status, statusNot ? "not" : "is"] as const)
      : (["jobs"] as const),
  schedules: () => ["schedules"] as const,
  statistics: () => ["statistics"] as const,
  logs: () => ["logs"] as const,
  settings: () => ["settings"] as const,
  editableSettings: () => ["editable-settings"] as const,
  downloader: () => ["downloader"] as const,
  backups: (profileId?: string) =>
    profileId ? (["backups", profileId] as const) : (["backups"] as const),
  inbox: (profileId?: string) =>
    profileId ? (["inbox", profileId] as const) : (["inbox"] as const),
  torrentHistory: () => ["torrent-history"] as const,
  torrentSearchView: (profileId: string) => ["torrent-search-view", profileId] as const,
  torrentAutomationShows: () => ["torrent-automation-shows"] as const,
  torrentAutomationShow: (id: string) => ["torrent-automation-show", id] as const,
  torrentAutomationRuns: () => ["torrent-automation-runs"] as const,
  torrentAutomationRun: (id: string) => ["torrent-automation-run", id] as const,
  downloads: () => ["downloads"] as const,
  capabilities: () => ["capabilities"] as const,
  sessions: () => ["sessions"] as const,
  locales: () => ["locales"] as const,
  localeCatalog: (locale: string, revision?: number) =>
    ["locales", "catalog", locale, revision ?? 0] as const,
  showSuggestions: () => ["show-suggestions"] as const,
  showSearch: (query: string) => ["show-search", query] as const,
  schedulePreview: (spec: string) => ["schedule-preview", spec] as const,
  local: (key: string, path: string) => [key, path] as const,
} as const;
