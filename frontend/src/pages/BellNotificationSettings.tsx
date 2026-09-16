import { BellRing } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useQueryClient } from "@tanstack/react-query";
import { invalidateResources } from "../queryInvalidation";
import { api, useApp } from "../lib";

const categoryGroups = [
  {
    title: "needsAttention",
    categories: [
      "scheduled_job_failures",
      "backup_failures",
      "provider_api_failures",
      "torrent_client_failures",
    ],
  },
  { title: "mediaUpdates", categories: ["episode_releases"] },
  {
    title: "successfulActivity",
    categories: ["backup_successes", "routine_background"],
  },
] as const;

export function BellNotificationSettings() {
  const { t } = useTranslation();
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
        {t("bell.title")}
      </h3>
      <p className="muted">{t("notifications.bellHelp")}</p>
      <div className="bell-category-list">
        {categoryGroups.map((group) => (
          <section className="bell-category-group" key={group.title}>
            <div className="bell-category-heading">
              <h4>{t(`bell.${group.title}`)}</h4>
              <p>{t(`bell.${group.title}Help`)}</p>
            </div>
            {group.categories.map((key) => (
              <label key={key} className="toggle-setting">
                <input
                  type="checkbox"
                  checked={selected.includes(key)}
                  disabled={busy !== null}
                  onChange={(event) => void toggle(key, event.target.checked)}
                />
                <span>
                  <strong>{t(`bell.${key}`)}</strong>
                  <small>{t(`bell.${key}Help`)}</small>
                </span>
              </label>
            ))}
          </section>
        ))}
      </div>
    </section>
  );
}
