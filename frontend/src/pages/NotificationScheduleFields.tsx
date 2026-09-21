import { useQuery } from "@tanstack/react-query";
import { api, useApp } from "../lib";
import { dateTimeFormatter, displayLocale } from "../dateFormatting";
import type { NotificationErrors } from "./notificationValidation";
import { parseNotificationTime } from "./notificationValidation";
import { queryKeys } from "../queryKeys";
import { useTranslation } from "react-i18next";

export function NotificationScheduleFields({
  data,
  errors,
  timeText,
  timeFormat,
  changeTime,
}: {
  data: any;
  errors: NotificationErrors;
  timeText: string;
  timeFormat: string;
  changeTime: (value: string) => void;
}) {
  const { t, i18n } = useTranslation();
  const { boot } = useApp();
  const serverTimezone = data.timezone || "UTC";
  const userTimezone = boot.preferences.timezone || "UTC";
  const deliveryTime = parseNotificationTime(timeText, timeFormat);
  const preview = useQuery<{ next_delivery: number; server_timezone: string }>({
    queryKey: queryKeys.notificationTimePreview(deliveryTime, serverTimezone),
    queryFn: ({ signal }) =>
      api(
        "/settings/notifications/preview",
        "POST",
        { delivery_time: deliveryTime },
        signal,
      ),
    enabled: !!deliveryTime,
    staleTime: 30_000,
  });
  const userTime =
    preview.data?.next_delivery && userTimezone !== serverTimezone
      ? dateTimeFormatter(displayLocale(i18n.resolvedLanguage), {
          timeZone: userTimezone,
          hour: "2-digit",
          minute: "2-digit",
          hour12: timeFormat === "12h",
        }).format(new Date(preview.data.next_delivery * 1000))
      : "";

  return (
    <>
      <div className="notification-schedule">
        <label className={errors.delivery_time ? "field-invalid" : ""}>
          <span className="notification-time-label">
            {t("notifications.releaseTime")}
            <span
              className="notification-timezone-chip"
              title={t("notifications.serverTimezone")}
            >
              {serverTimezone}
            </span>
          </span>
          <input
            aria-label={t("notifications.releaseTime")}
            value={timeText}
            placeholder={timeFormat === "12h" ? "9:00 AM" : "09:00"}
            aria-invalid={!!errors.delivery_time}
            onChange={(e) => changeTime(e.target.value)}
          />
          {errors.delivery_time && (
            <small className="field-error">{errors.delivery_time}</small>
          )}
          {!errors.delivery_time && userTime && (
            <small className="notification-local-time">
              {t("notifications.userTime", { time: userTime, timezone: userTimezone })}
            </small>
          )}
        </label>
      </div>
      <p className="muted small-text">
{t("notifications.scheduleHelp")}
      </p>
    </>
  );
}
