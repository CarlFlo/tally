import { dateTimeFormatter } from "./dateFormatting";

export const jobName = (key: string) =>
  key === "metadata"
    ? "Metadata sync"
    : key === "backup"
      ? "Automatic backup"
      : "Maintenance";

export const jobDescription = (key: string) =>
  key === "metadata"
    ? "Checks due shows for new episodes and updated details."
    : key === "backup"
      ? "Creates a verified copy of your database and saved settings."
      : "Removes expired cache entries, old sessions, and retained history.";

export const commonSchedules: Record<string, readonly [string, string][]> = {
  metadata: [
    ["Every 15 minutes", "*/15 * * * *"],
    ["Every hour", "0 * * * *"],
    ["Every 6 hours", "0 */6 * * *"],
    ["Once a day at 03:00", "0 3 * * *"],
  ],
  maintenance: [
    ["Every day at 03:30", "30 3 * * *"],
    ["Every Monday at 03:30", "30 3 * * 1"],
    ["First day of each month at 03:30", "30 3 1 * *"],
  ],
  backup: [
    ["Every day at 03:00", "0 3 * * *"],
    ["Every Sunday at 03:00", "0 3 * * 0"],
    ["Every 2 weeks", "0 3 */14 * *"],
    ["First day of each month at 03:00", "0 3 1 * *"],
  ],
};

export function scheduleRunLabel(
  value: number,
  timeFormat: string,
  timezone: string,
) {
  return dateTimeFormatter("en-GB", {
    timeZone: timezone,
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: timeFormat === "12h",
  }).format(new Date(value * 1000));
}
