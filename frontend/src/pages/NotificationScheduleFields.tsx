import type { NotificationErrors } from "./notificationValidation";

export function NotificationScheduleFields({
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
      <div className="notification-schedule">
        <label className={errors.delivery_time ? "field-invalid" : ""}>
          Daily release notification time
          <input
            value={timeText}
            placeholder={timeFormat === "12h" ? "9:00 AM" : "09:00"}
            aria-invalid={!!errors.delivery_time}
            onChange={(e) => changeTime(e.target.value)}
          />
          {errors.delivery_time && (
            <small className="field-error">{errors.delivery_time}</small>
          )}
        </label>
        <label className={errors.timezone ? "field-invalid" : ""}>
          Notification timezone
          <input
            list="notification-timezones"
            value={data.timezone}
            aria-invalid={!!errors.timezone}
            onChange={(e) => change("timezone", e.target.value)}
          />
          {errors.timezone && (
            <small className="field-error">{errors.timezone}</small>
          )}
          <datalist id="notification-timezones">
            {["UTC", ...Intl.supportedValuesOf("timeZone")].map((zone) => (
              <option key={zone} value={zone} />
            ))}
          </datalist>
        </label>
      </div>
      <p className="muted small-text">
        New episodes with an announced release time are sent at the next daily
        delivery time. System errors and other selected activity arrive
        immediately. Only shows followed on this server are included.
      </p>
    </>
  );
}
