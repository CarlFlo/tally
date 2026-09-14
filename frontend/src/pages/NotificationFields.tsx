import { ConnectionInput } from "../ConnectionInput";
import type { ReactNode } from "react";
import type { NotificationErrors } from "./notificationValidation";
import { NotificationScheduleFields } from "./NotificationScheduleFields";

export const notificationEvents = [
  ["system_error", "System errors", "Immediately"],
  ["job_failed", "Failed jobs", "Immediately"],
  ["show_added", "Shows added", "Immediately"],
  ["show_removed", "Shows removed", "Immediately"],
  ["watch_history_cleared", "Watch history cleared", "Immediately"],
  ["episode_released", "New episode releases", "At your selected time"],
];

export function NotificationFields({
  data,
  change,
  errors,
  timeText,
  timeFormat,
  changeTime,
}: {
  data: any;
  change: (key: string, value: any) => void;
  errors: NotificationErrors;
  timeText: string;
  timeFormat: string;
  changeTime: (value: string) => void;
}) {
  return (
    <>
      <label>
        Service
        <select
          value={data.type}
          onChange={(e) => change("type", e.target.value)}
        >
          <option value="webhook">Webhook</option>
          <option value="discord">Discord</option>
        </select>
      </label>
      {data.type === "discord" ? (
        <DiscordFields data={data} change={change} errors={errors} />
      ) : (
        <WebhookFields data={data} change={change} errors={errors} />
      )}
      <fieldset className="notification-events">
        <legend>Subscribe to events</legend>
        {notificationEvents.map(([key, label, timing]) => (
          <label key={key} className="toggle-setting">
            <input
              type="checkbox"
              checked={data.events.includes(key)}
              onChange={(e) =>
                change(
                  "events",
                  e.target.checked
                    ? [...data.events, key]
                    : data.events.filter((x: string) => x !== key),
                )
              }
            />
            <span>
              {label}
              <small>{timing}</small>
            </span>
          </label>
        ))}
      </fieldset>
      <NotificationScheduleFields
        data={data}
        change={change}
        errors={errors}
        timeText={timeText}
        timeFormat={timeFormat}
        changeTime={changeTime}
      />
    </>
  );
}

function DiscordFields({ data, change, errors }: any) {
  return (
    <>
      <Field label="Discord webhook URL" error={errors.discord_url}>
        <ConnectionInput
          label="Discord webhook URL"
          aria-label="Discord webhook URL"
          secret
          type="url"
          value={data.discord_url}
          maxLength={4096}
          placeholder="https://discord.com/api/webhooks/..."
          aria-invalid={!!errors.discord_url}
          onChange={(e) => change("discord_url", e.target.value)}
        />
      </Field>
      <Field label="Bot display name" error={errors.bot_name}>
        <input
          value={data.bot_name}
          maxLength={80}
          placeholder="Tally"
          aria-invalid={!!errors.bot_name}
          onChange={(e) => change("bot_name", e.target.value)}
        />
      </Field>
      <Field label="Custom message prefix" error={errors.prefix}>
        <input
          value={data.prefix}
          maxLength={500}
          placeholder="TV update:"
          aria-invalid={!!errors.prefix}
          onChange={(e) => change("prefix", e.target.value)}
        />
      </Field>
    </>
  );
}

function WebhookFields({ data, change, errors }: any) {
  const preview = webhookPreview(data.body);
  return (
    <>
      <Field label="Webhook URL" error={errors.url}>
        <ConnectionInput
          label="Webhook URL"
          aria-label="Webhook URL"
          secret
          type="url"
          value={data.url}
          maxLength={4096}
          placeholder="https://your-service.example/webhook"
          aria-invalid={!!errors.url}
          onChange={(e) => change("url", e.target.value)}
        />
      </Field>
      <Field label="Payload body (JSON)" error={errors.body}>
        <textarea
          className="notification-body"
          value={data.body}
          maxLength={16384}
          rows={6}
          spellCheck={false}
          aria-invalid={!!errors.body}
          onChange={(e) => change("body", e.target.value)}
        />
      </Field>
      <p className="muted small-text">
        Use{" "}
        {"{{message}}, {{event}}, {{show}}, {{time}}, {{level}}, or {{key}}"} in
        JSON string values.
      </p>
      {preview && (
        <pre
          className="notification-preview"
          aria-label="Webhook payload preview"
        >
          {preview}
        </pre>
      )}
    </>
  );
}

function Field({
  label,
  error,
  children,
}: {
  label: string;
  error?: string;
  children: ReactNode;
}) {
  return (
    <label className={error ? "field-invalid" : ""}>
      {label}
      {children}
      {error && <small className="field-error">{error}</small>}
    </label>
  );
}

function webhookPreview(body: string) {
  try {
    const values: Record<string, string> = {
      message: "New episode available",
      event: "episode_released",
      show: "Breaking Bad",
      time: "2026-09-13T20:00:00Z",
      level: "info",
      key: "episode:breaking-bad:s05e14",
    };
    return JSON.stringify(
      JSON.parse(body),
      (_key, value) =>
        typeof value === "string"
          ? value.replace(
              /{{(message|event|show|time|level|key)}}/g,
              (_, token) => values[token],
            )
          : value,
      2,
    );
  } catch {
    return "";
  }
}
