import { AlertTriangle, CheckCircle2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { bytes } from "./lib";

export type TorrentAssessmentReason = { code: string; detail?: string };

export type TorrentInspectionResult = {
  name: string;
  size: number;
  seeders: number;
  provider: string;
  uploader?: string;
  magnet: string;
  download_type: string;
  mb_per_minute?: number;
  confidence?: "high" | "medium" | "low" | "rejected";
  verification?: "verified" | "unverified";
  reasons?: TorrentAssessmentReason[];
  hard_rejections?: TorrentAssessmentReason[];
  previously_bad?: boolean;
  parsed?: {
    resolution?: string;
    source?: string;
    codec?: string;
    group?: string;
  };
  preferences?: {
    preferred_group?: boolean;
    preferred_uploader?: boolean;
    preferred_provider?: boolean;
  };
  size_profile?: {
    known?: boolean;
    mb_per_minute?: number;
    active_profile?: "live" | "animated";
    in_active_range?: boolean;
  };
};

const positiveReasonCodes = new Set([
  "video_payload",
  "no_blocked_payload",
]);

const concernReasonCodes = new Set([
  "ambiguous_identity",
  "unverifiable_magnet",
  "torrent_metadata_failed",
]);

function reasonLabel(code: string, detail: string | undefined, t: (key: string, options?: any) => string) {
  const key = `torrentConfidence.reason.${code}`;
  const fallback = code.replaceAll("_", " ");
  const label = t(key, { defaultValue: fallback });
  return detail ? `${label} · ${detail.replaceAll("_", " ")}` : label;
}

function evaluationSummary(result: TorrentInspectionResult, t: (key: string, options?: any) => string) {
  if (!result.confidence) {
    return t("torrentInspection.notEpisodeScored", {
      defaultValue: "No episode-specific confidence evaluation is available for this free-text result.",
    });
  }
  if (result.confidence === "rejected") {
    return t("torrentInspection.rejectedSummary", {
      defaultValue: "This candidate conflicts with the expected release and should not be selected automatically.",
    });
  }
  if (result.confidence === "high" && result.verification === "verified") {
    return t("torrentInspection.highVerifiedSummary", {
      defaultValue: "The release identity and torrent payload both match the expected episode.",
    });
  }
  if (result.confidence === "high") {
    return t("torrentInspection.highUnverifiedSummary", {
      defaultValue: "The release metadata strongly matches the expected episode, but the payload has not been inspected yet.",
    });
  }
  if (result.confidence === "medium") {
    return t("torrentInspection.mediumSummary", {
      defaultValue: "The candidate is plausible but has enough uncertainty to require review.",
    });
  }
  return t("torrentInspection.lowSummary", {
    defaultValue: "The available metadata is too weak or ambiguous for a confident match.",
  });
}

export function TorrentResultInspection({ result }: { result: TorrentInspectionResult }) {
  const { t } = useTranslation();
  const strengths: string[] = [];
  const concerns: string[] = [];

  if (result.size_profile?.known && typeof result.size_profile.mb_per_minute === "number") {
    const profile =
      result.size_profile.active_profile === "animated"
        ? t("torrentInspection.animatedProfile", { defaultValue: "animated" })
        : t("torrentInspection.liveProfile", { defaultValue: "live-action" });
    const value = result.size_profile.mb_per_minute.toFixed(1);
    if (result.size_profile.in_active_range) {
      strengths.push(
        t("torrentInspection.sizeWithinRange", {
          defaultValue: "{{value}} MB/min is within the configured {{profile}} range",
          value,
          profile,
        }),
      );
    } else {
      concerns.push(
        t("torrentInspection.sizeOutsideRange", {
          defaultValue: "{{value}} MB/min is outside the configured {{profile}} range",
          value,
          profile,
        }),
      );
    }
  }

  if (result.seeders > 0) {
    strengths.push(
      t("torrentInspection.seedersAvailable", {
        count: result.seeders,
        defaultValue: `${result.seeders} seeders available`,
      }),
    );
  } else {
    concerns.push(
      t("torrentInspection.noSeeders", {
        defaultValue: "No seeders are currently available",
      }),
    );
  }

  if (result.preferences?.preferred_uploader) {
    strengths.push(
      t("torrentInspection.preferredUploader", {
        defaultValue: "Uploader is on the preferred list",
      }),
    );
  }
  if (result.preferences?.preferred_provider) {
    strengths.push(
      t("torrentInspection.preferredProvider", {
        defaultValue: "Indexer/provider is on the preferred list",
      }),
    );
  }
  if (result.preferences?.preferred_group) {
    strengths.push(
      t("torrentInspection.preferredGroup", {
        defaultValue: "Release group is on the preferred list",
      }),
    );
  }
  if (result.verification === "verified") {
    strengths.push(
      t("torrentInspection.payloadVerified", {
        defaultValue: "Torrent payload has been inspected",
      }),
    );
  }

  for (const reason of result.reasons || []) {
    if (positiveReasonCodes.has(reason.code)) {
      strengths.push(reasonLabel(reason.code, reason.detail, t));
    } else if (concernReasonCodes.has(reason.code) && reason.code !== "unverifiable_magnet") {
      concerns.push(reasonLabel(reason.code, reason.detail, t));
    }
  }
  if (result.magnet && result.verification !== "verified") {
    concerns.push(
      t("torrentInspection.magnetUninspectable", {
        defaultValue: "Magnet payload cannot be inspected before download",
      }),
    );
  }
  if (result.previously_bad) {
    concerns.push(
      t("torrentInspection.previouslyBad", {
        defaultValue: "This infohash was previously marked bad",
      }),
    );
  }
  for (const reason of result.hard_rejections || []) {
    concerns.push(reasonLabel(reason.code, reason.detail, t));
  }

  const confidenceLabel = result.confidence
    ? t(`torrentConfidence.${result.confidence}`, {
        defaultValue: result.confidence.charAt(0).toUpperCase() + result.confidence.slice(1),
      })
    : t("torrentInspection.notEvaluated", { defaultValue: "Not episode-scored" });
  const verificationLabel = result.verification
    ? t(`torrentConfidence.${result.verification}`, {
        defaultValue: result.verification === "verified" ? "Verified" : "Unverified",
      })
    : "";

  return (
    <div className="torrent-result-inspection" onClick={(event) => event.stopPropagation()}>
      <div className="torrent-evaluation-head">
        <div>
          <h5>{t("torrentInspection.finalEvaluation", { defaultValue: "Final evaluation" })}</h5>
          <p className="torrent-evaluation-summary">{evaluationSummary(result, t)}</p>
        </div>
        <span className={`badge ${result.confidence ? `confidence-${result.confidence}` : ""}`}>
          {confidenceLabel}
          {verificationLabel && result.confidence !== "rejected" ? ` · ${verificationLabel}` : ""}
        </span>
      </div>

      <div className="torrent-evidence-grid">
        <section className="torrent-evidence-column strengths">
          <h5>
            <CheckCircle2 size={15} /> {t("torrentInspection.strengths", { defaultValue: "Strengths" })}
          </h5>
          {strengths.length ? (
            <ul>{strengths.map((item, index) => <li key={`${item}-${index}`}>{item}</li>)}</ul>
          ) : (
            <p className="torrent-evidence-empty">
              {t("torrentInspection.noStrengths", { defaultValue: "No strong positive signals yet" })}
            </p>
          )}
        </section>
        <section className="torrent-evidence-column concerns">
          <h5>
            <AlertTriangle size={15} /> {t("torrentInspection.concerns", { defaultValue: "Concerns" })}
          </h5>
          {concerns.length ? (
            <ul>{concerns.map((item, index) => <li key={`${item}-${index}`}>{item}</li>)}</ul>
          ) : (
            <p className="torrent-evidence-empty">
              {t("torrentInspection.noConcerns", { defaultValue: "No material concerns found in the available metadata" })}
            </p>
          )}
        </section>
      </div>

      <dl className="torrent-inspection-meta">
        <div><dt>{t("torrentInspection.quality", { defaultValue: "Quality" })}</dt><dd>{result.parsed?.resolution || "—"}</dd></div>
        <div><dt>{t("torrentInspection.source", { defaultValue: "Source" })}</dt><dd>{result.parsed?.source || "—"}</dd></div>
        <div><dt>{t("torrentInspection.codec", { defaultValue: "Codec" })}</dt><dd>{result.parsed?.codec || "—"}</dd></div>
        <div><dt>{t("torrentInspection.releaseGroup", { defaultValue: "Release group" })}</dt><dd>{result.parsed?.group || "—"}</dd></div>
        <div><dt>{t("torrentInspection.uploader", { defaultValue: "Uploader" })}</dt><dd>{result.uploader || "—"}</dd></div>
        <div><dt>{t("torrentInspection.provider", { defaultValue: "Provider" })}</dt><dd>{result.provider || "—"}</dd></div>
        <div><dt>{t("torrentInspection.size", { defaultValue: "Size" })}</dt><dd>{bytes(result.size)}</dd></div>
        {typeof result.mb_per_minute === "number" && (
          <div>
            <dt>{t("torrentInspection.mbPerMinute", { defaultValue: "MB/min" })}</dt>
            <dd>{result.mb_per_minute.toFixed(1)} MB/min</dd>
          </div>
        )}
        <div><dt>{t("torrentInspection.delivery", { defaultValue: "Delivery" })}</dt><dd>{result.download_type || "—"}</dd></div>
      </dl>
    </div>
  );
}
