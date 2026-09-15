const tokens = new Set(["event", "key", "level", "message", "show", "time"]);

export type NotificationErrors = Record<string, string>;

export function notificationErrors(data: any): NotificationErrors {
  const errors: NotificationErrors = {};
  const endpoint = data.type === "discord" ? data.discord_url : data.url;
  const endpointField = data.type === "discord" ? "discord_url" : "url";
  if (!validURL(endpoint)) errors[endpointField] = "Enter a valid HTTP(S) URL.";
  if (data.type === "webhook") validateBody(data.body, errors);
  if ((data.bot_name || "").length > 80)
    errors.bot_name = "Use 80 characters or fewer.";
  if ((data.prefix || "").length > 500)
    errors.prefix = "Use 500 characters or fewer.";
  if (!validTime(data.delivery_time))
    errors.delivery_time = "Enter a valid notification time.";
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
    errors.body = "Use a JSON object of 16 KB or less.";
    return;
  }
  try {
    const parsed = JSON.parse(body);
    if (!parsed || Array.isArray(parsed) || typeof parsed !== "object")
      errors.body = "Enter a JSON object.";
  } catch {
    errors.body = "Enter valid JSON.";
    return;
  }
  for (const match of body.matchAll(/{{([^}]+)}}/g)) {
    if (!tokens.has(match[1]))
      errors.body = `{{${match[1]}}} is not a supported placeholder.`;
  }
}
