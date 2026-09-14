import { ConnectionInput } from "../ConnectionInput";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { useLatestRequest } from "../useLatestRequest";
import { useQueryClient } from "@tanstack/react-query";
import { CheckCircle2, Plug, Save } from "lucide-react";
import { api, Busy, ErrorState, useApp, useLocal } from "../lib";
import { invalidateResources } from "../queryInvalidation";

type Field = {
  key: string;
  label: string;
  type: "text" | "url" | "password";
  required: boolean;
  secret: boolean;
  placeholder?: string;
  help?: string;
};
type Connection = {
  adapter: string;
  fields: Record<string, string>;
  secrets_configured: Record<string, boolean>;
  revision: number;
};
type ClientData = {
  adapters: { id: string; name: string; fields: Field[] }[];
  settings: Connection;
};

export function DownloaderSettings() {
  const query = useLocal<ClientData>(
    "downloader",
    "/downloader?reveal=1",
    true,
  );
  const [reloadKey, setReloadKey] = useState(0);
  return (
    <section className="panel settings-card client-settings">
      <h3>
        <Plug size={19} />
        Torrent client
      </h3>
      <p className="muted">
        Choose where to send the torrents you select. This connection is shared
        by all profiles.
      </p>
      {query.error && (
        <ErrorState error={query.error} retry={() => query.refetch()} />
      )}
      {query.isPending && <Busy />}
      {query.data && (
        <ClientForm
          key={reloadKey}
          data={query.data}
          reload={async () => {
            await query.refetch();
            setReloadKey((key) => key + 1);
          }}
        />
      )}
    </section>
  );
}

function ClientForm({
  data,
  reload,
}: {
  data: ClientData;
  reload: () => Promise<void>;
}) {
  const { notify } = useApp();
  const cache = useQueryClient();
  const form = useRef<HTMLFormElement>(null);
  const startTest = useLatestRequest();
  const [adapter, setAdapter] = useState(data.settings.adapter);
  const [fields, setFields] = useState<Record<string, string>>(
    data.settings.fields,
  );
  const [touched, setTouched] = useState<Record<string, boolean>>({});
  const [cleared, setCleared] = useState<Record<string, boolean>>({});
  const [revision, setRevision] = useState(data.settings.revision);
  const [changed, setChanged] = useState(false);
  const [busy, setBusy] = useState<"test" | "save" | null>(null);
  const [feedback, setFeedback] = useState<{
    message: string;
    error: boolean;
  } | null>(null);
  const definition = data.adapters.find((item) => item.id === adapter);
  useEffect(() => {
    if (changed || data.settings.revision <= revision) return;
    setAdapter(data.settings.adapter);
    setFields(data.settings.fields);
    setTouched({});
    setCleared({});
    setRevision(data.settings.revision);
  }, [changed, data, revision]);
  function change(key: string, value: string) {
    setChanged(true);
    setTouched((old) => ({ ...old, [key]: true }));
    setFields((old) => ({ ...old, [key]: value }));
    setFeedback(null);
  }
  function payload() {
    const values: Record<string, string> = {};
    for (const field of definition?.fields || []) {
      if (field.secret && !touched[field.key] && !cleared[field.key]) continue;
      if (field.secret && cleared[field.key]) values[field.key] = "";
      else if (!field.secret || fields[field.key])
        values[field.key] = fields[field.key] || "";
    }
    return { adapter, fields: values, revision };
  }
  async function run(action: "test" | "save", event?: FormEvent) {
    event?.preventDefault();
    if (!form.current?.reportValidity()) return;
    setBusy(action);
    setFeedback(null);
    const signal = action === "test" ? startTest() : undefined;
    try {
      if (action === "test") {
        const result = await api<{ message: string }>(
          "/downloader/test",
          "POST",
          payload(),
          signal,
        );
        setFeedback({
          message: result.message,
          error: false,
        });
      } else {
        const result = await api<Connection>("/downloader", "PUT", payload());
        setRevision(result.revision);
        setChanged(false);
        await invalidateResources(cache, [
          "settings",
          "downloader",
          "capabilities",
        ]);
        notify(adapter ? "Torrent client saved" : "Torrent client disabled");
      }
    } catch (error) {
      if (signal?.aborted) return;
      setFeedback({ message: (error as Error).message, error: true });
    } finally {
      setBusy(null);
    }
  }
  return (
    <form ref={form} onSubmit={(event) => run("save", event)}>
      <fieldset disabled={busy !== null} className="client-fields">
        <label>
          Torrent client
          <select
            value={adapter}
            onChange={(event) => {
              setChanged(true);
              setAdapter(event.target.value);
              setFields({});
              setCleared({});
              setFeedback(null);
            }}
          >
            <option value="">No torrent client</option>
            {data.adapters.map((client) => (
              <option key={client.id} value={client.id}>
                {client.name}
              </option>
            ))}
          </select>
        </label>
        {definition?.fields.map((field) => {
          const saved =
            adapter === data.settings.adapter &&
            data.settings.secrets_configured[field.key];
          return (
            <div key={field.key}>
              <label>
                {field.label}
                <ConnectionInput
                  type={field.type === "url" ? "url" : "text"}
                  secret={field.secret}
                  hiddenByDefault={field.secret}
                  label={field.label}
                  required={field.required && !(field.secret && saved)}
                  maxLength={4096}
                  value={fields[field.key] || ""}
                  disabled={field.secret && cleared[field.key]}
                  placeholder={
                    field.secret && saved
                      ? "Saved — leave blank to keep it"
                      : field.placeholder
                  }
                  aria-describedby={field.help ? `client-${field.key}-help` : undefined}
                  onChange={(event) => change(field.key, event.target.value)}
                />
              </label>
              {field.help && (
                <p
                  className="small-text muted client-field-help"
                  id={`client-${field.key}-help`}
                >
                  {field.help}
                </p>
              )}
              {field.secret && saved && !field.required && (
                <label className="client-clear-secret">
                  <input
                    type="checkbox"
                    checked={!!cleared[field.key]}
                    onChange={(event) => {
                      setChanged(true);
                      setCleared((old) => ({
                        ...old,
                        [field.key]: event.target.checked,
                      }));
                      setFeedback(null);
                    }}
                  />
                  Clear saved {field.label.toLowerCase()}
                </label>
              )}
            </div>
          );
        })}
        {!adapter && (
          <p className="muted small-text">
            You can still search and copy magnets. Sending becomes available
            after you configure a client.
          </p>
        )}
        <div className="client-actions">
          <button
            type="button"
            className="button"
            disabled={!adapter}
            onClick={() => run("test")}
          >
            {busy === "test" ? <Busy /> : <Plug size={17} />}Test connection
          </button>
          <button className="button primary" type="submit">
            {busy === "save" ? <Busy /> : <Save size={17} />}Save torrent client
          </button>
        </div>
      </fieldset>
      {feedback && (
        <div
          className={feedback.error ? "error-box" : "client-test-success"}
          role={feedback.error ? "alert" : "status"}
        >
          {!feedback.error && <CheckCircle2 size={18} />}
          <span>{feedback.message}</span>
        </div>
      )}
      {feedback?.error && (
        <button
          className="text-button"
          type="button"
          disabled={busy !== null}
          onClick={reload}
        >
          Reload saved settings
        </button>
      )}
    </form>
  );
}
