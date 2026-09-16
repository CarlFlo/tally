import { useState } from "react";
import { Moon, Sun, Monitor } from "lucide-react";
import { useQueryClient } from "@tanstack/react-query";
import { api, type Boot } from "./lib";
import { queryKeys } from "./queryKeys";
import { useTranslation } from "react-i18next";
export function LoginAppearance({ boot }: { boot: Boot }) {
  const { t } = useTranslation();
  const cache = useQueryClient(),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  return (
    <div className="login-appearance">
      <div
        role="group"
        aria-label={t("appearance.title")}
        data-selection={boot.browser_theme || "system"}
      >
        <span className="appearance-thumb" aria-hidden="true" />
        {(
          [
            { theme: "light", Icon: Sun },
            { theme: "dark", Icon: Moon },
            { theme: "system", Icon: Monitor },
          ] as const
        ).map(({ theme, Icon }) => (
          <button
            key={theme}
            className="icon-button"
            aria-label={`${t(`appearance.${theme}`)} ${t("appearance.title").toLowerCase()}`}
            title={`${t(`appearance.${theme}`)} ${t("appearance.title").toLowerCase()}`}
            aria-pressed={(boot.browser_theme || "system") === theme}
            disabled={busy}
            onClick={async () => {
              setBusy(true);
              setError("");
              try {
                await api("/browser/preferences", "PATCH", { theme });
                cache.setQueryData<Boot>(queryKeys.bootstrap(), (old) =>
                  old ? { ...old, browser_theme: theme } : old,
                );
              } catch (e) {
                setError((e as Error).message);
              } finally {
                setBusy(false);
              }
            }}
          >
            <Icon size={17} />
          </button>
        ))}
      </div>
      {error && <small role="alert">{error}</small>}
    </div>
  );
}
