import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useLocation } from "react-router-dom";

export function usePopover() {
  const [open, setOpen] = useState(false);
  const root = useRef<HTMLDivElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const location = useLocation();
  useLayoutEffect(() => setOpen(false), [location.key]);
  useEffect(() => {
    if (!open) return;
    function outside(event: PointerEvent) {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    }
    function focusOutside(event: FocusEvent) {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    }
    function escape(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
        trigger.current?.focus();
      }
    }
    document.addEventListener("pointerdown", outside);
    document.addEventListener("focusin", focusOutside);
    document.addEventListener("keydown", escape);
    return () => {
      document.removeEventListener("pointerdown", outside);
      document.removeEventListener("focusin", focusOutside);
      document.removeEventListener("keydown", escape);
    };
  }, [open]);
  return { open, setOpen, root, trigger };
}
