import { useLayoutEffect, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { AlertCircle, CheckCircle2, X } from "lucide-react";
export type Toast = { message: string; error: boolean; retry?: () => void };
export function Notice({
  toast,
  dismiss,
}: {
  toast: Toast;
  dismiss: () => void;
}) {
  const [host, setHost] = useState<HTMLElement>(document.body);
  const ref = useRef<HTMLDivElement>(null);
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
  // A modal makes everything outside it inert. Keep the notification inside the
  // active dialog, then use the popover top layer for its fixed screen position.
  return createPortal(
    <div
      ref={ref}
      popover="manual"
      className={"toast " + (toast.error ? "is-error" : "")}
      role={toast.error ? "alert" : "status"}
    >
      {toast.error ? <AlertCircle size={19} /> : <CheckCircle2 size={19} />}
      <span>{toast.message}</span>
      {toast.retry && (
        <button
          className="button small"
          onClick={() => {
            dismiss();
            toast.retry?.();
          }}
        >
          Retry
        </button>
      )}
      <button aria-label="Dismiss notification" onClick={dismiss}>
        <X size={16} />
      </button>
    </div>,
    host,
  );
}
