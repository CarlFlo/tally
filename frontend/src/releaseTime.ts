import { useEffect, useState } from "react";
import type { Episode } from "./lib";
import { dateTimeFormatter } from "./dateFormatting";
export function useNow(interval = 1000) {
  const [now, setNow] = useState(Date.now());
  useEffect(() => {
    const timer = setInterval(() => setNow(Date.now()), interval);
    return () => clearInterval(timer);
  }, [interval]);
  return now;
}
export function released(ep: Episode, timezone: string, now = Date.now()) {
  const stamp = Date.parse(ep.airstamp);
  if (Number.isFinite(stamp)) return stamp <= now;
  const today = dateTimeFormatter("en-CA", { timeZone: timezone }).format(
    now,
  );
  return !!ep.airdate && ep.airdate < today;
}
export function countdown(ep: Episode, timezone: string, now: number) {
  const stamp = Date.parse(ep.airstamp);
  if (!Number.isFinite(stamp))
    return released(ep, timezone, now) ? "Available" : "Time TBA";
  const seconds = Math.max(0, Math.ceil((stamp - now) / 1000));
  if (!seconds) return "Available";
  const days = Math.floor(seconds / 86400),
    hours = Math.floor((seconds % 86400) / 3600),
    minutes = Math.floor((seconds % 3600) / 60);
  return days
    ? `in ${days}d ${hours}h ${minutes}m`
    : `in ${hours ? hours + "h " : ""}${minutes}m ${seconds % 60}s`;
}
