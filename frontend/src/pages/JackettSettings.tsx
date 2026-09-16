import { useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Plug, Save } from "lucide-react";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { ConnectionInput } from "../ConnectionInput";
import { useLatestRequest } from "../useLatestRequest";
import { api, Busy, ErrorState, useApp, useLocal, type Boot } from "../lib";
import { invalidateResources } from "../queryInvalidation";
import { queryKeys } from "../queryKeys";
import { useTranslation } from "react-i18next";

type JackettConfig = {
  base_url: string;
  api_key: string;
  enabled: boolean;
};

type SavedSearch = { data: JackettConfig; revision: number };

export function JackettSettings() {
  const query = useLocal<SavedSearch>("editable-settings", "/settings/search", true);
  if (query.error) return <ErrorState error={query.error} retry={() => query.refetch()} />;
  if (!query.data) return <Busy />;
  return <JackettForm saved={query.data} />;
}

function JackettForm({ saved }: { saved: SavedSearch }) {
  const { t } = useTranslation();
  const { notify } = useApp();
  const cache = useQueryClient();
  const form = useRef<HTMLFormElement>(null);
  const startTest = useLatestRequest();
  const previous = useRef(saved);
  const [data, setData] = useState(saved.data);
  const [revision, setRevision] = useState(saved.revision);
  const [busy, setBusy] = useState<"test" | "save" | null>(null);
  const [feedback, setFeedback] = useState<{ message: string; error: boolean } | null>(null);
  useEffect(() => {
    if (JSON.stringify(data) === JSON.stringify(previous.current.data)) {
      setData(saved.data);
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
    if ((action === "test" || data.enabled) && !form.current?.reportValidity()) return;
    setBusy(action);
    setFeedback(null);
    const signal = action === "test" ? startTest() : undefined;
    try {
      if (action === "test") {
        const result = await api<{ message: string }>("/settings/search/test", "POST", { data }, signal);
        setFeedback({ message: result.message, error: false });
      } else {
        const result = await api<{ revision: number }>("/settings/search", "PUT", { data, revision });
        setRevision(result.revision);
        cache.setQueryData<Boot>(queryKeys.bootstrap(), (current) =>
          current ? { ...current, jackett_enabled: data.enabled } : current,
        );
        await invalidateResources(cache, ["settings", "editable-settings", "capabilities", "bootstrap"]);
        notify(t(data.enabled ? "searchSettings.saved" : "searchSettings.disabled"));
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
          <label className="toggle-setting">
            <input type="checkbox" checked={data.enabled} onChange={(event) => change({ enabled: event.target.checked })} />
            {t("searchSettings.enable")}
          </label>
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
              <ConnectionInput label={t("searchSettings.apiKey")} secret hiddenByDefault value={data.api_key} required maxLength={4096} onChange={(event) => change({ api_key: event.target.value })} />
            </label>
            <p className="small-text muted client-field-help">{t("searchSettings.keyHelp")}</p>
          </div>
          <div className="client-actions">
            <button type="button" className="button" onClick={() => run("test")}>
              {busy === "test" ? <Busy /> : <Plug size={17} />}{t("connection.test")}
            </button>
            <button className="button primary" type="submit" formNoValidate={!data.enabled}>
              {busy === "save" ? <Busy /> : <Save size={17} />}{t("searchSettings.save")}
            </button>
          </div>
        </fieldset>
        {feedback && <div className={feedback.error ? "error-box" : "client-test-success"} role={feedback.error ? "alert" : "status"}>
          {!feedback.error && <CheckCircle2 size={18} />}<span>{feedback.message}</span>
        </div>}
      </form>
    </section>
  );
}
