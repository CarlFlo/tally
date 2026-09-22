import { useEffect, useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Bell, Send, Trash2 } from "lucide-react";
import { api, Busy, ErrorState, useApp, useLocal } from "../lib";
import { NotificationFields } from "./NotificationFields";
import { useLatestRequest } from "../useLatestRequest";
import { invalidateResources } from "../queryInvalidation";
import { queryKeys } from "../queryKeys";
import { useTranslation } from "react-i18next";
import { UnsavedChangesBar, useUnsavedChangesWarning } from "../UnsavedChangesBar";
import "../unsaved-changes.css";
import {
  displayNotificationTime,
  notificationEndpointReady,
  notificationErrors,
  parseNotificationTime,
} from "./notificationValidation";
const defaults = (data: any, serverTimezone: string) => ({
  url: "",
  discord_url: "",
  bot_name: "",
  prefix: "",
  enabled: false,
  ...data,
  events: data.events ?? ["system_error", "job_failed"],
  type: data.type || "webhook",
  body:
    data.body ||
    '{"app":"Tally","message":"{{message}}","event":"{{event}}","show":"{{show}}","time":"{{time}}"}',
  delivery_time: data.delivery_time || "09:00",
  timezone: serverTimezone || data.timezone || "UTC",
});
export function NotificationSettings() {
  const query = useLocal<any>(
    "editable-settings",
    "/settings/notifications",
    true,
  );
  return (
    <>
      {query.error && (
        <ErrorState error={query.error} retry={() => query.refetch()} />
      )}
      {query.isPending && <Busy />}
      {query.data && <NotificationForm saved={query.data} />}
    </>
  );
}
function NotificationForm({ saved }: { saved: any }) {
  const { t } = useTranslation();
  const startTest = useLatestRequest();
  const { boot, notify } = useApp(),
    cache = useQueryClient();
  const [data, setData] = useState(() =>
    defaults(saved.data, saved.server_timezone),
  );
  const [secretsConfigured, setSecretsConfigured] = useState<Record<string, boolean>>(
    saved.secrets_configured || {},
  );
  const [clearedSecrets, setClearedSecrets] = useState<Record<string, boolean>>({});
  const [timeText, setTimeText] = useState(() =>
    displayNotificationTime(data.delivery_time, boot.preferences.time_format),
  );
  const [stored, setStored] = useState(data),
    [revision, setRevision] = useState(saved.revision);
  const [busy, setBusy] = useState(false),
    [feedback, setFeedback] = useState("");
  const normalized = {
    ...data,
    delivery_time: parseNotificationTime(
      timeText,
      boot.preferences.time_format,
    ),
  };
  const effectiveConfigured = {
    url: !!secretsConfigured.url && !clearedSecrets.url,
    discord_url: !!secretsConfigured.discord_url && !clearedSecrets.discord_url,
  };
  const errors = notificationErrors(normalized, effectiveConfigured);
  const savedValid = notificationEndpointReady(stored, secretsConfigured);
  const toggleMessage = savedValid
    ? ""
    : t("notifications.saveBeforeEnable");
  const hasChanges =
    JSON.stringify(normalized) !== JSON.stringify(stored) ||
    Object.values(clearedSecrets).some(Boolean);
  useUnsavedChangesWarning(
    hasChanges,
    busy,
    t("common.unsavedNavigation", { defaultValue: "You have unsaved changes. Leave this page without saving?" }),
  );
  useEffect(() => {
    setTimeText(
      displayNotificationTime(data.delivery_time, boot.preferences.time_format),
    );
  }, [boot.preferences.time_format]);
  useEffect(() => {
    if (saved.revision <= revision || JSON.stringify(data) !== JSON.stringify(stored))
      return;
    const next = defaults(saved.data, saved.server_timezone);
    setData(next);
    setStored(next);
    setSecretsConfigured(saved.secrets_configured || {});
    setClearedSecrets({});
    setRevision(saved.revision);
    setTimeText(
      displayNotificationTime(next.delivery_time, boot.preferences.time_format),
    );
  }, [boot.preferences.time_format, data, revision, saved, stored]);
  const change = (key: string, value: any) => {
    if ((key === "url" || key === "discord_url") && value) {
      setClearedSecrets((old) => ({ ...old, [key]: false }));
    }
    setData((old: any) => ({ ...old, [key]: value }));
  };
  const clearSecret = (key: "url" | "discord_url", clear: boolean) => {
    setClearedSecrets((old) => ({ ...old, [key]: clear }));
    if (clear) {
      setData((old: any) => ({ ...old, [key]: "" }));
    }
  };
  async function persist(next: any) {
    const response = await api("/settings/notifications", "PUT", {
      data: next,
      revision,
      clear_secrets: clearedSecrets,
    });
    const serverTimezone = saved.server_timezone || next.timezone || "UTC";
    const nextConfigured = {
      url: !clearedSecrets.url && (!!secretsConfigured.url || !!next.url),
      discord_url:
        !clearedSecrets.discord_url &&
        (!!secretsConfigured.discord_url || !!next.discord_url),
    };
    const persisted = {
      ...next,
      url: "",
      discord_url: "",
      timezone: serverTimezone,
    };
    setRevision(response.revision);
    setSecretsConfigured(nextConfigured);
    setClearedSecrets({});
    setStored(persisted);
    setData(persisted);
    cache.setQueryData(queryKeys.local("editable-settings", "/settings/notifications"), {
      data: persisted,
      revision: response.revision,
      server_timezone: serverTimezone,
      secrets_configured: nextConfigured,
    });
    await invalidateResources(cache, ["settings"]);
  }
  async function toggle(enabled: boolean) {
    change("enabled", enabled);
    setFeedback("");
  }
  async function save(event?: FormEvent) {
    event?.preventDefault();
    if (Object.keys(errors).length) {
      return;
    }
    setBusy(true);
    setFeedback("");
    try {
      const enabledChanged = normalized.enabled !== stored.enabled;
      await persist(normalized);
      notify(
        enabledChanged
          ? t(normalized.enabled ? "notifications.enabledNotice" : "notifications.disabledNotice")
          : t("notifications.saved"),
      );
    } catch (e) {
      setFeedback((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className="panel settings-card notification-settings">
      <div className="section-heading">
        <div>
          <h3>
            <Bell size={19} />
            {t("notifications.services")}
          </h3>
          <p className="muted">{t("notifications.description")}</p>
        </div>
        <span title={toggleMessage}>
          <label
            className={`toggle-setting notification-master${data.enabled ? " is-enabled" : ""}`}
          >
            <input
              type="checkbox"
              role="switch"
              aria-label={t("notifications.enableAll")}
              checked={data.enabled}
              disabled={busy || !savedValid}
              onChange={(e) => void toggle(e.target.checked)}
            />
            {data.enabled ? t("notifications.alertsOn") : t("notifications.alertsOff")}
          </label>
        </span>
      </div>
      <form onSubmit={save}>
        <fieldset disabled={busy}>
          <NotificationFields
            data={data}
            change={change}
            errors={errors}
            timeText={timeText}
            timeFormat={boot.preferences.time_format}
            changeTime={setTimeText}
            secretsConfigured={secretsConfigured}
            clearedSecrets={clearedSecrets}
            clearSecret={clearSecret}
          />
        </fieldset>
        {feedback && (
          <p className="form-feedback" role="alert">
            {feedback}
          </p>
        )}
        <div className="notification-buttons">
          <button
            className="button"
            type="button"
            disabled={busy}
            onClick={async () => {
              if (Object.keys(errors).length) {
                return;
              }
              setBusy(true);
              setFeedback("");
              const signal = startTest();
              try {
                const result = await api(
                  "/settings/notifications/test",
                  "POST",
                  normalized,
                  signal,
                );
                notify(result.message);
              } catch (e) {
                if (signal.aborted) return;
                setFeedback((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <Send size={17} />
            {t("notifications.testNotification")}
          </button>
          {data.type === "discord" &&
            secretsConfigured.discord_url &&
            !clearedSecrets.discord_url && (
              <button
                className="button danger"
                type="button"
                disabled={busy}
                onClick={() => clearSecret("discord_url", true)}
              >
                <Trash2 size={17} />
                {t("common.reset")}
              </button>
            )}
        </div>
      </form>
      <UnsavedChangesBar
        hasChanges={hasChanges}
        busy={busy}
        onRevert={() => {
          setData(stored);
          setClearedSecrets({});
          setTimeText(displayNotificationTime(stored.delivery_time, boot.preferences.time_format));
          setFeedback("");
        }}
        onSave={() => void save()}
        statusLabel={t("common.unsavedChanges", { defaultValue: "Unsaved changes" })}
        revertLabel={t("common.revertChanges", { defaultValue: "Revert changes" })}
        saveLabel={t("common.saveChanges", { defaultValue: "Save changes" })}
      />
    </section>
  );
}
