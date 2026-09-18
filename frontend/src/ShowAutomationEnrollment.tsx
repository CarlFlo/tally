import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Bot, Check } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api, useApp, type Show } from "./lib";
import { invalidateResources } from "./queryInvalidation";
import { queryKeys } from "./queryKeys";

type Enrollment = { policy: string; enabled: boolean };

export function ShowAutomationEnrollmentButton({ show }: { show: Show }) {
  const { t } = useTranslation();
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [busy, setBusy] = useState(false);
  const canManage = !!boot.profile?.is_admin && !!boot.torrent_search_enabled;

  const enrollment = useQuery<Enrollment>({
    queryKey: queryKeys.torrentAutomationShow(show.id),
    queryFn: ({ signal }) =>
      api(`/torrents/automation/shows/${show.id}`, "GET", undefined, signal),
    enabled: canManage,
  });

  if (!canManage) return null;

  const enabled = !!enrollment.data?.enabled;
  async function toggle() {
    setBusy(true);
    try {
      const next = !enabled;
      const result = await api<Enrollment>(
        `/torrents/automation/shows/${show.id}`,
        "PUT",
        { enabled: next },
      );
      cache.setQueryData(queryKeys.torrentAutomationShow(show.id), result);
      await invalidateResources(cache, ["torrent-automation-shows"]);
      notify(
        next
          ? t("torrentAutomation.showEnabled", {
              defaultValue: "Automation enabled for {{name}}",
              name: show.name,
            })
          : t("torrentAutomation.showDisabled", {
              defaultValue: "Automation disabled for {{name}}",
              name: show.name,
            }),
      );
    } catch (error) {
      notify((error as Error).message, true);
    } finally {
      setBusy(false);
    }
  }

  return (
    <button
      type="button"
      className="button show-automation-button"
      aria-pressed={enabled}
      disabled={busy || enrollment.isPending}
      onClick={() => void toggle()}
    >
      {enabled ? <Check size={17} /> : <Bot size={17} />}
      {enabled
        ? t("torrentAutomation.showAutomationOn", { defaultValue: "Automation on" })
        : t("torrentAutomation.showAutomationOff", { defaultValue: "Automation off" })}
    </button>
  );
}
