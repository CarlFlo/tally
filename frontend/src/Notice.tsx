import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AlertCircle, CheckCircle2, X } from "lucide-react";
import { useTranslation } from "react-i18next";
export type Toast = { message: string; error: boolean; retry?: () => void };
export function Notice({
  toast,
  dismiss,
}: {
  toast: Toast;
  dismiss: () => void;
}) {
  const { t, i18n } = useTranslation();
  const [host, setHost] = useState<HTMLElement>(document.body);
  const [leaving, setLeaving] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const afterClose = useRef<(() => void) | null>(null);
  useLayoutEffect(() => {
    const sync = () => {
      const dialogs =
        document.querySelectorAll<HTMLDialogElement>("dialog[open]");
      setHost(dialogs.item(dialogs.length - 1) || document.body);
    };
    sync();
    const observer = new MutationObserver(sync);
    observer.observe(document.body, {
      subtree: true,
      childList: true,
      attributes: true,
      attributeFilter: ["open"],
    });
    return () => observer.disconnect();
  }, []);
  useLayoutEffect(() => {
    // A dialog can be removed in the same commit that changes the message.
    // The observer moves the portal to its next host after that commit.
    if (ref.current?.isConnected) ref.current.showPopover();
  }, [host, toast]);

  useEffect(() => {
    setLeaving(false);
    afterClose.current = null;
    if (toast.retry) return;
    const visibleFor = toast.error ? 4500 : 2200;
    const timer = window.setTimeout(() => setLeaving(true), visibleFor);
    return () => window.clearTimeout(timer);
  }, [toast]);

  useEffect(() => {
    if (!leaving) return;
    const timer = window.setTimeout(() => {
      dismiss();
      const action = afterClose.current;
      afterClose.current = null;
      action?.();
    }, 180);
    return () => window.clearTimeout(timer);
  }, [dismiss, leaving]);

  const close = (action?: () => void) => {
    if (leaving) return;
    afterClose.current = action || null;
    setLeaving(true);
  };

  // Profile save is the one success toast that can straddle a locale change.
  // If its text came from any loaded catalog, resolve the key again using the
  // locale that is active when the toast renders.
  const message =
    !toast.error &&
    Object.keys(i18n.store.data).some(
      (locale) =>
        i18n.getFixedT(locale)("settings.profileUpdated") === toast.message,
    )
      ? t("settings.profileUpdated")
      : toast.message;

  // A modal makes everything outside it inert. Keep the notification inside the
  // active dialog, then use the popover top layer for its fixed screen position.
  return createPortal(
    <div
      ref={ref}
      popover="manual"
      className={
        "toast " +
        (toast.error ? "is-error " : "") +
        (leaving ? "is-leaving" : "")
      }
      role={toast.error ? "alert" : "status"}
    >
      {toast.error ? <AlertCircle size={19} /> : <CheckCircle2 size={19} />}
      <span>{message}</span>
      {toast.retry && (
        <button
          className="button small"
          onClick={() => close(toast.retry)}
        >
          {t("common.retry")}
        </button>
      )}
      <button aria-label={t("inbox.dismiss")} onClick={() => close()}>
        <X size={16} />
      </button>
    </div>,
    host,
  );
}
