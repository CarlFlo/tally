import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useState } from "react";
import { api, useApp, useLocal } from "../lib";
import { invalidateResources } from "../queryInvalidation";

export function DebugSettings() {
  const { boot, notify } = useApp();
  const cache = useQueryClient();
  const deployment = useLocal<any>("settings", "/settings");
  const [enabled, setEnabled] = useState(boot.preferences.debug_mode);
  const [saving, setSaving] = useState(false);
  useEffect(() => {
    if (!saving) setEnabled(boot.preferences.debug_mode);
  }, [boot.preferences.debug_mode, saving]);
  const environment = deployment.data?.environment || {};
  return (
    <div className="settings-card-list">
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
                await invalidateResources(cache, ["bootstrap"]);
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
      {deployment.data?.operator && (
        <section className="panel settings-card">
          <h3>Deployment environment</h3>
          <p className="muted">
            Effective non-sensitive environment values loaded when Tally
            started. Secrets and credential values are intentionally not shown.
          </p>
          <div className="environment-settings">
            {Object.entries(environment).map(([key, value]) => (
              <div className="setting-row" key={key}>
                <span><code>{key}</code></span>
                <strong>{String(value)}</strong>
              </div>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
