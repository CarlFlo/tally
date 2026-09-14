import { useEffect, useRef, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useLocation } from "react-router-dom";
import { MoreVertical, RefreshCw, RotateCcw, Trash2 } from "lucide-react";
import { api, Confirm, useApp, type Show } from "./lib";

type Action = "remove" | "clear";
export function ShowActionsMenu({
  show,
  detail = false,
  onRemoved,
}: {
  show: Show;
  detail?: boolean;
  onRemoved?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [confirm, setConfirm] = useState<Action | null>(null);
  const [busy, setBusy] = useState(false);
  const root = useRef<HTMLDivElement>(null),
    trigger = useRef<HTMLButtonElement>(null);
  const { notify } = useApp();
  const cache = useQueryClient(),
    location = useLocation();
  useEffect(() => {
    setOpen(false);
    setConfirm(null);
  }, [location.key]);
  useEffect(() => {
    if (!open) return;
    root.current
      ?.querySelector<HTMLButtonElement>('[role="menuitem"]')
      ?.focus();
    const outside = (event: PointerEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("pointerdown", outside);
    return () => document.removeEventListener("pointerdown", outside);
  }, [open]);
  function close() {
    setOpen(false);
    trigger.current?.focus();
  }
  async function perform(action: Action) {
    setBusy(true);
    try {
      await api(
        `/shows/${show.id}${action === "clear" ? "/watch-history" : ""}`,
        "DELETE",
      );
      // A committed deletion still invalidates shared data, but must not
      // navigate away from the user's new page after this menu unmounts.
      if (action === "remove" && root.current?.isConnected) onRemoved?.();
      await Promise.all(
        ["shows", "show", "calendar", "logs", "show-actions"].map((key) =>
          cache.invalidateQueries({ queryKey: [key] }),
        ),
      );
      notify(
        action === "clear"
          ? `Watch history cleared for ${show.name}`
          : `${show.name} removed`,
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <div
      className="show-actions-menu"
      ref={root}
      onClick={(e) => e.stopPropagation()}
    >
      <button
        ref={trigger}
        className="icon-button"
        aria-label={`Show actions: ${show.name}`}
        aria-haspopup="menu"
        aria-expanded={open}
        disabled={busy}
        onClick={() => setOpen(!open)}
      >
        <MoreVertical size={17} />
      </button>
      {open && (
        <div
          role="menu"
          aria-label={`Actions for ${show.name}`}
          className="action-popover"
          onKeyDown={(e) => {
            const items = Array.from(
              root.current!.querySelectorAll<HTMLButtonElement>(
                '[role="menuitem"]',
              ),
            );
            const index = items.indexOf(
              document.activeElement as HTMLButtonElement,
            );
            if (e.key === "Escape") {
              e.preventDefault();
              close();
            }
            if (e.key === "Tab") {
              setOpen(false);
            }
            if (["ArrowDown", "ArrowUp", "Home", "End"].includes(e.key)) {
              e.preventDefault();
              items[
                e.key === "Home"
                  ? 0
                  : e.key === "End"
                    ? items.length - 1
                    : (index + (e.key === "ArrowUp" ? -1 : 1) + items.length) %
                      items.length
              ]?.focus();
            }
          }}
        >
          <button
            role="menuitem"
            onClick={() => {
              close();
              setConfirm("clear");
            }}
          >
            <RotateCcw size={16} />
            Clear watch history
          </button>
          {detail && (
            <button
              role="menuitem"
              onClick={async () => {
                close();
                setBusy(true);
                try {
                  await api(`/shows/${show.id}/refresh`, "POST", {});
                  notify("Metadata refresh queued");
                } catch (e) {
                  notify((e as Error).message, true);
                } finally {
                  setBusy(false);
                }
              }}
            >
              <RefreshCw size={16} />
              Refresh metadata
            </button>
          )}
          <button
            role="menuitem"
            className="danger-text"
            title="Hold Shift to remove without confirmation"
            onClick={(e) => {
              close();
              if (e.shiftKey)
                void perform("remove").catch((error) =>
                  notify(error.message, true),
                );
              else setConfirm("remove");
            }}
          >
            <Trash2 size={16} />
            Remove show
          </button>
        </div>
      )}
      {confirm && (
        <Confirm
          title={
            confirm === "clear"
              ? `Clear watch history for ${show.name}?`
              : `Remove ${show.name}?`
          }
          message={
            confirm === "clear"
              ? "All episodes of this show will become unwatched for your profile. Downloaded markers are kept."
              : "This show will leave your library and calendar. Your progress is kept if you follow it again. Hold Shift when removing to skip this prompt."
          }
          onClose={() => {
            setConfirm(null);
            trigger.current?.focus();
          }}
          onConfirm={() => perform(confirm)}
        />
      )}
    </div>
  );
}
