import { useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Plug } from "lucide-react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { ConnectionInput } from "../ConnectionInput";
import { useLatestRequest } from "../useLatestRequest";
import { api, Busy, ErrorState, useApp, useLocal, type Boot } from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { queryKeys } from "../queryKeys";
import { useTranslation } from "react-i18next";
import { UnsavedChangesBar, useUnsavedChangesWarning } from "../UnsavedChangesBar";
import "../unsaved-changes.css";

type JackettConfig = {
  base_url: string;
  api_key: string;
  enabled: boolean;
};

type SavedSearch = {
  data: JackettConfig;
  revision: number;
  secrets_configured?: Record<string, boolean>;
};

export function JackettSettings() {
  const { t } = useTranslation();
  const { notify } = useApp();
  const cache = useQueryClient();
  const query = useLocal<SavedSearch>("editable-settings", "/settings/search", true);
  const [toggleBusy, setToggleBusy] = useState(false);
  const [enabled, setEnabled] = useState<boolean | null>(null);
  useEffect(() => {
    if (query.data && !toggleBusy) setEnabled(query.data.data.enabled);
  }, [query.data, toggleBusy]);
  if (query.error) return <ErrorState error={query.error} retry={() => query.refetch()} />;
  if (!query.data) return <Busy />;
  const saved = query.data;
  const featureEnabled = enabled ?? saved.data.enabled;

  async function toggle(next: boolean) {
    const previous = featureEnabled;
    setEnabled(next);
    setToggleBusy(true);
    try {
      await api("/settings/search", "PUT", {
        data: { ...saved.data, enabled: next },
        revision: saved.revision,
      });
      cache.setQueryData<Boot>(queryKeys.bootstrap(), (current) =>
        current
          ? {
              ...current,
              jackett_enabled: next,
              torrent_search_enabled: next,
            }
          : current,
      );
      await invalidateResources(cache, [
        "settings",
        "editable-settings",
        "capabilities",
        "bootstrap",
      ]);
      notify(t(next ? "searchSettings.enabled" : "searchSettings.disabled"));
    } catch (error) {
      setEnabled(previous);
      notify((error as Error).message, true);
    } finally {
      setToggleBusy(false);
    }
  }

  return (
    <div className="integration-settings">
      <div className="section-heading settings-group-heading">
        <div>
          <h2>{t("settings.torrentSearch")}</h2>
        </div>
      </div>
      <section className="panel settings-card feature-toggle-setting">
        <label className="toggle-setting">
          <input
            type="checkbox"
            checked={featureEnabled}
            disabled={toggleBusy}
            onChange={(event) => void toggle(event.target.checked)}
          />
          {t("searchSettings.enable")}
        </label>
        <p className="muted small-text">{t("searchSettings.toggleHelp")}</p>
      </section>
      <JackettForm saved={saved} />
    </div>
  );
}

function JackettForm({ saved }: { saved: SavedSearch }) {
  const { t } = useTranslation();
  const { notify } = useApp();
  const cache = useQueryClient();
  const form = useRef<HTMLFormElement>(null);
  const startTest = useLatestRequest();
  const previous = useRef(saved);
  const [data, setData] = useState(saved.data);
  const [apiKeyConfigured, setApiKeyConfigured] = useState(
    !!saved.secrets_configured?.api_key,
  );
  const [revision, setRevision] = useState(saved.revision);
  const [busy, setBusy] = useState<"test" | "save" | null>(null);
  const [feedback, setFeedback] = useState<{ message: string; error: boolean } | null>(null);
  const hasChanges = JSON.stringify(data) !== JSON.stringify(previous.current.data);
  useUnsavedChangesWarning(hasChanges, busy !== null, t("common.unsavedNavigation", { defaultValue: "You have unsaved changes. Leave this page without saving?" }));
  useEffect(() => {
    if (JSON.stringify(data) === JSON.stringify(previous.current.data)) {
      setData(saved.data);
      setApiKeyConfigured(!!saved.secrets_configured?.api_key);
      setRevision(saved.revision);
    }
    previous.current = saved;
  }, [data, saved]);
  function change(next: Partial<JackettConfig>) {
    setData((current) => ({ ...current, ...next }));
    setFeedback(null);
  }
  async function run(action: "test" | "save", event?: FormEvent) {
    event?.preventDefault();
    if (!form.current?.reportValidity()) return;
    setBusy(action);
    setFeedback(null);
    const signal = action === "test" ? startTest() : undefined;
    try {
      if (action === "test") {
        const result = await api<{ message: string }>("/settings/search/test", "POST", { data }, signal);
        setFeedback({ message: result.message, error: false });
      } else {
        const result = await api<{ revision: number }>("/settings/search", "PUT", {
          data: { ...data, enabled: saved.data.enabled },
          revision,
        });
        const configured = apiKeyConfigured || data.api_key.trim() !== "";
        const redacted = { ...data, api_key: "" };
        setData(redacted);
        setApiKeyConfigured(configured);
        setRevision(result.revision);
        previous.current = {
          data: redacted,
          revision: result.revision,
          secrets_configured: { api_key: configured },
        };
        await invalidateResources(cache, [
          "settings",
          "editable-settings",
          "capabilities",
        ]);
        notify(t("searchSettings.saved"));
      }
    } catch (error) {
      if (signal?.aborted) return;
      setFeedback({ message: (error as Error).message, error: true });
    } finally {
      setBusy(null);
    }
  }
  return (
    <section className="panel settings-card client-settings">
      <h3><Plug size={19} />{t("searchSettings.jackett")}</h3>
      <p className="muted">{t("searchSettings.description")}</p>
      <form ref={form} onSubmit={(event) => run("save", event)} autoComplete="off">
        <fieldset disabled={busy !== null} className="client-fields">
          <div>
            <label>
              {t("searchSettings.baseURL")}
              <ConnectionInput label={t("searchSettings.baseURL")} type="url" value={data.base_url} required placeholder="http://jackett:9117" onChange={(event) => change({ base_url: event.target.value })} />
            </label>
            <p className="small-text muted client-field-help">{t("searchSettings.baseHelp")}</p>
          </div>
          <div>
            <label>
              {t("searchSettings.apiKey")}
              <ConnectionInput
                label={t("searchSettings.apiKey")}
                secret
                hiddenByDefault
                value={data.api_key}
                required={!apiKeyConfigured}
                maxLength={4096}
                placeholder={
                  apiKeyConfigured
                    ? t("connection.savedPlaceholder")
                    : undefined
                }
                onChange={(event) => change({ api_key: event.target.value })}
              />
            </label>
            <p className="small-text muted client-field-help">{t("searchSettings.keyHelp")}</p>
          </div>
          <div className="client-actions">
            <button type="button" className="button" onClick={() => run("test")}>
              {busy === "test" ? <Busy /> : <Plug size={17} />}{t("connection.test")}
            </button>
          </div>
        </fieldset>
        {feedback && <div className={feedback.error ? "error-box" : "client-test-success"} role={feedback.error ? "alert" : "status"}>
          {!feedback.error && <CheckCircle2 size={18} />}<span>{feedback.message}</span>
        </div>}
      </form>
      <UnsavedChangesBar
        hasChanges={hasChanges}
        busy={busy !== null}
        onRevert={() => {
          setData(previous.current.data);
          setApiKeyConfigured(!!previous.current.secrets_configured?.api_key);
          setFeedback(null);
        }}
        onSave={() => void run("save")}
        statusLabel={t("common.unsavedChanges", { defaultValue: "Unsaved changes" })}
        revertLabel={t("common.revertChanges", { defaultValue: "Revert changes" })}
        saveLabel={t("common.saveChanges", { defaultValue: "Save changes" })}
      />
    </section>
  );
}
