import { useEffect, useState, type FormEvent } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Bell, Save, Send } from "lucide-react";
import { api, Busy, ErrorState, useApp, useLocal } from "../lib";
import { NotificationFields } from "./NotificationFields";
import { useLatestRequest } from "../useLatestRequest";
import {
  displayNotificationTime,
  notificationErrors,
  parseNotificationTime,
} from "./notificationValidation";
const defaults = (data: any, timezone: string) => ({
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
  timezone: data.timezone || timezone,
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
  const startTest = useLatestRequest();
  const { boot, notify } = useApp(),
    cache = useQueryClient();
  const [data, setData] = useState(() =>
    defaults(saved.data, boot.preferences.timezone),
  );
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
  const errors = notificationErrors(normalized);
  const savedValid = Object.keys(notificationErrors(stored)).length === 0;
  const toggleMessage = savedValid
    ? ""
    : "Save valid notification service settings before enabling alerts.";
  useEffect(() => {
    setTimeText(
      displayNotificationTime(data.delivery_time, boot.preferences.time_format),
    );
  }, [boot.preferences.time_format]);
  useEffect(() => {
    if (saved.revision <= revision || JSON.stringify(data) !== JSON.stringify(stored))
      return;
    const next = defaults(saved.data, boot.preferences.timezone);
    setData(next);
    setStored(next);
    setRevision(saved.revision);
    setTimeText(
      displayNotificationTime(next.delivery_time, boot.preferences.time_format),
    );
  }, [boot.preferences.time_format, boot.preferences.timezone, data, revision, saved, stored]);
  const change = (key: string, value: any) =>
    setData((old: any) => ({ ...old, [key]: value }));
  async function persist(next: any) {
    const response = await api("/settings/notifications", "PUT", {
      data: next,
      revision,
    });
    setRevision(response.revision);
    setStored(next);
    cache.setQueryData(["editable-settings", "/settings/notifications"], {
      data: next,
      revision: response.revision,
    });
    await cache.invalidateQueries({ queryKey: ["settings"] });
  }
  async function toggle(enabled: boolean) {
    const previous = stored;
    setStored({ ...stored, enabled });
    setBusy(true);
    setFeedback("");
    try {
      await persist({ ...stored, enabled });
      change("enabled", enabled);
      notify(enabled ? "Notifications enabled" : "All notifications disabled");
    } catch (e) {
      setStored(previous);
      setFeedback((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  async function save(event: FormEvent) {
    event.preventDefault();
    if (Object.keys(errors).length) {
      return;
    }
    setBusy(true);
    setFeedback("");
    try {
      await persist(normalized);
      setData(normalized);
      notify("Notification settings saved");
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
            Notification Services
          </h3>
          <p className="muted">Choose where Tally sends your alerts.</p>
        </div>
        <span title={toggleMessage}>
          <label
            className={`toggle-setting notification-master${stored.enabled ? " is-enabled" : ""}`}
          >
            <input
              type="checkbox"
              role="switch"
              aria-label="Enable all notifications"
              checked={stored.enabled}
              disabled={busy || !savedValid}
              onChange={(e) => void toggle(e.target.checked)}
            />
            {stored.enabled ? "Alerts on" : "Alerts off"}
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
          />
        </fieldset>
        {feedback && (
          <p className="form-feedback" role="alert">
            {feedback}
          </p>
        )}
        <div className="notification-buttons">
          <button className="button primary" disabled={busy}>
            <Save size={17} />
            Save settings
          </button>
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
            Test Notification
          </button>
        </div>
      </form>
    </section>
  );
}
