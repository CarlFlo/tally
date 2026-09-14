import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { api, useApp } from "../lib";

export function DebugSettings() {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const [enabled, setEnabled] = useState(boot.preferences.debug_mode);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    if (!saving) setEnabled(boot.preferences.debug_mode);
  }, [boot.preferences.debug_mode, saving]);
  return (
    <section className="panel settings-card">
      <h3>Debug previews</h3>
      <p className="muted">
        Show preview controls in Jobs and Add Show to check failure and paused
        states. Previews leave real jobs and library data alone.
      </p>
      <label className="toggle-setting">
        <input
          type="checkbox"
          checked={enabled}
          disabled={saving}
          onChange={async (event) => {
            const value = event.target.checked;
            setEnabled(value);
            setSaving(true);
            try {
              await api("/preferences", "PATCH", { debug_mode: value });
              await cache.invalidateQueries({ queryKey: ["bootstrap"] });
            } catch (error) {
              setEnabled(!value);
              notify((error as Error).message, true);
            } finally {
              setSaving(false);
            }
          }}
        />
        Enable debug mode
      </label>
    </section>
  );
}
