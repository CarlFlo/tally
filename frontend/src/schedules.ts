import { dateTimeFormatter, displayLocale } from "./dateFormatting";
import { i18n } from "./i18n";

export const jobName = (key: string) =>
  key === "metadata"
    ? i18n.t("jobs.metadata")
    : key === "backup"
      ? i18n.t("jobs.automaticBackup")
      : i18n.t("jobs.maintenance");

export const jobDescription = (key: string) =>
  key === "metadata"
    ? i18n.t("jobs.metadataDescription")
    : key === "backup"
      ? i18n.t("jobs.backupDescription")
      : i18n.t("jobs.maintenanceDescription");

export const commonSchedules: Record<string, readonly [string, string][]> = {
  metadata: [
    ["schedule.every15", "*/15 * * * *"],
    ["schedule.everyHour", "0 * * * *"],
    ["schedule.every6", "0 */6 * * *"],
    ["schedule.daily0300", "0 3 * * *"],
  ],
  maintenance: [
    ["schedule.daily0330", "30 3 * * *"],
    ["schedule.monday0330", "30 3 * * 1"],
    ["schedule.firstMonth0330", "30 3 1 * *"],
  ],
  backup: [
    ["schedule.daily0300", "0 3 * * *"],
    ["schedule.sunday0300", "0 3 * * 0"],
    ["schedule.every2Weeks", "0 3 */14 * *"],
    ["schedule.firstMonth0300", "0 3 1 * *"],
  ],
};

export function scheduleRunLabel(
  value: number,
  timeFormat: string,
  timezone: string,
) {
  return dateTimeFormatter(displayLocale(i18n.resolvedLanguage), {
    timeZone: timezone,
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hour12: timeFormat === "12h",
  }).format(new Date(value * 1000));
}
