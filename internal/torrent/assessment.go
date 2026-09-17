package torrent

import (
	"fmt"
	"strings"
)

type Confidence string

const (
	ConfidenceHigh     Confidence = "high"
	ConfidenceMedium   Confidence = "medium"
	ConfidenceLow      Confidence = "low"
	ConfidenceRejected Confidence = "rejected"
)

type VerificationState string

const (
	VerificationUnverified VerificationState = "unverified"
	VerificationVerified   VerificationState = "verified"
)

type AssessmentReasonCode string

const (
	ReasonShowMatch             AssessmentReasonCode = "show_match"
	ReasonEpisodeMatch          AssessmentReasonCode = "episode_match"
	ReasonHealthySwarm          AssessmentReasonCode = "healthy_swarm"
	ReasonExpectedSize          AssessmentReasonCode = "expected_size"
	ReasonVideoPayload          AssessmentReasonCode = "video_payload"
	ReasonNoBlockedPayload      AssessmentReasonCode = "no_blocked_payload"
	ReasonKnownBadInfoHash      AssessmentReasonCode = "known_bad_infohash"
	ReasonInfoHashMismatch      AssessmentReasonCode = "infohash_mismatch"
	ReasonWrongShow             AssessmentReasonCode = "wrong_show"
	ReasonWrongEpisode          AssessmentReasonCode = "wrong_episode"
	ReasonAmbiguousIdentity     AssessmentReasonCode = "ambiguous_identity"
	ReasonUnsupportedPack       AssessmentReasonCode = "unsupported_pack"
	ReasonMissingVideo          AssessmentReasonCode = "missing_video"
	ReasonBlockedPayload        AssessmentReasonCode = "blocked_payload"
	ReasonUnverifiableMagnet    AssessmentReasonCode = "unverifiable_magnet"
	ReasonTorrentMetadataFailed AssessmentReasonCode = "torrent_metadata_failed"
)

type AssessmentReason struct {
	Code   AssessmentReasonCode `json:"code"`
	Detail string               `json:"detail,omitempty"`
}

type ParsedRelease struct {
	Title        string `json:"title,omitempty"`
	Season       int    `json:"season,omitempty"`
	Episode      int    `json:"episode,omitempty"`
	Resolution   string `json:"resolution,omitempty"`
	Source       string `json:"source,omitempty"`
	Codec        string `json:"codec,omitempty"`
	Group        string `json:"group,omitempty"`
	MultiEpisode bool   `json:"multi_episode,omitempty"`
	SeasonPack   bool   `json:"season_pack,omitempty"`
}

type TorrentFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type PayloadInspection struct {
	Files           []TorrentFile `json:"files,omitempty"`
	VideoFiles      int           `json:"video_files"`
	SubtitleFiles   int           `json:"subtitle_files"`
	ArchiveFiles    int           `json:"archive_files"`
	ExecutableFiles int           `json:"executable_files"`
	TotalSize       int64         `json:"total_size"`
	MainVideoSize   int64         `json:"main_video_size"`
}

type ReleaseAssessment struct {
	Confidence     Confidence         `json:"confidence"`
	Verification   VerificationState  `json:"verification"`
	Reasons        []AssessmentReason `json:"reasons,omitempty"`
	HardRejections []AssessmentReason `json:"hard_rejections,omitempty"`
	Parsed         ParsedRelease      `json:"parsed"`
	Payload        *PayloadInspection `json:"payload,omitempty"`
	InfoHash       string             `json:"info_hash,omitempty"`

	// Score is intentionally internal. User-facing UI should present the
	// confidence band and reasons rather than a misleading percentage.
	Score int `json:"-"`
}

func (a ReleaseAssessment) Rejected() bool {
	return a.Confidence == ConfidenceRejected || len(a.HardRejections) > 0
}

func (a ReleaseAssessment) EligibleForAutomaticDownload() bool {
	return a.Confidence == ConfidenceHigh && a.Verification == VerificationVerified && len(a.HardRejections) == 0
}

func VerifyTorrentBytes(base ReleaseAssessment, candidate SearchResult, data []byte) (ReleaseAssessment, error) {
	metadata, err := ParseTorrentMetadata(data)
	if err != nil {
		base.Verification = VerificationUnverified
		base.Reasons = append(base.Reasons, AssessmentReason{Code: ReasonTorrentMetadataFailed})
		return base, fmt.Errorf("torrent metadata could not be inspected")
	}
	assessment := ApplyPayloadVerification(base, candidate, metadata)
	if !assessment.Rejected() {
		return assessment, nil
	}
	for _, reason := range assessment.HardRejections {
		switch reason.Code {
		case ReasonInfoHashMismatch:
			return assessment, fmt.Errorf("torrent metadata did not match the selected result")
		case ReasonBlockedPayload:
			return assessment, fmt.Errorf("torrent contains executable or script files")
		case ReasonMissingVideo:
			return assessment, fmt.Errorf("torrent contains no supported video files")
		}
	}
	return assessment, fmt.Errorf("torrent failed inspection")
}

func ApplyPayloadVerification(base ReleaseAssessment, candidate SearchResult, metadata TorrentMetadata) ReleaseAssessment {
	inspection := InspectTorrentPayload(metadata)
	base.Verification = VerificationVerified
	base.Payload = &inspection
	if metadata.InfoHashV1 != "" {
		base.InfoHash = metadata.InfoHashV1
	} else {
		base.InfoHash = metadata.InfoHashV2
	}

	if candidate.InfoHash != "" && metadata.InfoHashV1 != "" && !strings.EqualFold(candidate.InfoHash, metadata.InfoHashV1) {
		base.HardRejections = append(base.HardRejections, AssessmentReason{Code: ReasonInfoHashMismatch})
	}
	if inspection.VideoFiles == 0 {
		base.HardRejections = append(base.HardRejections, AssessmentReason{Code: ReasonMissingVideo})
	} else {
		base.Reasons = append(base.Reasons, AssessmentReason{Code: ReasonVideoPayload})
	}
	if inspection.ExecutableFiles > 0 {
		base.HardRejections = append(base.HardRejections, AssessmentReason{Code: ReasonBlockedPayload, Detail: "executable_file"})
	} else {
		base.Reasons = append(base.Reasons, AssessmentReason{Code: ReasonNoBlockedPayload})
	}
	if len(base.HardRejections) > 0 {
		base.Confidence = ConfidenceRejected
	}
	return base
}
