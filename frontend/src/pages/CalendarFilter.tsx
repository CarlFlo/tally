import { Check, ChevronDown, ListFilter } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

const options = ["all", "unwatched", "watched"] as const;

export function CalendarFilter({
  value,
  onChange,
}: {
  value: string;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);
  const selected = t(`calendar.${value === "all" ? "allEpisodes" : value}`);

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
          aria-label={t("calendar.filter")}
        >
          {options.map((key) => (
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
              <span>{t(`calendar.${key === "all" ? "allEpisodes" : key}`)}</span>
              {value === key && <Check size={15} />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
