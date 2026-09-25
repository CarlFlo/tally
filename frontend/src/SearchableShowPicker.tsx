import { Check, ChevronDown, Search } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";

type ShowOption = { id: string; name: string };

export function SearchableShowPicker({ shows, value, onChange, disabled = false }: {
  shows: ShowOption[];
  value: string;
  onChange: (id: string) => void;
  disabled?: boolean;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [search, setSearch] = useState("");
  const root = useRef<HTMLDivElement>(null);
  const searchInput = useRef<HTMLInputElement>(null);
  const trigger = useRef<HTMLButtonElement>(null);
  const label = t("advancedFlows.assignShow");
  const selected = shows.find((show) => show.id === value);
  const matches = shows.filter((show) => show.name.toLocaleLowerCase().includes(search.trim().toLocaleLowerCase()));

  useEffect(() => {
    if (!open) return;
    searchInput.current?.focus();
    const closeOutside = (event: MouseEvent) => {
      if (!root.current?.contains(event.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", closeOutside);
    return () => document.removeEventListener("mousedown", closeOutside);
  }, [open]);

  function choose(id: string) {
    onChange(id);
    setOpen(false);
    setSearch("");
    trigger.current?.focus();
  }

  return <div className="advanced-show-picker" ref={root}>
    <span className="advanced-show-picker-label">{label}</span>
    <button ref={trigger} type="button" className="advanced-show-picker-trigger" aria-label={label} aria-haspopup="listbox" aria-expanded={open} disabled={disabled} onClick={() => setOpen((wasOpen) => !wasOpen)}>
      <span>{selected?.name || t("advancedFlows.chooseShow")}</span><ChevronDown size={16} />
    </button>
    {open && <div className="advanced-show-picker-menu" onKeyDown={(event) => { if (event.key === "Escape") { event.preventDefault(); setOpen(false); trigger.current?.focus(); } }}>
      <div className="advanced-show-picker-search"><Search size={16} /><input ref={searchInput} type="search" value={search} onChange={(event) => setSearch(event.target.value)} aria-label={t("advancedFlows.searchShows")} placeholder={t("advancedFlows.searchShows")} /></div>
      <div className="advanced-show-picker-options" role="listbox" aria-label={label}>
        {!search.trim() && <button type="button" role="option" aria-selected={!value} onClick={() => choose("")}>{t("advancedFlows.noShowSelected")}{!value && <Check size={15} />}</button>}
        {matches.map((show) => <button key={show.id} type="button" role="option" aria-selected={value === show.id} onClick={() => choose(show.id)}>{show.name}{value === show.id && <Check size={15} />}</button>)}
        {!matches.length && <p className="muted">{t("torrentAutomation.noShowMatches")}</p>}
      </div>
    </div>}
  </div>;
}
