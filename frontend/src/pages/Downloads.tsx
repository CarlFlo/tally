import { useQuery, useQueryClient } from "@tanstack/react-query";
import {
  Activity,
  ArrowDown,
  ArrowUp,
  Download,
  Pause,
  Play,
  Trash2,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { api, Busy, Dialog, Empty, ErrorState, bytes, useApp } from "../lib";
import { queryKeys } from "../queryKeys";

type DownloadItem = {
  hash: string;
  name: string;
  state: string;
  progress: number;
  size: number;
  downloaded: number;
  download_speed: number;
  upload_speed: number;
  ratio: number;
  added_on: number;
  category: string;
};

type DownloadData = {
  torrents: DownloadItem[];
  stats: {
    total: number;
    active: number;
    download_speed: number;
    upload_speed: number;
  };
};

function stopped(state: string) {
  const normalized = state.toLowerCase();
  return normalized.includes("stopped") || normalized.includes("paused");
}

export function DownloadsPage() {
  const { t } = useTranslation();
  const { notify } = useApp();
  const cache = useQueryClient();
  const [busy, setBusy] = useState<string | null>(null);
  const [removing, setRemoving] = useState<DownloadItem | null>(null);
  const downloads = useQuery<DownloadData>({
    queryKey: queryKeys.downloads(),
    queryFn: ({ signal }) =>
      api<DownloadData>("/torrents/downloads", "GET", undefined, signal),
    refetchInterval: 5000,
    refetchIntervalInBackground: false,
  });

  async function act(item: DownloadItem, action: "stop" | "start") {
    const key = item.hash + ":" + action;
    setBusy(key);
    try {
      await api("/torrents/downloads/" + item.hash + "/" + action, "POST");
      cache.setQueryData<DownloadData>(queryKeys.downloads(), (current) =>
        current
          ? {
              ...current,
              torrents: current.torrents.map((torrent) =>
                torrent.hash === item.hash
                  ? {
                      ...torrent,
                      state: action === "stop" ? "paused" : "downloading",
                    }
                  : torrent,
              ),
            }
          : current,
      );
      notify(t(action === "stop" ? "downloads.paused" : "downloads.resumed"));
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(null);
    }
  }

  async function remove(item: DownloadItem, deleteFiles: boolean) {
    const key = item.hash + ":remove:" + String(deleteFiles);
    setBusy(key);
    try {
      await api(
        "/torrents/downloads/" +
          item.hash +
          "?delete_files=" +
          String(deleteFiles),
        "DELETE",
      );
      cache.setQueryData<DownloadData>(queryKeys.downloads(), (current) =>
        current
          ? {
              torrents: current.torrents.filter(
                (torrent) => torrent.hash !== item.hash,
              ),
              stats: {
                ...current.stats,
                total: Math.max(0, current.stats.total - 1),
                active: Math.max(
                  0,
                  current.stats.active -
                    (item.download_speed > 0 || item.upload_speed > 0 ? 1 : 0),
                ),
                download_speed: Math.max(
                  0,
                  current.stats.download_speed - item.download_speed,
                ),
                upload_speed: Math.max(
                  0,
                  current.stats.upload_speed - item.upload_speed,
                ),
              },
            }
          : current,
      );
      setRemoving(null);
      notify(
        t(deleteFiles ? "downloads.removedWithFiles" : "downloads.removed"),
      );
    } catch (e) {
      notify((e as Error).message, true);
    } finally {
      setBusy(null);
    }
  }

  const data = downloads.data;
  return (
    <div className="page">
      <div className="page-heading">
        <div>
          <span className="eyebrow">{t("downloads.eyebrow")}</span>
          <h1>
            {t("downloads.title")}<span className="accent">.</span>
          </h1>
          <p>{t("downloads.description")}</p>
        </div>
      </div>

      {downloads.error && (
        <ErrorState error={downloads.error} retry={() => downloads.refetch()} />
      )}

      {data && (
        <>
          <div className="stats-grid downloads-stats">
            {[
              {
                label: t("downloads.total"),
                value: data.stats.total.toLocaleString(),
                icon: <Download size={20} />,
                cls: "purple",
              },
              {
                label: t("downloads.active"),
                value: data.stats.active.toLocaleString(),
                icon: <Activity size={20} />,
                cls: "mint",
              },
              {
                label: t("downloads.downloadSpeed"),
                value: bytes(data.stats.download_speed) + "/s",
                icon: <ArrowDown size={20} />,
                cls: "amber",
              },
              {
                label: t("downloads.uploadSpeed"),
                value: bytes(data.stats.upload_speed) + "/s",
                icon: <ArrowUp size={20} />,
                cls: "rose",
              },
            ].map((metric) => (
              <div className="panel stat-card" key={metric.label}>
                <span className={"metric-icon " + metric.cls}>{metric.icon}</span>
                <strong>{metric.value}</strong>
                <span>{metric.label}</span>
              </div>
            ))}
          </div>

          <section className="panel downloads-panel">
            {!data.torrents.length ? (
              <Empty icon={<Download size={31} />} title={t("downloads.emptyTitle")}>
                {t("downloads.emptyHelp")}
              </Empty>
            ) : (
              <div className="downloads-list">
                {data.torrents.map((item) => {
                  const isStopped = stopped(item.state);
                  const progress = Math.max(0, Math.min(100, item.progress * 100));
                  const toggleAction = isStopped ? "start" : "stop";
                  const toggleKey = item.hash + ":" + toggleAction;
                  return (
                    <div className="download-row" key={item.hash}>
                      <div className="download-row-main">
                        <div className="download-row-heading">
                          <h3>{item.name}</h3>
                          <span className="badge">{item.state}</span>
                        </div>
                        <div
                          className="download-progress-track"
                          aria-label={t("downloads.progress", {
                            percent: Math.round(progress),
                          })}
                          role="progressbar"
                          aria-valuemin={0}
                          aria-valuemax={100}
                          aria-valuenow={Math.round(progress)}
                        >
                          <span style={{ width: progress + "%" }} />
                        </div>
                        <div className="download-meta">
                          <span>{Math.round(progress)}%</span>
                          <span>{bytes(item.downloaded)} / {bytes(item.size)}</span>
                          <span>↓ {bytes(item.download_speed)}/s</span>
                          <span>↑ {bytes(item.upload_speed)}/s</span>
                          <span>{t("downloads.ratio", { ratio: item.ratio.toFixed(2) })}</span>
                        </div>
                      </div>
                      <div className="download-actions">
                        <button
                          type="button"
                          className="button small"
                          disabled={busy !== null}
                          onClick={() => act(item, toggleAction)}
                        >
                          {busy === toggleKey ? (
                            <Busy />
                          ) : isStopped ? (
                            <Play size={16} />
                          ) : (
                            <Pause size={16} />
                          )}
                          {t(isStopped ? "downloads.resume" : "downloads.pause")}
                        </button>
                        <button
                          type="button"
                          className="button small danger"
                          disabled={busy !== null}
                          title={t("downloads.removeHelp")}
                          onClick={() => setRemoving(item)}
                        >
                          <Trash2 size={16} />
                          {t("downloads.remove")}
                        </button>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </section>
          <p className="search-footnote">{t("downloads.pollingNote")}</p>
        </>
      )}
      {removing && (
        <Dialog
          title={t("downloads.removeTitle")}
          onClose={() => {
            if (!busy) setRemoving(null);
          }}
        >
          <p className="muted">
            {t("downloads.removePrompt", { name: removing.name })}
          </p>
          <div className="dialog-actions">
            <button
              type="button"
              className="button"
              disabled={busy !== null}
              onClick={() => setRemoving(null)}
            >
              {t("common.cancel")}
            </button>
            <button
              type="button"
              className="button danger"
              disabled={busy !== null}
              onClick={() => void remove(removing, false)}
            >
              {busy === removing.hash + ":remove:false" && <Busy />}
              {t("downloads.removeTorrentOnly")}
            </button>
            <button
              type="button"
              className="button danger"
              disabled={busy !== null}
              onClick={() => void remove(removing, true)}
            >
              {busy === removing.hash + ":remove:true" && <Busy />}
              {t("downloads.removeTorrentAndFiles")}
            </button>
          </div>
        </Dialog>
      )}
      {downloads.isPending && !data && <Busy />}
    </div>
  );
}
