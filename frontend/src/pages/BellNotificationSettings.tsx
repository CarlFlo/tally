import { BellRing } from "lucide-react";
import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { invalidateResources } from "../queryInvalidation";
import { api, useApp } from "../lib";

const categoryGroups = [
  {
    title: "Needs attention",
    description: "Problems that may need you to take action.",
    categories: [
      [
        "scheduled_job_failures",
        "Scheduled job failures",
        "Problems with metadata or maintenance jobs.",
      ],
      [
        "backup_failures",
        "Backup failures",
        "A scheduled or manual backup could not complete.",
      ],
      [
        "provider_api_failures",
        "Provider and API failures",
        "A metadata or search provider cannot be reached.",
      ],
      [
        "torrent_client_failures",
        "Torrent client failures",
        "Sending a torrent to the configured client failed.",
      ],
    ],
  },
  {
    title: "Media updates",
    description: "New episodes becoming available for followed shows.",
    categories: [
      [
        "episode_releases",
        "New episode releases",
        "An episode from a followed show is available.",
      ],
    ],
  },
  {
    title: "Successful activity",
    description: "Optional confirmations for work that completed normally.",
    categories: [
      [
        "backup_successes",
        "Successful backups",
        "A backup finished successfully.",
      ],
      [
        "routine_background",
        "Routine background tasks",
        "Successful maintenance and metadata work.",
      ],
    ],
  },
] as const;

export function BellNotificationSettings() {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [selected, setSelected] = useState(boot.preferences.bell_categories);
  const [busy, setBusy] = useState<string | null>(null);
  useEffect(() => {
    if (!busy) setSelected(boot.preferences.bell_categories);
  }, [boot.preferences.bell_categories, busy]);
  async function toggle(key: string, enabled: boolean) {
    const previous = selected;
    const next = enabled
      ? [...selected, key]
      : selected.filter((item) => item !== key);
    setSelected(next);
    setBusy(key);
    try {
      await api("/preferences", "PATCH", { bell_categories: next });
      await invalidateResources(cache, ["bootstrap", "inbox"]);
    } catch (error) {
      setSelected(previous);
      notify((error as Error).message, true);
    } finally {
      setBusy(null);
    }
  }
  return (
    <section className="panel settings-card bell-settings">
      <h3>
        <BellRing size={19} />
        Bell Notifications
      </h3>
      <p className="muted">Choose what appears in the bell.</p>
      <div className="bell-category-list">
        {categoryGroups.map((group) => (
          <section className="bell-category-group" key={group.title}>
            <div className="bell-category-heading">
              <h4>{group.title}</h4>
              <p>{group.description}</p>
            </div>
            {group.categories.map(([key, label, description]) => (
              <label key={key} className="toggle-setting">
                <input
                  type="checkbox"
                  checked={selected.includes(key)}
                  disabled={busy !== null}
                  onChange={(event) => void toggle(key, event.target.checked)}
                />
                <span>
                  <strong>{label}</strong>
                  <small>{description}</small>
                </span>
              </label>
            ))}
          </section>
        ))}
      </div>
    </section>
  );
}
