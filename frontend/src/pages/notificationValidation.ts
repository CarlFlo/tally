import { i18n } from "../i18n";
const tokens = new Set(["event", "key", "level", "message", "show", "time"]);

export type NotificationErrors = Record<string, string>;

export function notificationErrors(data: any): NotificationErrors {
  const errors: NotificationErrors = {};
  const endpoint = data.type === "discord" ? data.discord_url : data.url;
  const endpointField = data.type === "discord" ? "discord_url" : "url";
  if (!validURL(endpoint)) errors[endpointField] = i18n.t("validation.validURL");
  if (data.type === "webhook") validateBody(data.body, errors);
  if ((data.bot_name || "").length > 80)
    errors.bot_name = i18n.t("validation.max80");
  if ((data.prefix || "").length > 500)
    errors.prefix = i18n.t("validation.max500");
  if (!validTime(data.delivery_time))
    errors.delivery_time = i18n.t("validation.validNotificationTime");
  return errors;
}

export function displayNotificationTime(time: string, format: string) {
  if (!validTime(time) || format !== "12h") return time;
  const [hour, minute] = time.split(":").map(Number);
  return `${hour % 12 || 12}:${String(minute).padStart(2, "0")} ${hour < 12 ? "AM" : "PM"}`;
}

export function parseNotificationTime(value: string, format: string) {
  const text = value.trim();
  if (format !== "12h") return validTime(text) ? text : "";
  const match = text.match(/^(1[0-2]|0?[1-9]):([0-5]\d)\s*(AM|PM)$/i);
  if (!match) return "";
  let hour = Number(match[1]) % 12;
  if (match[3].toUpperCase() === "PM") hour += 12;
  return `${String(hour).padStart(2, "0")}:${match[2]}`;
}

function validURL(value: string) {
  try {
    const url = new URL(value);
    return (
      (url.protocol === "http:" || url.protocol === "https:") &&
      value.length <= 4096
    );
  } catch {
    return false;
  }
}

function validTime(value: string) {
  return /^([01]\d|2[0-3]):[0-5]\d$/.test(value || "");
}

function validateBody(body: string, errors: NotificationErrors) {
  if ((body || "").length > 16384) {
    errors.body = i18n.t("validation.json16k");
    return;
  }
  try {
    const parsed = JSON.parse(body);
    if (!parsed || Array.isArray(parsed) || typeof parsed !== "object")
      errors.body = i18n.t("validation.jsonObject");
  } catch {
    errors.body = i18n.t("validation.validJSON");
    return;
  }
  for (const match of body.matchAll(/{{([^}]+)}}/g)) {
    if (!tokens.has(match[1]))
      errors.body = i18n.t("validation.unsupportedPlaceholder", { token: `{{${match[1]}}}` });
  }
}
