import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "./queryKeys";
import { requestPool, RequestPoolOverloadError } from "./requestPool";

async function localeRequest<T>(path: string, signal?: AbortSignal): Promise<T> {
  const requestSignal = signal ?? new AbortController().signal;
  try {
    return await requestPool.run(requestSignal, async () => {
      const response = await fetch("/api" + path, {
        credentials: "same-origin",
        signal: requestSignal,
      });
      const data = await response.json();
      if (!response.ok)
        throw new Error(
          data.error ||
            i18n.t("errors.requestFailed", {
              status: response.status,
              defaultValue: `Request failed (${response.status})`,
            }),
        );
      return data as T;
    });
  } catch (error) {
    if (error instanceof RequestPoolOverloadError)
      throw new Error(
        i18n.t("errors.tooManyRequests", {
          defaultValue: "Too many pending requests. Please try again.",
        }),
      );
    throw error;
  }
}

export type LocaleStatus = {
  locale: string;
  name: string;
  direction?: "ltr" | "rtl";
  catalog_version?: number;
  valid: boolean;
  error?: string;
  error_code?: string;
};

type LocaleIndex = {
  revision: number;
  locales: LocaleStatus[];
};

type LocaleCatalog = {
  meta: {
    locale: string;
    name: string;
    direction: "ltr" | "rtl";
    catalogVersion: number;
  };
  messages: Record<string, unknown>;
  revision: number;
};

void i18n.use(initReactI18next).init({
  lng: "en",
  fallbackLng: "en",
  resources: {},
  interpolation: { escapeValue: false },
  returnNull: false,
  react: { useSuspense: false },
});

type LocaleContextValue = {
  locales: LocaleStatus[];
  requestedLocale: string;
  activeLocale: string;
  previewLocale: (locale: string | null) => void;
  localeStatus: (locale: string) => LocaleStatus | undefined;
};

const LocaleContext = createContext<LocaleContextValue>({
  locales: [],
  requestedLocale: "en",
  activeLocale: "en",
  previewLocale: () => {},
  localeStatus: () => undefined,
});

function replaceResourceBundle(
  locale: string,
  messages: Record<string, unknown>,
) {
  if (i18n.hasResourceBundle(locale, "translation"))
    i18n.removeResourceBundle(locale, "translation");
  i18n.addResourceBundle(locale, "translation", messages, true, true);
}

export function useLocalization() {
  return useContext(LocaleContext);
}

export function LocalizationProvider({
  profileLocale,
  children,
}: {
  profileLocale?: string | null;
  children: ReactNode;
}) {
  const [preview, setPreview] = useState<string | null>(null);
  const index = useQuery<LocaleIndex>({
    queryKey: queryKeys.locales(),
    queryFn: ({ signal }) => localeRequest("/locales", signal),
  });
  const statuses = index.data?.locales ?? [];
  const byCode = useMemo(
    () => new Map(statuses.map((status) => [status.locale, status])),
    [statuses],
  );
  const requestedLocale = preview || profileLocale || "en";
  const requested = byCode.get(requestedLocale);
  const candidateLocale = requested?.valid ? requestedLocale : "en";

  const english = useQuery<LocaleCatalog>({
    queryKey: queryKeys.localeCatalog("en", index.data?.revision),
    queryFn: ({ signal }) => localeRequest("/locales/en", signal),
    enabled: !!index.data,
  });
  const active = useQuery<LocaleCatalog>({
    queryKey: queryKeys.localeCatalog(candidateLocale, index.data?.revision),
    queryFn: ({ signal }) =>
      localeRequest("/locales/" + encodeURIComponent(candidateLocale), signal),
    enabled: !!index.data && candidateLocale !== "en",
  });
  const activeLocale =
    candidateLocale !== "en" && active.isError && !active.data
      ? "en"
      : candidateLocale;

  useLayoutEffect(() => {
    if (!english.data) return;
    replaceResourceBundle("en", english.data.messages);
  }, [english.data]);

  useLayoutEffect(() => {
    const catalog = activeLocale === "en" ? english.data : active.data;
    if (!catalog) return;
    replaceResourceBundle(catalog.meta.locale, catalog.messages);
    void i18n.changeLanguage(activeLocale);
    document.documentElement.lang = activeLocale;
    document.documentElement.dir = catalog.meta.direction || "ltr";
  }, [active.data, activeLocale, english.data]);

  useEffect(() => {
    // A saved profile change becomes authoritative once bootstrap refreshes.
    setPreview(null);
  }, [profileLocale]);

  const previewLocale = useCallback((locale: string | null) => {
    setPreview(locale);
  }, []);
  const localeStatus = useCallback(
    (locale: string) => byCode.get(locale),
    [byCode],
  );
  const value = useMemo(
    () => ({
      locales: statuses,
      requestedLocale,
      activeLocale,
      previewLocale,
      localeStatus,
    }),
    [
      activeLocale,
      localeStatus,
      previewLocale,
      requestedLocale,
      statuses,
    ],
  );

  // English must be available before user-facing copy is rendered. This also
  // guarantees a safe UI if another locale becomes unavailable.
  const criticalError =
    (!index.data && index.error) || (!english.data && english.error);
  if (criticalError) {
    const retry = () => {
      if (!index.data) void index.refetch();
      else void english.refetch();
    };
    return (
      <div className="startup" role="alert">
        <strong>
          {i18n.t("errors.localizationLoadFailed", {
            defaultValue: "Localization could not be loaded.",
          })}
        </strong>
        <button className="button" type="button" onClick={retry}>
          {i18n.t("common.retry", { defaultValue: "Retry" })}
        </button>
      </div>
    );
  }
  if (
    index.isPending ||
    english.isPending ||
    (preview === null && candidateLocale !== "en" && active.isPending)
  ) {
    return <div className="startup">Tally</div>;
  }

  return (
    <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
  );
}

export { i18n };
