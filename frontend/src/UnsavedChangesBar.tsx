import { Undo2, Save } from "lucide-react";
import { useEffect, useRef } from "react";
import { useBeforeUnload } from "react-router";

export function useUnsavedChangesWarning(
  hasChanges: boolean,
  busy: boolean,
  message: string,
) {
  const navigationApproved = useRef(false);
  useBeforeUnload((event) => {
    if (!hasChanges || busy || navigationApproved.current) return;
    event.preventDefault();
    event.returnValue = "";
  });
  useEffect(() => {
    if (!hasChanges || busy) return;
    const onClick = (event: MouseEvent) => {
      if (
        event.defaultPrevented || event.button !== 0 || event.metaKey ||
        event.ctrlKey || event.shiftKey || event.altKey ||
        !(event.target instanceof Element)
      ) return;
      const link = event.target.closest("a[href]") as HTMLAnchorElement | null;
      if (!link || (link.target && link.target !== "_self") || link.hasAttribute("download")) return;
      const destination = new URL(link.href, window.location.href);
      if (destination.href === window.location.href) return;
      if (!window.confirm(message)) {
        event.preventDefault();
        return;
      }
      navigationApproved.current = true;
      window.setTimeout(() => { navigationApproved.current = false; }, 0);
    };
    document.addEventListener("click", onClick, true);
    return () => document.removeEventListener("click", onClick, true);
  }, [busy, hasChanges, message]);
}

export function UnsavedChangesBar({
  hasChanges,
  busy,
  onRevert,
  onSave,
  statusLabel,
  revertLabel,
  saveLabel,
}: {
  hasChanges: boolean;
  busy: boolean;
  onRevert: () => void;
  onSave: () => void;
  statusLabel: string;
  revertLabel: string;
  saveLabel: string;
}) {
  return (
    <div className={`unsaved-changes-bar${hasChanges ? " has-unsaved-changes" : ""}`} aria-hidden={!hasChanges}>
      <span className="unsaved-changes-status">{statusLabel}</span>
      <button className="button ghost" type="button" disabled={busy || !hasChanges} onClick={onRevert}>
        <Undo2 size={16} />{revertLabel}
      </button>
      <button className="button primary" type="button" disabled={busy || !hasChanges} onClick={onSave}>
        {busy ? <span className="button-spinner" /> : <Save size={16} />}{saveLabel}
      </button>
    </div>
  );
}
