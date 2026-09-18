import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Busy, Dialog, useApp } from "./lib";
import { jobName } from "./schedules";

export function ExperimentalJobConfirmation({
  jobKey,
  onConfirm,
  onClose,
}: {
  jobKey: string;
  onConfirm: () => Promise<boolean | void>;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const { notify } = useApp();
  const [busy, setBusy] = useState(false);
  const name = jobName(jobKey);

  return (
    <Dialog
      title={t("schedule.experimentalTitle", {
        defaultValue: "Enable experimental job?",
      })}
      onClose={onClose}
    >
      <p className="muted">
        {t("schedule.experimentalMessage", {
          defaultValue:
            "{{name}} is an experimental feature. It may change and can perform automatic actions in the background. Only enable it if you understand and accept that behavior.",
          name,
        })}
      </p>
      <div className="dialog-actions">
        <button type="button" className="button" disabled={busy} onClick={onClose}>
          {t("common.cancel")}
        </button>
        <button
          type="button"
          className="button primary"
          disabled={busy}
          onClick={async () => {
            setBusy(true);
            try {
              const confirmed = await onConfirm();
              if (confirmed !== false) onClose();
            } catch (error) {
              notify((error as Error).message, true);
            } finally {
              setBusy(false);
            }
          }}
        >
          {busy && <Busy />}
          {t("schedule.experimentalEnableAnyway", {
            defaultValue: "Yes, I understand, enable anyway",
          })}
        </button>
      </div>
    </Dialog>
  );
}
