import { ConnectionInput } from "../ConnectionInput";
import type { ReactNode } from "react";
import type { NotificationErrors } from "./notificationValidation";
import { NotificationScheduleFields } from "./NotificationScheduleFields";
import { useTranslation } from "react-i18next";
import { i18n } from "../i18n";

export const notificationEvents = [
  ["system_error", "notifications.systemErrors", "notifications.immediately"],
  ["job_failed", "notifications.failedJobs", "notifications.immediately"],
  ["show_added", "notifications.showsAdded", "notifications.immediately"],
  ["show_removed", "notifications.showsRemoved", "notifications.immediately"],
  ["watch_history_cleared", "notifications.historyCleared", "notifications.immediately"],
  ["profile_access_changed", "notifications.profileAccess", "notifications.immediately"],
  ["episode_released", "notifications.episodeReleased", "notifications.selectedTime"],
];

export function NotificationFields({
  data,
  change,
  errors,
  timeText,
  timeFormat,
  changeTime,
  secretsConfigured = {},
}: {
  data: any;
  change: (key: string, value: any) => void;
  errors: NotificationErrors;
  timeText: string;
  timeFormat: string;
  changeTime: (value: string) => void;
  secretsConfigured?: Record<string, boolean>;
}) {
  const { t } = useTranslation();
  return (
    <>
      <label>
        {t("notifications.service")}
        <select
          value={data.type}
          onChange={(e) => change("type", e.target.value)}
        >
          <option value="webhook">{t("notifications.webhook")}</option>
          <option value="discord">{t("notifications.discord")}</option>
        </select>
      </label>
      {data.type === "discord" ? (
        <DiscordFields data={data} change={change} errors={errors} secretsConfigured={secretsConfigured} />
      ) : (
        <WebhookFields data={data} change={change} errors={errors} secretsConfigured={secretsConfigured} />
      )}
      <fieldset className="notification-events">
        <legend>{t("notifications.subscribe")}</legend>
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
              {t(label)}
              <small>{t(timing)}</small>
            </span>
          </label>
        ))}
      </fieldset>
      <NotificationScheduleFields
        data={data}
        errors={errors}
        timeText={timeText}
        timeFormat={timeFormat}
        changeTime={changeTime}
      />
    </>
  );
}

function DiscordFields({ data, change, errors, secretsConfigured }: any) {
  const { t } = useTranslation();
  return (
    <>
      <Field label={t("notifications.discordURL")} error={errors.discord_url}>
        <ConnectionInput
          label={t("notifications.discordURL")}
          aria-label={t("notifications.discordURL")}
          secret
          type="url"
          value={data.discord_url}
          maxLength={4096}
          placeholder={
            secretsConfigured.discord_url
              ? t("connection.savedPlaceholder")
              : "https://discord.com/api/webhooks/..."
          }
          aria-invalid={!!errors.discord_url}
          onChange={(e) => change("discord_url", e.target.value)}
        />
      </Field>
      <Field label={t("notifications.botName")} error={errors.bot_name}>
        <input
          value={data.bot_name}
          maxLength={80}
          placeholder="Tally"
          aria-invalid={!!errors.bot_name}
          onChange={(e) => change("bot_name", e.target.value)}
        />
      </Field>
      <Field label={t("notifications.prefix")} error={errors.prefix}>
        <input
          value={data.prefix}
          maxLength={500}
          placeholder={t("notifications.prefixPlaceholder")}
          aria-invalid={!!errors.prefix}
          onChange={(e) => change("prefix", e.target.value)}
        />
      </Field>
    </>
  );
}

function WebhookFields({ data, change, errors, secretsConfigured }: any) {
  const { t } = useTranslation();
  const preview = webhookPreview(data.body);
  return (
    <>
      <Field label={t("notifications.webhookURL")} error={errors.url}>
        <ConnectionInput
          label={t("notifications.webhookURL")}
          aria-label={t("notifications.webhookURL")}
          secret
          type="url"
          value={data.url}
          maxLength={4096}
          placeholder={
            secretsConfigured.url
              ? t("connection.savedPlaceholder")
              : "https://your-service.example/webhook"
          }
          aria-invalid={!!errors.url}
          onChange={(e) => change("url", e.target.value)}
        />
      </Field>
      <Field label={t("notifications.payload")} error={errors.body}>
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
{t("notifications.placeholdersHelp", {
          tokens: "{{message}}, {{event}}, {{show}}, {{time}}, {{level}}, or {{key}}",
        })}
      </p>
      {preview && (
        <pre
          className="notification-preview"
          aria-label={t("notifications.preview")}
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
      message: i18n.t("notifications.previewMessage"),
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
