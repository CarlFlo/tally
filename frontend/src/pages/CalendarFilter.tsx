import { Check, ChevronDown, ListFilter } from "lucide-react";
import { useEffect, useRef, useState } from "react";

const options = [
  ["all", "All episodes"],
  ["unwatched", "Unwatched"],
  ["watched", "Watched"],
] as const;

export function CalendarFilter({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const selected =
    options.find(([key]) => key === value)?.[1] || "All episodes";

  useEffect(() => {
    function close(event: MouseEvent) {
      if (!ref.current?.contains(event.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", close);
    return () => document.removeEventListener("mousedown", close);
  }, []);

  return (
    <div className="calendar-filter-menu" ref={ref}>
      <button
        type="button"
        className="calendar-filter-trigger"
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen(!open)}
        onKeyDown={(event) => event.key === "Escape" && setOpen(false)}
      >
        <ListFilter size={15} />
        {selected}
        <ChevronDown size={14} />
      </button>
      {open && (
        <div
          className="calendar-filter-options"
          role="listbox"
          aria-label="Filter episodes"
        >
          {options.map(([key, label]) => (
            <button
              key={key}
              type="button"
              role="option"
              aria-selected={value === key}
              onClick={() => {
                onChange(key);
                setOpen(false);
              }}
            >
              <span>{label}</span>
              {value === key && <Check size={15} />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
